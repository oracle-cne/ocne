// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/commands/image/create"
	"github.com/oracle-cne/ocne/pkg/commands/image/upload"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	capiclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

const (
	DriverName       = "oci"
	ClusterNameLabel = "cluster.x-k8s.io/cluster-name"

	OciCcmChart         = "oci-ccm"
	OciCcmNamespace     = "kube-system"
	OciCcmRelease       = "oci-ccm"
	OciCcmVersion       = "1.28.0"
	OciCcmSecretName    = "oci-cloud-controller-manager"
	OciCcmCsiSecretName = "oci-volume-provisioner"
)

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

func (cad *ClusterApiDriver) getApplications() ([]install.ApplicationDescription, error) {
	ociConfig, err := oci.GetConfig()
	if err != nil {
		return nil, err
	}

	return []install.ApplicationDescription{
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CertManagerChart,
				Namespace: constants.CertManagerNamespace,
				Release:   constants.CertManagerRelease,
				Version:   constants.CertManagerVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CoreCAPIChart,
				Namespace: constants.CoreCAPINamespace,
				Release:   constants.CoreCAPIRelease,
				Version:   constants.CoreCAPIVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.OCICAPIChart,
				Namespace: constants.OCICAPINamespace,
				Release:   constants.OCICAPIRelease,
				Version:   constants.OCICAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"authConfig": map[string]interface{}{
						"fingerprint":          ociConfig.Fingerprint,
						"key":                  ociConfig.Key,
						"passphrase":           ociConfig.Passphrase,
						"region":               ociConfig.Region,
						"tenancy":              ociConfig.Tenancy,
						"useInstancePrincipal": fmt.Sprintf("%t", ociConfig.UseInstancePrincipal),
						"user":                 ociConfig.User,
					},
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.KubeadmBootstrapCAPIChart,
				Namespace: constants.KubeadmBootstrapCAPINamespace,
				Release:   constants.KubeadmBootstrapCAPIRelease,
				Version:   constants.KubeadmBootstrapCAPIVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.KubeadmControlPlaneCAPIChart,
				Namespace: constants.KubeadmControlPlaneCAPINamespace,
				Release:   constants.KubeadmControlPlaneCAPIRelease,
				Version:   constants.KubeadmControlPlaneCAPIVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
	}, nil
}

func (cad *ClusterApiDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	ociConfig, err := oci.GetConfig()
	if err != nil {
		return nil, err
	}

	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment)
	if err != nil {
		return nil, err
	}

	// Once the kubeconfig is available, the OCICluster will be populated
	// with enough information to extract some important values.  In
	// particular, the OCID of the VCN
	ociClusterObj, err := k8s.FindIn(cad.ClusterResources, func(u unstructured.Unstructured) bool {
		if u.GetKind() == "OCICluster" {
			return true
		}
		return false
	})
	ociCluster, err := k8s.GetResourceByIdentifier(restConfig, "infrastructure.cluster.x-k8s.io", "v1beta2", "OCICluster", ociClusterObj.GetName(), ociClusterObj.GetNamespace())
	if err != nil {
		return nil, err
	}

	vcnId, found, err := unstructured.NestedString(ociCluster.Object, "spec", "networkSpec", "vcn", "id")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("could not find VCN OCID in OCICluster")
	}

	authCreds := map[string]interface{}{
		"auth": map[string]interface{}{
			"region":                ociConfig.Region,
			"tenancy":               ociConfig.Tenancy,
			"user":                  ociConfig.User,
			"key":                   ociConfig.Key,
			"passphrase":            ociConfig.Passphrase,
			"fingerprint":           ociConfig.Fingerprint,
			"useInstancePrincipals": ociConfig.UseInstancePrincipal,
		},
		"compartment": compartmentId,
		"vcn":         vcnId,
		"loadBalancer": map[string]interface{}{
			"subnet1":                    cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet1,
			"subnet2":                    cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet2,
			"securityListManagementMode": "None",
		},
	}
	authCredBytes, err := yaml.Marshal(authCreds)
	if err != nil {
		return nil, err
	}

	ociCcmCreds := map[string][]byte{
		"cloud-provider.yaml": authCredBytes,
	}
	ociCsiCreds := map[string][]byte{
		"config.yaml": authCredBytes,
	}

	ret := []install.ApplicationDescription{
		install.ApplicationDescription{
			PreInstall: func() error {
				err := k8s.CreateSecret(kubeClient, OciCcmNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: OciCcmSecretName,
					},
					Data: ociCcmCreds,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				err = k8s.CreateSecret(kubeClient, OciCcmNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: OciCcmCsiSecretName,
					},
					Data: ociCsiCreds,
					Type: "Opaque",
				})
				return err
			},
			Application: &types.Application{
				Name:      OciCcmChart,
				Namespace: OciCcmNamespace,
				Release:   OciCcmRelease,
				Version:   OciCcmVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
	}

	return ret, nil
}

