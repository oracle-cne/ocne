// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/application"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	capicommon "github.com/oracle-cne/ocne/pkg/cluster/driver/capi"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	oci2 "github.com/oracle-cne/ocne/pkg/cluster/template/oci"
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
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	capiclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

const (
	DriverName       = "oci"
	ClusterNameLabel = "cluster.x-k8s.io/cluster-name"

	OciCcmChart         = "oci-ccm"
	OciCcmNamespace     = "kube-system"
	OciCcmRelease       = "oci-ccm"
	OciCcmVersion       = ""
	OciCcmSecretName    = "oci-cloud-controller-manager"
	OciCcmCsiSecretName = "oci-volume-provisioner"
)

// Go does not allow slices or maps as constants.  Pretend they are.
var OCIClusterControlPlaneEndpointHost = []string{"spec", "controlPlaneEndpoint", "host"}

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
	ociConfig, err := oci.GetConfig(cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return nil, err
	}

	proxyValues := map[string]interface{}{
		"httpsProxy": cad.ClusterConfig.Providers.Oci.Proxy.HttpsProxy,
		"httpProxy":  cad.ClusterConfig.Providers.Oci.Proxy.HttpProxy,
		"noProxy":    cad.ClusterConfig.Providers.Oci.Proxy.NoProxy,
	}

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
	}, nil
}

