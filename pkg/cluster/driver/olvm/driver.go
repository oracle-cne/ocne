// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/kubepki"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"strings"
)

const (
	DriverName       = "olvm"
	ClusterNameLabel = "cluster.x-k8s.io/cluster-name"
	EnvUsername      = "OCNE_OLVM_USERNAME"
	EnvPassword      = "OCNE_OLVM_PASSWORD"
	EnvScope         = "OCNE_OLVM_SCOPE"
)

type OlvmDriver struct {
	Ephemeral            bool
	BootstrapKubeConfig  string
	KubeConfig           string
	Config               *types.Config
	ClusterConfig        *types.ClusterConfig
	ClusterResources     string
	PKIInfo              *kubepki.PKIInfo
	UploadCertificateKey string
	ResourceNamespace    string
	FromTemplate         bool
	Deleted              bool
}

// CreateDriver creates an OLVM CAPI driver.
func CreateDriver(clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
	var err error
	doTemplate := false
	cd := *clusterConfig.ClusterDefinition
	cdi := *clusterConfig.ClusterDefinitionInline
	if cd != "" && cdi != "" {
		// Can't mix inline and file-based resources
		return nil, fmt.Errorf("cluster configuration has file-based and inline resources")
	} else if cd == "" && cdi == "" {
		// If no configuration is provided, make one.  We may need to upload an
		// image.
		doTemplate = true

	} else if cd != "" {
		// If the path to the cluster definition is not
		// absolute, then assume it is relative to the
		// cluster config working directory.
		if !filepath.IsAbs(cd) {
			cd = filepath.Join(*clusterConfig.WorkingDirectory, cd)
			cd, err = filepath.Abs(cd)
			if err != nil {
				return nil, err
			}
		}
		cdiBytes, err := os.ReadFile(cd)
		if err != nil {
			return nil, err
		}
		cdi = string(cdiBytes)
	}

	// Unlike other cluster drivers, it is not feasible to have zero
	// worker nodes.  Cluster API will not create control plane nodes
	// with taints removed, and it can get upset if they are removed.
	// Require at least one.
	//
	// If someone really wants to have no workers, then they are free
	// to pass in a cluster definition.
	if *clusterConfig.WorkerNodes == 0 {
		*clusterConfig.WorkerNodes = 1
	}

	// It's also not feasible to have zero control plane nodes.
	if *clusterConfig.ControlPlaneNodes == 0 {
		*clusterConfig.ControlPlaneNodes = 1
	}

	cad := &OlvmDriver{
		ClusterConfig:    clusterConfig,
		ClusterResources: cdi,
		FromTemplate:     doTemplate,
	}

	bootstrapKubeConfig, isEphemeral, err := start.EnsureCluster(*clusterConfig.KubeConfig, clusterConfig)
	if err != nil {
		return nil, err
	}

	cad.Ephemeral = isEphemeral
	cad.BootstrapKubeConfig = bootstrapKubeConfig

	_, kubeClient, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return nil, err
	}

	// Install any necessary components into the admin cluster
	capiApplications, err := cad.getApplications()
	if err != nil {
		return nil, err
	}

	err = install.InstallApplications(capiApplications, cad.BootstrapKubeConfig, *clusterConfig.Quiet)
	if err != nil {
		return nil, err
	}

	// Wait for all controllers to come online.  This is done
	// as a separate step so that all the image pulls can happen
	// in parallel because the application installation is
	// linear
	err = cad.waitForControllers(kubeClient)
	if err != nil {
		return nil, err
	}

	cad.KubeConfig, err = client.GetKubeconfigPath(fmt.Sprintf("kubeconfig.%s", *cad.ClusterConfig.Name))
	if err != nil {
		return nil, err
	}

	return cad, nil
}

// waitForControllers waits for all the CAPI controllers to be ready.
func (cad *OlvmDriver) waitForControllers(kubeClient kubernetes.Interface) error {
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for Core Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.CoreCAPINamespace, constants.CoreCAPIDeployment, 1)
			},
		},
		{
			Message: "Waiting for Kubadm Boostrap Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.KubeadmBootstrapCAPINamespace, constants.KubeadmBootstrapCAPIDeployment, 1)
			},
		},
		{
			Message: "Waiting for Kubadm Control Plane Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.KubeadmControlPlaneCAPINamespace, constants.KubeadmControlPlaneCAPIDeployment, 1)
			},
		},
		{
			Message: "Waiting for Olvm Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.OLVMCAPIOperatorNamespace, constants.OLVMCAPIDeployment, 1)
			},
		},
	})
	if haveError {
		return fmt.Errorf("Not all Cluster API controllers became available")
	}
	return nil
}

// getClusterObject gets the CAPI cluster object that is being started or deleted.
func (cad *OlvmDriver) getClusterObject() (unstructured.Unstructured, error) {
	clusterObj, err := k8s.FindIn(cad.ClusterResources, func(u unstructured.Unstructured) bool {
		if u.GetKind() != "Cluster" {
			return false
		}
		if u.GetAPIVersion() != "cluster.x-k8s.io/v1beta1" {
			return false
		}
		_, ok := u.GetLabels()[ClusterNameLabel]
		return ok
	})
	if err != nil {
		if k8s.IsNotExist(err) {
			return unstructured.Unstructured{}, fmt.Errorf("Cluster resources do not include a valid cluster.x-k8s.io/v1beta1/Cluster")
		} else {
			return unstructured.Unstructured{}, err
		}
	}
	return clusterObj, err
}

// applyResources creates resources in a cluster if the resource does not
// already exist.  If the resource already exists, it is not modified.
func (cad *OlvmDriver) applyResources(restConfig *rest.Config) error {
	resources, err := k8s.Unmarshall(bufio.NewReader(bytes.NewBufferString(cad.ClusterResources)))
	if err != nil {
		return err
	}

	for _, r := range resources {
		err = k8s.CreateResourceIfNotExist(restConfig, &r)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}
	}

	return nil
}

func (cad *OlvmDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	return fmt.Errorf("Joining new nodes to this cluster is done by editing the KubeadmControlPlane and MachineDeployment resources in the management cluster")
}

func (cad *OlvmDriver) Stop() error {
	return fmt.Errorf("OlvmDriver.Stop() is not implemented")
}

func (cad *OlvmDriver) GetKubeconfigPath() string {
	return cad.KubeConfig
}

func (cad *OlvmDriver) GetKubeAPIServerAddress() string {
	return *cad.ClusterConfig.VirtualIp
}

func (cad *OlvmDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster:\n    use %s", cad.KubeConfig)
}

func (Cad *OlvmDriver) DefaultCNIInterfaces() []string {
	// let CNI pick the interface
	return nil
}

func (cad *OlvmDriver) Stage(version string) (string, string, bool, error) {
	return "", "", false, fmt.Errorf("Implement me")
}
