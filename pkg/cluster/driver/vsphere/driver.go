// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vsphere

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	capicommon "github.com/oracle-cne/ocne/pkg/cluster/driver/capi"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	vspheretemplate "github.com/oracle-cne/ocne/pkg/cluster/template/vsphere"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DriverName = "vsphere"

	VsphereCcmChart     = "vsphere-ccm"
	VsphereCcmNamespace = "kube-system"
	VsphereCcmRelease   = "vsphere-ccm"
	VsphereCcmVersion   = ""
)

// ClusterApiDriver implements a vSphere-backed Cluster API flow.
// It mirrors the OCI driver pattern but omits OCI-specific image/CCM logic.
type ClusterApiDriver struct {
	Ephemeral           bool
	BootstrapKubeConfig string
	KubeConfig          string
	Config              *types.Config
	ClusterConfig       *types.ClusterConfig
	ClusterResources    string
	ResourceNamespace   string
	FromTemplate        bool
	Deleted             bool
}

// getApplications installs core CAPI controllers and vSphere provider controllers into the bootstrap cluster.
func (cad *ClusterApiDriver) getApplications() ([]install.ApplicationDescription, error) {
	proxyValues := map[string]interface{}{}

	// We reuse internal catalog charts for core CAPI components.
	return []install.ApplicationDescription{
		{
			Application: &types.Application{
				Name:      constants.CertManagerChart,
				Namespace: constants.CertManagerNamespace,
				Release:   constants.CertManagerRelease,
				Version:   constants.CertManagerVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
		{
			Application: &types.Application{
				Name:      constants.CoreCAPIChart,
				Namespace: constants.CoreCAPINamespace,
				Release:   constants.CoreCAPIRelease,
				Version:   constants.CoreCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		{
			Application: &types.Application{
				Name:      constants.KubeadmBootstrapCAPIChart,
				Namespace: constants.KubeadmBootstrapCAPINamespace,
				Release:   constants.KubeadmBootstrapCAPIRelease,
				Version:   constants.KubeadmBootstrapCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		{
			Application: &types.Application{
				Name:      constants.KubeadmControlPlaneCAPIChart,
				Namespace: constants.KubeadmControlPlaneCAPINamespace,
				Release:   constants.KubeadmControlPlaneCAPIRelease,
				Version:   constants.KubeadmControlPlaneCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		// vSphere infrastructure provider chart (assumed to be in internal catalog)
		{
			Application: &types.Application{
				Name:      constants.VsphereCAPIChart,
				Namespace: constants.VsphereCAPINamespace,
				Release:   constants.VsphereCAPIRelease,
				Version:   constants.VsphereCAPIVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
	}, nil
}

// getWorkloadClusterApplications installs optional addons into the workload cluster (e.g., CCM if needed).
func (cad *ClusterApiDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	// For now, no default workload addons; extend if needed for CCM/CSI when packaged internally.
	return []install.ApplicationDescription{}, nil
}

func (cad *ClusterApiDriver) Start() (bool, bool, error) {
	var err error
	doTemplate := false
	cd := cad.ClusterConfig.ClusterDefinition
	cdi := cad.ClusterConfig.ClusterDefinitionInline
	if cd != "" && cdi != "" {
		return false, false, fmt.Errorf("cluster configuration has file-based and inline resources")
	} else if cd == "" && cdi == "" {
		doTemplate = true
	} else if cd != "" {
		if !filepath.IsAbs(cd) {
			cd = filepath.Join(cad.ClusterConfig.WorkingDirectory, cd)
			cd, err = filepath.Abs(cd)
			if err != nil {
				return false, false, err
			}
		}
		cdiBytes, err := os.ReadFile(cd)
		if err != nil {
			return false, false, err
		}
		cdi = string(cdiBytes)
	}

	if cad.ClusterConfig.WorkerNodes == 0 {
		cad.ClusterConfig.WorkerNodes = 1
	}
	if cad.ClusterConfig.ControlPlaneNodes == 0 {
		cad.ClusterConfig.ControlPlaneNodes = 1
	}

	// Validate provider basics here if needed; template helper also validates required fields

	cad.FromTemplate = doTemplate
	cad.ClusterResources = cdi

	// Create / ensure bootstrap (ephemeral) cluster
	bootstrapKubeConfig, isEphemeral, err := start.EnsureCluster(cad.Config.Providers.Vsphere.KubeConfigPath, cad.Config, cad.ClusterConfig)
	if err != nil {
		return false, false, err
	}

	cad.Ephemeral = isEphemeral
	cad.BootstrapKubeConfig = bootstrapKubeConfig

	// Install controllers into bootstrap cluster
	capiApps, err := cad.getApplications()
	if err != nil {
		return false, false, err
	}

	if log.GetLevel() != log.DebugLevel && log.GetLevel() != log.TraceLevel {
		rest.SetDefaultWarningHandler(rest.NoWarnings{})
	}
	err = install.InstallApplications(capiApps, cad.BootstrapKubeConfig, cad.Config.Quiet)
	if err != nil {
		return false, false, err
	}
	rest.SetDefaultWarningHandler(rest.WarningLogger{})

	_, kubeClient, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return false, false, err
	}

	// Wait for controllers
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for Core Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.CoreCAPINamespace, constants.CoreCAPIDeployment, 1)
			},
		},
		{
			Message: "Waiting for Kubadm Bootstrap Cluster API Controllers",
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
			Message: "Waiting for vSphere Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.VsphereCAPINamespace, constants.VsphereCAPIDeployment, 1)
			},
		},
	})
	if haveError {
		return false, false, fmt.Errorf("Not all Cluster API controllers became available")
	}

	cad.KubeConfig, err = client.GetKubeconfigPath(fmt.Sprintf("kubeconfig.%s", cad.ClusterConfig.Name))
	if err != nil {
		return false, false, err
	}

	// If using a template, render it now.
	if cad.FromTemplate {
		cdi, err := vspheretemplate.GetVsphereTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return false, false, err
		}
		cad.ClusterResources = cdi
	}

	// Fetch Cluster object to get namespace/name
	clusterObj, err := capicommon.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return false, false, err
	}
	cad.ResourceNamespace = clusterObj.GetNamespace()

	restConfig, clientIface, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return false, false, err
	}

	if err := k8s.CreateNamespaceIfNotExists(clientIface, cad.ResourceNamespace); err != nil {
		return false, false, err
	}

	log.Info("Applying Cluster API resources")
	if err := cad.applyResources(restConfig); err != nil {
		return false, false, err
	}

	// Wait for kubeconfig secret
	clusterName, _ := clusterObj.GetLabels()[capicommon.ClusterNameLabel]
	kubeconfig, err := cad.waitForKubeconfig(clientIface, clusterName)
	if err != nil {
		return false, false, err
	}
	if err := os.WriteFile(cad.KubeConfig, []byte(kubeconfig), 0700); err != nil {
		return false, false, err
	}

	_, workloadClient, err := client.GetKubeClient(cad.KubeConfig)
	if err != nil {
		return false, false, err
	}
	if _, err := k8s.WaitUntilGetNodesSucceeds(workloadClient); err != nil {
		return false, false, err
	}

	// Install workload addons (currently none by default)
	workloadApps, err := cad.getWorkloadClusterApplications(restConfig, workloadClient)
	if err != nil {
		return false, false, err
	}
	if len(workloadApps) > 0 {
		log.Info("Installing applications into workload cluster")
		if err := install.InstallApplications(workloadApps, cad.KubeConfig, cad.Config.Quiet); err != nil {
			return false, false, err
		}
	}

	return false, false, nil
}