func (cad *ClusterApiDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	ociConfig, err := oci.GetConfig(cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return nil, err
	}

	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment, cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return nil, err
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
		"vcn":         cad.ClusterConfig.Providers.Oci.Vcn,
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
		{
			PreInstall: func() error {
				err := k8s.CreateSecretIfNotExists(kubeClient, OciCcmNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: OciCcmSecretName,
					},
					Data: ociCcmCreds,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				err = k8s.CreateSecretIfNotExists(kubeClient, OciCcmNamespace, &v1.Secret{
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

func (cad *ClusterApiDriver) getOciCcmOptions(restConfig *rest.Config) error {
	// If the values are already set, don't try to set them.  This accounts
	// for two cases: this function has already been called, or there are
	// specific values set in the cluster configuration.
	if cad.ClusterConfig.Providers.Oci.Vcn != "" && cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet1 != "" && cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet2 != "" {
		return nil
	}

	// The values are not populated.  Go get the OCICluster associated
	// with the Cluster and find the values.
	ociCluster, err := cad.getOCIClusterObject()
	if err != nil {
		return err
	}

	ociClusterNs := ociCluster.GetNamespace()
	ociClusterName := ociCluster.GetName()

	err = k8s.GetResource(restConfig, &ociCluster)
	if err != nil {
		return err
	}

	// The values that are required are buried inside .spec.networkSpec.vcn
	spec, err := getMapVal(ociCluster.Object, "spec", ociClusterNs, ociClusterName)
	if err != nil {
		return err
	}

	networkSpec, err := getMapVal(spec, "networkSpec", ociClusterNs, ociClusterName)
	if err != nil {
		return err
	}

	vcn, err := getMapVal(networkSpec, "vcn", ociClusterNs, ociClusterName)
	if err != nil {
		return err
	}

	vcnId, err := getStringVal(vcn, "id", ociClusterNs, ociClusterName)
	if err != nil {
		return err
	}
	log.Debugf("Found VCN OCID %s", vcnId)

	if vcnId == "" {
		return fmt.Errorf("OCICluster %s/%s has an empty vcn id", ociCluster.GetNamespace(), ociCluster.GetName())
	}

	subnets, err := getListVal(vcn, "subnets", ociClusterNs, ociClusterName)
	if err != nil {
		return err
	}

	var serviceSubnets []string
	for _, snIface := range subnets {
		sn, ok := snIface.(map[string]interface{})
		if !ok {
			continue
		}

		log.Debugf("Checking subnet %+v", sn)
		role, err := getStringVal(sn, "role", ociClusterNs, ociClusterName)
		if err != nil {
			continue
		}

		if role != "service-lb" {
			continue
		}

		subnetId, err := getStringVal(sn, "id", ociClusterNs, ociClusterName)
		if err != nil {
			continue
		}

		serviceSubnets = append(serviceSubnets, subnetId)
		log.Debugf("Found service-lb subnet OCID %s", subnetId)
	}

	if len(serviceSubnets) == 0 {
		return fmt.Errorf("OCICluster %s/%s does not have a service-lb subnet", ociCluster.GetNamespace(), ociCluster.GetName())
	}

	cad.ClusterConfig.Providers.Oci.Vcn = vcnId
	cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet1 = serviceSubnets[0]
	if len(serviceSubnets) > 1 {
		cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet2 = serviceSubnets[len(serviceSubnets)-1]
	}

	return nil
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

func getMapVal(source map[string]interface{}, val string, ns string, name string) (map[string]interface{}, error) {
	valRef, ok := source[val]
	if !ok {
		return nil, fmt.Errorf("Cluster %s/%s does not have a %s field", ns, name, val)
	}

	value, ok := valRef.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Cluster %s/%s field %s has an unexpected format", ns, name, val)
	}

	return value, nil
}

func getListVal(source map[string]interface{}, val string, ns string, name string) ([]interface{}, error) {
	valRef, ok := source[val]
	if !ok {
		return nil, fmt.Errorf("Cluster %s/%s does not have a %s field", ns, name, val)
	}

	value, ok := valRef.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Cluster %s/%s field %s has an unexpected format", ns, name, val)
	}

	return value, nil
}

func getStringVal(source map[string]interface{}, val string, ns string, name string) (string, error) {
	valRef, ok := source[val]
	if !ok {
		return "", fmt.Errorf("Cluster %s/%s does not have a %s field", ns, name, val)
	}

	value, ok := valRef.(string)
	if !ok {
		return "", fmt.Errorf("Cluster %s/%s field %s has an unexpected format", ns, name, val)
	}

	return value, nil
}

func (cad *ClusterApiDriver) getOCIClusterObject() (unstructured.Unstructured, error) {
	clusterObj, err := capicommon.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	// The Cluster should have an infrastructure ref that points
	// to the OCICluster.  It looks like:
	//
	// spec:
	//  infrastructureRef:
	//   kind:
	//   name:
	clusterNs := clusterObj.GetNamespace()
	clusterName := clusterObj.GetName()
	clusterSpec, err := getMapVal(clusterObj.Object, "spec", clusterNs, clusterName)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	infraRef, err := getMapVal(clusterSpec, "infrastructureRef", clusterNs, clusterName)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	kind, err := getStringVal(infraRef, "kind", clusterNs, clusterName)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	name, err := getStringVal(infraRef, "name", clusterNs, clusterName)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	// Validate the reference
	if kind != "OCICluster" {
		return unstructured.Unstructured{}, fmt.Errorf("Cluster %s/%s points to an unsupported infrastructure reference %s", clusterNs, clusterName, kind)
	}

	// Find the OCICluster in the templates
	ociClusterObj, err := k8s.FindIn(cad.ClusterResources, func(u unstructured.Unstructured) bool {
		if u.GetKind() != kind {
			return false
		}
		if u.GetName() != name {
			return false
		}
		return true
	})

	if err != nil {
		if k8s.IsNotExist(err) {
			return unstructured.Unstructured{}, fmt.Errorf("Cluster resources do not include a valid cluster.x-k8s.io/v1beta1/Cluster")
		} else {
			return unstructured.Unstructured{}, err
		}
	}

	return ociClusterObj, err
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

	// Unlike other cluster drivers, it is not feasible to have zero
	// worker nodes.  Cluster API will not create control plane nodes
	// with taints removed, and it can get upset if they are removed.
	// Require at least one.
	//
	// If someone really wants to have no workers, then they are free
	// to pass in a cluster definition.
	if clusterConfig.WorkerNodes == 0 {
		clusterConfig.WorkerNodes = 1
	}

	// It's also not feasible to have zero control plane nodes.
	if clusterConfig.ControlPlaneNodes == 0 {
		clusterConfig.ControlPlaneNodes = 1
	}

	// Validate the provider configuration.  For OCI-CCM several pieces of
	// configuration are required.  Specifically, a compartment, a vcn and
	// two subnets (which can be the same).  These values are fed into the
	// OCI-CCM configuration.
	if clusterConfig.Providers.Oci.Compartment == "" {
		return nil, fmt.Errorf("the oci provider requires a compartment in the provider with configuration")
	}

	// If the user has asked for a 1.26 cluster and has not overridden the control plane shape, force the shape to
	// be an amd-compatible shape since 1.26 does not support arm
	if strings.TrimPrefix(clusterConfig.KubeVersion, "v") == "1.26" && slices.Contains(constants.OciArmCompatibleShapes[:], clusterConfig.Providers.Oci.ControlPlaneShape.Shape) {
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

func (cad *ClusterApiDriver) ensureImage(name string, arch string, version string, force bool) (string, string, error) {
	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment, cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return "", "", err
	}

	// Check for a local image.  First see if there is already an image
	// available in OCI
	_, found, err := oci.GetImage(constants.OciImageName, version, arch, compartmentId, cad.ClusterConfig.Providers.Oci.Profile)
	if found && !force {
		// An image was found.  Perfect.
		return "", "", nil
	} else if err != nil {
		return "", "", err
	}

	// Check to see if a converted image already exists.  If so, don't bother
	// making a new one.
	imageName, err := create.DefaultImagePath(create.ProviderTypeOCI, version, arch)
	if err != nil {
		return "", "", err
	}

	_, err = os.Stat(imageName)
	if err != nil && !os.IsNotExist(err) && !force {
		return "", "", err
	}

	// No image exists.  Make one.  Save the existing KC value and substitute
	// the ephemeral one.  Set it back when done.
	oldKcfg := cad.Config.KubeConfig
	oldKver := cad.ClusterConfig.KubeVersion
	cad.Config.KubeConfig = cad.BootstrapKubeConfig
	cad.ClusterConfig.KubeVersion = version
	if cad.ClusterConfig.OsTag == "" {
		cad.ClusterConfig.OsTag = version
	}
	err = create.Create(cad.Config, cad.ClusterConfig, create.CreateOptions{
		ProviderType: create.ProviderTypeOCI,
		Architecture: arch,
	})
	cad.Config.KubeConfig = oldKcfg
	cad.ClusterConfig.KubeVersion = oldKver
	if err != nil {
		return "", "", err
	}

	// Image creation is done.  Upload it.
	imageId, workRequestId, err := upload.UploadAsync(upload.UploadOptions{
		ClusterConfig:     cad.ClusterConfig,
		ProviderType:      upload.ProviderTypeOCI,
		Profile:           cad.ClusterConfig.Providers.Oci.Profile,
		BucketName:        cad.ClusterConfig.Providers.Oci.ImageBucket,
		CompartmentName:   compartmentId,
		ImagePath:         imageName,
		ImageName:         constants.OciImageName,
		KubernetesVersion: version,
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

	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment, cad.ClusterConfig.Providers.Oci.Profile)
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
		controlPlaneImageId, workRequest, err = cad.ensureImage(constants.OciImageName, controlPlaneArch, cad.ClusterConfig.KubeVersion, false)
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
		controlPlaneImageId, controlPlaneWorkRequest, err = cad.ensureImage(constants.OciImageName, controlPlaneArch, cad.ClusterConfig.KubeVersion, false)
		if err != nil {
			return err
		}
		workerImageId, workerWorkRequest, err = cad.ensureImage(constants.OciImageName, workerArch, cad.ClusterConfig.KubeVersion, false)
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
	err = oci.WaitForWorkRequests(imageImports, cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return err
	}
	if controlPlaneImageId != "" {
		err = upload.EnsureImageDetails(compartmentId, cad.ClusterConfig.Providers.Oci.Profile, controlPlaneImageId, controlPlaneArch)
		if err != nil {
			return err
		}
	}
	if workerImageId != "" {
		err = upload.EnsureImageDetails(compartmentId, cad.ClusterConfig.Providers.Oci.Profile, workerImageId, workerArch)
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

// applyResources creates resources in a cluster if the resource does not
// already exist.  If the resource already exists, it is not modified.
func (cad *ClusterApiDriver) applyResources(restConfig *rest.Config) error {
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

func (cad *ClusterApiDriver) Start() (bool, bool, error) {
	// If there is a need to generate a template, do so.
	if cad.FromTemplate {
		// If there is a need to generate a template, ensure that an
		// image is present.
		err := cad.ensureImages()
		if err != nil {
			return false, false, err
		}

		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return false, false, err
		}

		cad.ClusterResources = cdi
	}

	if err := oci2.ValidateClusterResources(cad.ClusterResources); err != nil {
		return false, false, err
	}

	// A fair bit of metadata is anchored by the Cluster
	// object in the bundle of Cluster API resources.  Fetch it
	// to a) make sure it exists and b) fetch useful information
	clusterObj, err := capicommon.GetClusterObject(cad.ClusterResources)
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

	log.Info("Applying Cluster API resources")
	err = cad.applyResources(restConfig)
	if err != nil {
		return false, false, err
	}

	// Get logs for the OCI controller
	ociPods, err := application.PodsForApplication(constants.OCICAPIRelease, constants.OCICAPINamespace, cad.BootstrapKubeConfig)
	log.Debugf("Have %d OCI CAPI pods", len(ociPods))
	if err != nil {
		return false, false, err
	}
	podLogs := []*util.ScanCloser{}
	for _, op := range ociPods {
		podLog, err := k8s.GetPodLogs(clientIface, op, "")
		if err != nil {
			return false, false, err
		}
		re := "^[A-Z][0-9]+"
		md, err := util.NewMessageDispatcher(re, NewOciLogHandler())
		if err != nil {
			err = fmt.Errorf("Internal error: regex \"%s\" does not compile: %v", re, err)
			return false, false, err
		}
		podLogs = append(podLogs, util.Scan(podLog, md))
	}
	defer func(toClose []*util.ScanCloser) {
		for _, tc := range toClose {
			tc.Close()
		}
	}(podLogs)

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

	// Populate the OCI-CCM configuration based on the contents of
	// the OCICluster object.
	err = cad.getOciCcmOptions(restConfig)
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

func (cad *ClusterApiDriver) tryMove(fromBootstrap bool) error {
	// Move the resources to the new cluster
	capiClient, err := capiclient.New(context.TODO(), "")
	if err != nil {
		return nil
	}

	fromKcfg := cad.KubeConfig
	toKcfg := cad.BootstrapKubeConfig
	if fromBootstrap {
		fromKcfg = cad.BootstrapKubeConfig
		toKcfg = cad.KubeConfig
	}

	err = capiClient.Move(context.TODO(), capiclient.MoveOptions{
		FromKubeconfig: capiclient.Kubeconfig{Path: fromKcfg, Context: ""},
		ToKubeconfig:   capiclient.Kubeconfig{Path: toKcfg, Context: ""},
		Namespace:      cad.ResourceNamespace,
		DryRun:         false,
	})
	return err
}

func (cad *ClusterApiDriver) moveCluster(fromBootstrap bool) error {
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Migrating Cluster API resources",
			WaitFunction: func(i interface{}) error {
				c, _ := i.(*ClusterApiDriver)
				for {
					err := c.tryMove(fromBootstrap)
					if err != nil && strings.Contains(err.Error(), "cannot start the move operation") {
						// Tolerate errors where the cluster resources
						// are in flux.  They will either stabilize
						// or not.
						log.Debugf("Could not move cluster: %v", err)

						// Spinning up infrastructure can
						// take a while...
						time.Sleep(time.Second * 3)
						continue
					} else if err != nil {
						return err
					}
					break
				}
				return nil
			},
			Args: cad,
		},
	})
	if haveError {
		return fmt.Errorf("Could not move Cluster API resources")
	}
	return nil
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
	log.Info("Migrating Cluster API resources into self-managed cluster")
	err = cad.moveCluster(true)

	// Scale the bootstrap cluster controllers back up.
	return nil
}

func (cad *ClusterApiDriver) Join(kubeconfigPath string, controlPlaneNodes int, workerNodes int) error {
	return fmt.Errorf("Joining new nodes to this cluster is done by editing the KubeadmControlPlane and MachineDeployment resources in the management cluster")
}

func (cad *ClusterApiDriver) Stop() error {
	return fmt.Errorf("ClusterApiDriver.Stop() is not implemented")
}

func (cad *ClusterApiDriver) waitForClusterDeletion(clusterName string, clusterNs string) error {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	_, _, err = util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		u, err := k8s.GetResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, clusterNs)
		if u != nil {
			log.Debugf("Found cluster %s/%s with UID %s", clusterNs, clusterName, u.GetUID())
		} else {
			log.Debugf("Resource for cluster %s/%s was nil", clusterNs, clusterName)
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, false, nil
			}

			return nil, false, err
		}

		return nil, false, fmt.Errorf("Cluster %s/%s is not yet deleted", clusterNs, clusterName)

	}, nil, 20*time.Minute)
	return err
}

func (cad *ClusterApiDriver) deleteCluster(clusterName string, clusterNs string) error {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	log.Infof("Deleting Cluster %s/%s", clusterNs, clusterName)
	err = k8s.DeleteResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, clusterNs)
	if err != nil {
		return err
	}

	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for deletion",
			WaitFunction: func(i interface{}) error {
				return cad.waitForClusterDeletion(clusterName, clusterNs)
			},
		},
	})

	if haveError {
		return fmt.Errorf("Error deleting cluster")
	}
	return nil
}

