// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"os"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/commands/image/create"
	"github.com/oracle-cne/ocne/pkg/commands/image/upload"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	capiclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

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
	clusterName := clusterObj.GetName()

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