func (cad *ClusterApiDriver) waitForControllers(kubeClient kubernetes.Interface) error {
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
			Message: "Waiting for OCI Cluster API Controllers",
			WaitFunction: func(i interface{}) error {
				return k8s.WaitForDeployment(kubeClient, constants.OCICAPINamespace, constants.OCICAPIDeployment, 1)
			},
		},
	})
	if haveError {
		return fmt.Errorf("Not all Cluster API controllers became available")
	}
	return nil
}

func (cad *ClusterApiDriver) getClusterObject() (unstructured.Unstructured, error) {
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

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
	var err error
	doTemplate := false
	cd := clusterConfig.ClusterDefinition
	cdi := clusterConfig.ClusterDefinitionInline
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
			cd = filepath.Join(clusterConfig.WorkingDirectory, cd)
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

	// Validate the provider configuration.  For OCI-CCM several pieces of
	// configuration are required.  Specifically, a compartment, a vcn and
	// two subnets (which can be the same).  These values are fed into the
	// OCI-CCM configuration.
	if clusterConfig.Providers.Oci.Compartment == "" {
		return nil, fmt.Errorf("the oci provider requires a compartment in the provider with configuration")
	}
	if clusterConfig.Providers.Oci.Vcn == "" {
		return nil, fmt.Errorf("the oci provider requires a vcn in the provider configuration to use with OCI-CCM")
	}
	if clusterConfig.Providers.Oci.LoadBalancer.Subnet1 == "" || clusterConfig.Providers.Oci.LoadBalancer.Subnet2 == "" {
		return nil, fmt.Errorf("the oci provider requires two subnets in the provider configuration to use with OCI-CCM")
	}

	// If the user has asked for a 1.26 cluster and has not overridden the control plane shape, force the shape to
	// be an amd-compatible shape since 1.26 does not support arm
	if strings.TrimPrefix(clusterConfig.KubeVersion, "v") == "1.26" && clusterConfig.Providers.Oci.ControlPlaneShape.Shape == constants.OciVmStandardA1Flex {
		clusterConfig.Providers.Oci.ControlPlaneShape.Shape = constants.OciVmStandardE4Flex
	}

	cad := &ClusterApiDriver{
		Config:           config,
		ClusterConfig:    clusterConfig,
		ClusterResources: cdi,
		FromTemplate:     doTemplate,
	}
	bootstrapKubeConfig, isEphemeral, err := start.EnsureCluster(config.Providers.Oci.KubeConfigPath, config, clusterConfig)
	if err != nil {
		return nil, err
	}

	cad.Ephemeral = isEphemeral
	cad.BootstrapKubeConfig = bootstrapKubeConfig

	// Install any necessary components into the admin cluster
	capiApplications, err := cad.getApplications()
	if err != nil {
		return nil, err
	}

	err = install.InstallApplications(capiApplications, cad.BootstrapKubeConfig, config.Quiet)
	if err != nil {
		return nil, err
	}

	_, kubeClient, err := client.GetKubeClient(cad.BootstrapKubeConfig)
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

	cad.KubeConfig, err = client.GetKubeconfigPath(fmt.Sprintf("kubeconfig.%s", cad.ClusterConfig.Name))
	if err != nil {
		return nil, err
	}

	return cad, nil
}