func (cad *ClusterApiDriver) Delete() error {
	log.Debugf("Entering Delete for CAPI cluster %s", cad.ClusterConfig.Name)
	cad.Deleted = true
	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
		cad.ClusterResources = cdi
	}

	// Get the namespace.  This is done by finding the metadata
	// for the Cluster resource.
	clusterObj, err := capicommon.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return err
	}

	// No need to check if the label exists again.  The filter function
	// already verified that.
	cad.ResourceNamespace = clusterObj.GetNamespace()
	clusterName := clusterObj.GetName()

	// If this is a self-managed cluster, pivot back into the bootstrap cluster.
	// This is in a for loop so there are break semantics available
	for {
		if !cad.ClusterConfig.Providers.Oci.SelfManaged {
			break
		}

		// If the kubeconfig for the this cluster does not exist,
		// assume that the resources are in the target cluster.
		_, err = os.Stat(cad.KubeConfig)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}

		// If the objects are not in this cluster, don't try
		// to move them.  Treat any error as not being able to
		// find the objects.  If they actually are in the cluster
		// but cannot be found, the deletion code from the bootstrap
		// cluster will fail.
		restConfig, _, err := client.GetKubeClient(cad.KubeConfig)
		if err != nil {
			break
		}
		_, err = k8s.GetResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterObj.GetName(), clusterObj.GetNamespace())
		if err != nil {
			break
		}

		err = cad.moveCluster(false)
		if err != nil {
			return err
		}

		// Only ever do this loop once.  The goal is to have
		// access to break, not to do this more than once.
		break
	}

	return cad.deleteCluster(clusterName, clusterObj.GetNamespace())
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
	cluster, err := cad.getOCIClusterObject()
	if err != nil {
		log.Errorf("Could not get Kubernetes API Server address: %+v", err)
		return ""
	}

	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	err = k8s.GetResource(restConfig, &cluster)
	if err != nil {
		log.Errorf("Could not read OCICluster from management cluster: %+v", err)
		return ""
	}

	ret, _, err := unstructured.NestedString(cluster.Object, OCIClusterControlPlaneEndpointHost...)
	log.Debugf("Found Control Plane Endpoint Host %s", ret)
	if err != nil {
		log.Errorf("Could not get Kubernetes API Server address: %+v", err)
		return ""
	}
	return ret
}

func (cad *ClusterApiDriver) PostInstallHelpStanza() string {
	return fmt.Sprintf("To access the cluster:\n    use %s", cad.KubeConfig)
}

func (cad *ClusterApiDriver) DefaultCNIInterfaces() []string {
	return []string{}
}