// Helper functions largely lifted/adapted from OCI driver
func (cad *ClusterApiDriver) applyResources(restConfig *rest.Config) error {
	resources, err := k8s.Unmarshall(bufio.NewReader(bytes.NewBufferString(cad.ClusterResources)))
	if err != nil {
		return err
	}
	for _, r := range resources {
		if err := k8s.CreateResourceIfNotExist(restConfig, &r); err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}
	}
	return nil
}

func (cad *ClusterApiDriver) waitForKubeconfig(client kubernetes.Interface, clusterName string) (string, error) {
	var kubeconfig string
	kcfgSecretIface, _, err := util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		secrets, err := k8s.FindSecretsByLabelKeyVal(client, cad.ResourceNamespace, capicommon.ClusterNameLabel, clusterName)
		if err != nil {
			return nil, false, err
		}
		if len(secrets.Items) == 0 {
			return nil, false, fmt.Errorf("no kubeconfig found for cluster %s in namespace %s", clusterName, cad.ResourceNamespace)
		}
		for _, s := range secrets.Items {
			if strings.Contains(s.ObjectMeta.Name, "kubeconfig") {
				return &s, false, nil
			}
		}
		return nil, false, fmt.Errorf("no kubeconfig found for cluster %s in namespace %s", clusterName, cad.ResourceNamespace)
	}, nil, 20*time.Minute)
	if err != nil {
		return "", err
	}
	kcfgSecret, ok := kcfgSecretIface.(*v1.Secret)
	if !ok {
		return "", fmt.Errorf("internal error: kubeconfig secret is not a secret")
	}
	kcfgBytes, ok := kcfgSecret.Data["value"]
	if !ok {
		return "", fmt.Errorf("%s is not a valid kubeconfig secret", kcfgSecret.ObjectMeta.Name)
	}
	kubeconfig = string(kcfgBytes)
	return kubeconfig, nil
}