func (cad *ClusterApiDriver) ensureImage(arch string) (string, string, error) {
	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment)
	if err != nil {
		return "", "", err
	}

	// Check for a local image.  First see if there is already an image
	// available in OCI
	_, err = oci.GetImage(constants.OciImageName, cad.ClusterConfig.KubeVersion, arch, compartmentId)
	if err == nil {
		// An image was found.  Perfect.
		return "", "", nil
	}

	// Check to see if a converted image already exists.  If so, don't bother
	// making a new one.
	imageName, err := create.DefaultImagePath(create.ProviderTypeOCI, cad.ClusterConfig.KubeVersion, arch)
	if err != nil {
		return "", "", err
	}

	_, err = os.Stat(imageName)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}

	// No image exists.  Make one.  Save the existing KC value and substitute
	// the ephemeral one.  Set it back when done.
	oldKcfg := cad.Config.KubeConfig
	cad.Config.KubeConfig = cad.BootstrapKubeConfig
	err = create.Create(cad.Config, cad.ClusterConfig, create.CreateOptions{
		ProviderType: create.ProviderTypeOCI,
		Architecture: arch,
	})
	cad.Config.KubeConfig = oldKcfg
	if err != nil {
		return "", "", err
	}

	// Image creation is done.  Upload it.
	imageId, workRequestId, err := upload.UploadAsync(upload.UploadOptions{
		ProviderType:      upload.ProviderTypeOCI,
		BucketName:        cad.ClusterConfig.Providers.Oci.ImageBucket,
		CompartmentName:   compartmentId,
		ImagePath:         imageName,
		ImageName:         constants.OciImageName,
		KubernetesVersion: cad.ClusterConfig.KubeVersion,
		ImageArchitecture: arch,
	})
	if err != nil {
		return "", "", err
	}

	return imageId, workRequestId, nil
}

func (cad *ClusterApiDriver) ensureImages() error {
	controlPlaneArch := oci.ArchitectureFromShape(cad.ClusterConfig.Providers.Oci.ControlPlaneShape.Shape)
	workerArch := oci.ArchitectureFromShape(cad.ClusterConfig.Providers.Oci.WorkerShape.Shape)

	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment)
	if err != nil {
		return err
	}

	// If the control plane arch and worker arch are the same, only import the
	// one image.
	imageImports := map[string]string{}
	controlPlaneImageId := ""
	workerImageId := ""
	if controlPlaneArch == workerArch {
		workRequest := ""
		var err error
		controlPlaneImageId, workRequest, err = cad.ensureImage(controlPlaneArch)
		if err != nil {
			return err
		}
		if workRequest != "" {
			imageImports[workRequest] = "Importing image"
		}
	} else {
		controlPlaneWorkRequest := ""
		workerWorkRequest := ""
		var err error
		controlPlaneImageId, controlPlaneWorkRequest, err = cad.ensureImage(controlPlaneArch)
		if err != nil {
			return err
		}
		workerImageId, workerWorkRequest, err = cad.ensureImage(workerArch)
		if err != nil {
			return err
		}

		if controlPlaneWorkRequest != "" {
			imageImports[controlPlaneWorkRequest] = "Importing control plane image"
		}
		if workerWorkRequest != "" {
			imageImports[workerWorkRequest] = "Importing worker image"
		}
	}
	err = oci.WaitForWorkRequests(imageImports)
	if err != nil {
		return err
	}
	if controlPlaneImageId != "" {
		err = upload.EnsureImageDetails(compartmentId, controlPlaneImageId, controlPlaneArch)
		if err != nil {
			return err
		}
	}
	if workerImageId != "" {
		err = upload.EnsureImageDetails(compartmentId, workerImageId, workerArch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cad *ClusterApiDriver) waitForKubeconfig(client kubernetes.Interface, clusterName string) (string, error) {
	var kubeconfig string
	kcfgSecretIface, _, err := util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		log.Debugf("Looking for secrets in %s with label %s = %s", cad.ResourceNamespace, ClusterNameLabel, clusterName)
		secrets, err := k8s.FindSecretsByLabelKeyVal(client, cad.ResourceNamespace, ClusterNameLabel, clusterName)
		if err != nil {
			log.Debugf("Error finding secret: %+v", err)
			return nil, false, err
		}
		if len(secrets.Items) == 0 {
			log.Debugf("No secrets found")
			return nil, false, fmt.Errorf("no kubeconfig found for cluster %s in namespace %s", clusterName, cad.ResourceNamespace)
		}

		// Find the secret that looks kubeconfig'y
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

	// Get the kubeconfig string out of the secret
	kcfgBytes, ok := kcfgSecret.Data["value"]
	if !ok {
		return "", fmt.Errorf("%s is not a valid kubeconfig secret", kcfgSecret.ObjectMeta.Name)
	}
	kubeconfig = string(kcfgBytes)

	return kubeconfig, nil
}

func (cad *ClusterApiDriver) Start() (bool, bool, error) {
	// If there is a need to generate a template, do so.
	if cad.FromTemplate {
		// If there is a need to generate a template, ensure that an
		// image is present.
		err := cad.ensureImages()
		if err != nil {
			return false, false, err
		}

		cdi, err := template.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return false, false, err
		}

		cad.ClusterResources = cdi
	}

	if err := template.ValidateClusterResources(cad.ClusterResources); err != nil {
		return false, false, err
	}

	// A fair bit of metadata is anchored by the Cluster
	// object in the bundle of Cluster API resources.  Fetch it
	// to a) make sure it exists and b) fetch useful information
	clusterObj, err := cad.getClusterObject()
	if err != nil {
		return false, false, err
	}

	cad.ResourceNamespace = clusterObj.GetNamespace()

	// Apply the given yaml.
	restConfig, clientIface, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return false, false, err
	}

	err = k8s.CreateNamespaceIfNotExists(clientIface, cad.ResourceNamespace)
	if err != nil {
		return false, false, err
	}

	ya, err := k8s.NewYAMLApplier(restConfig, "")
	if err != nil {
		return false, false, err
	}

	log.Info("Applying Cluster API resources")
	err = ya.ApplyS(cad.ClusterResources)
	if err != nil {
		return false, false, err
	}

	// Get the kubeconfig.  This is done by finding a secret
	// that has the same label as the top level Cluster resource.
	clusterName, _ := clusterObj.GetLabels()[ClusterNameLabel]

	var kubeconfig string
	logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for kubeconfig",
			WaitFunction: func(i interface{}) error {
				kubeconfig, err = cad.waitForKubeconfig(clientIface, clusterName)
				if err != nil {
					return err
				}
				return nil
			},
		},
	})

	if err != nil {
		return false, false, err
	}

	err = os.WriteFile(cad.KubeConfig, []byte(kubeconfig), 0700)
	if err != nil {
		return false, false, err
	}

	_, kubeClient, err := client.GetKubeClient(cad.KubeConfig)

	// Wait for the cluster to start
	_, err = k8s.WaitUntilGetNodesSucceeds(kubeClient)
	if err != nil {
		return false, false, err
	}

	// Once the cluster has started, install all the necessary applications
	// for the workload cluster.
	workloadApps, err := cad.getWorkloadClusterApplications(restConfig, kubeClient)
	if err != nil {
		return false, false, err
	}

	log.Info("Installing applications into workload cluster")
	err = install.InstallApplications(workloadApps, cad.KubeConfig, cad.Config.Quiet)
	if err != nil {
		return false, false, err
	}

	return false, false, nil
}