// Stop/Join/Delete/Close follow the CAPI driver pattern; some are intentionally unimplemented
func (cad *ClusterApiDriver) Stop() error {
	return fmt.Errorf("ClusterApiDriver.Stop() is not implemented")
}

func (cad *ClusterApiDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	return fmt.Errorf("Joining new nodes to this cluster is done by editing the KubeadmControlPlane and MachineDeployment resources in the management cluster")
}

func (cad *ClusterApiDriver) Delete() error {
	cad.Deleted = true
	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
		cad.ClusterResources = cdi
	}

	clusterObj, err := capicommon.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return err
	}
	cad.ResourceNamespace = clusterObj.GetNamespace()
	clusterName := clusterObj.GetName()

	// Delete cluster from bootstrap
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}
	log.Infof("Deleting Cluster %s/%s", cad.ResourceNamespace, clusterName)
	if err := k8s.DeleteResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, cad.ResourceNamespace); err != nil {
		return err
	}
	// Wait for deletion
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for deletion",
			WaitFunction: func(i interface{}) error {
				_, _, err := util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
					u, err := k8s.GetResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, cad.ResourceNamespace)
					if err != nil && strings.Contains(err.Error(), "not found") {
						return nil, false, nil
					}
					if err != nil {
						return nil, false, err
					}
					if u != nil {
						return nil, false, fmt.Errorf("Cluster %s/%s is not yet deleted", cad.ResourceNamespace, clusterName)
					}
					return nil, false, nil
				}, nil, 20*time.Minute)
				return err
			},
		},
	})
	if haveError {
		return fmt.Errorf("Error deleting cluster")
	}
	return nil
}

func (cad *ClusterApiDriver) Close() error {
	if cad.Deleted {
		return nil
	}
	if cad.Ephemeral && cad.ClusterConfig.Providers.Vsphere.Namespace != "" {
		if err := start.StopEphemeralCluster(cad.Config, cad.ClusterConfig); err != nil {
			return err
		}
	}
	return nil
}

func (cad *ClusterApiDriver) PostStart() error {
	// No self-managed pivot logic by default for vsphere in this initial implementation
	return nil
}

func (cad *ClusterApiDriver) GetKubeconfigPath() string {
	return cad.KubeConfig
}

func (cad *ClusterApiDriver) GetKubeAPIServerAddress() string {
	// vSphere API endpoint is set in the Cluster resources; fetch from cluster definition
	cluster, err := capicommon.GetClusterObject(cad.ClusterResources)
	if err != nil {
		log.Errorf("Could not get Kubernetes API Server address: %+v", err)
		return ""
	}
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		log.Errorf("Could not read cluster from management cluster: %+v", err)
		return ""
	}
	if err := k8s.GetResource(restConfig, &cluster); err != nil {
		log.Errorf("Could not read cluster from management cluster: %+v", err)
		return ""
	}
	host, _, err := unstructured.NestedString(cluster.Object, capicommon.ClusterEndpointHost...)
	if err != nil {
		log.Errorf("Could not get Kubernetes API Server address: %+v", err)
		return ""
	}
	return host
}

func (cad *ClusterApiDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster:\n    use %s", cad.KubeConfig)
}

func (cad *ClusterApiDriver) DefaultCNIInterfaces() []string {
	return []string{}
}

// Stage satisfies the ClusterDriver interface. No staging needed for vSphere.
func (cad *ClusterApiDriver) Stage(stageDir string) (string, string, bool, error) {
	return "", "", true, nil
}

// Register the driver
func init() {
	driver.RegisterDriver(DriverName, func(config *types.Config, clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
		ccCopy := types.CopyClusterConfig(clusterConfig)
		cad := &ClusterApiDriver{
			Config:        config,
			ClusterConfig: &ccCopy,
		}
		return cad, nil
	})
}