func (cad *ClusterApiDriver) PostStart() error {
	// If the cluster is not self managed, then the configuration is
	// complete.
	if !cad.ClusterConfig.Providers.Oci.SelfManaged {
		return nil
	}

	// Install the Cluster API controllers into the new cluster
	capiApplications, err := cad.getApplications()
	if err != nil {
		return err
	}

	err = install.InstallApplications(capiApplications, cad.KubeConfig, cad.Config.Quiet)
	if err != nil {
		return err
	}

	_, kubeClient, err := client.GetKubeClient(cad.KubeConfig)
	if err != nil {
		return err
	}

	// Wait for controllers to settle.  Nodes should have their cloud
	// provider taints removed.  There should also be at least one
	// node that has no control plane taints.
	err = cad.waitForControllers(kubeClient)
	if err != nil {
		return err
	}

	// Move the resources to the new cluster
	capiClient, err := capiclient.New(context.TODO(), "")
	if err != nil {
		return nil
	}

	log.Info("Migrating Cluster API resources into self-managed cluster")
	err = capiClient.Move(context.TODO(), capiclient.MoveOptions{
		FromKubeconfig: capiclient.Kubeconfig{Path: cad.BootstrapKubeConfig, Context: ""},
		ToKubeconfig:   capiclient.Kubeconfig{Path: cad.KubeConfig, Context: ""},
		Namespace:      cad.ResourceNamespace,
		DryRun:         false,
	})
	if err != nil {
		return err
	}

	// Scale the bootstrap cluster controllers back up.
	return nil
}

func (cad *ClusterApiDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	return fmt.Errorf("Joining new nodes to this cluster is done by editing the KubeadmControlPlane and MachineDeployment resources in the management cluster")
}

func (cad *ClusterApiDriver) Stop() error {
	return fmt.Errorf("ClusterApiDriver.Stop() is not implemented")
}

func (cad *ClusterApiDriver) Delete() error {
	log.Debugf("Entering Delete for CAPI cluster %s", cad.ClusterConfig.Name)
	cad.Deleted = true
	if cad.FromTemplate {
		cdi, err := template.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
		cad.ClusterResources = cdi
	}

	// Get the namespace.  This is done by finding the metadata
	// for the Cluster resource.
	clusterObj, err := cad.getClusterObject()
	if err != nil {
		return err
	}

	// No need to check if the label exists again.  The filter function
	// already verified that.
	cad.ResourceNamespace = clusterObj.GetNamespace()
	resourceName := clusterObj.GetName()

	// If this is a self-managed cluster, pivot back into the bootstrap cluster.
	if cad.ClusterConfig.Providers.Oci.SelfManaged {
		log.Infof("Migrating Cluster API resources to bootstrap cluster")
		capiClient, err := capiclient.New(context.TODO(), "")
		if err != nil {
			return nil
		}

		err = capiClient.Move(context.TODO(), capiclient.MoveOptions{
			FromKubeconfig: capiclient.Kubeconfig{Path: cad.KubeConfig, Context: ""},
			ToKubeconfig:   capiclient.Kubeconfig{Path: cad.BootstrapKubeConfig, Context: ""},
			Namespace:      cad.ResourceNamespace,
			DryRun:         false,
		})
		if err != nil {
			return err
		}
	}

	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	log.Infof("Deleting Cluster %s/%s", cad.ResourceNamespace, resourceName)
	err = k8s.DeleteResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", resourceName, cad.ResourceNamespace)
	if err != nil {
		return err
	}

	// Wait here for cluster resources to be deleted.  See
	// comment in cad.Close() below for an explanation.

	return nil
}

func (cad *ClusterApiDriver) Close() error {
	// There needs to be some logic to figure out when a cluster
	// is done being deleted.  It is not reasoble to develop
	// this against the OCI CAPI provider because it is unreliable
	// when deleting clusters.  For now, leave the ephemeral one
	// behind so that deletion can continue in the background.
	if cad.Deleted {
		return nil
	}

	if cad.Ephemeral && cad.ClusterConfig.Providers.Oci.SelfManaged {
		err := start.StopEphemeralCluster(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cad *ClusterApiDriver) GetKubeconfigPath() string {
	return cad.KubeConfig
}

func (cad *ClusterApiDriver) GetKubeAPIServerAddress() string {
	return ""
}

func (cad *ClusterApiDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster:\n    use %s", cad.KubeConfig)
}

func (Cad *ClusterApiDriver) DefaultCNIInterfaces() []string {
	// ens3 is the default OCI vNIC name for x86
	// enp0s6 is the default for arm
	return []string{"ens3", "enp0s6"}
}
