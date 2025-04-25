// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oracle-cne/ocne/pkg/k8s"

	"github.com/oracle-cne/ocne/pkg/util"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util/capi"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	"github.com/oracle/oci-go-sdk/v65/core"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TemplateData - for each vmTemplateName maintain a list of OLVMMachineTemplates that
// reference it.
type TemplateData struct {
	Template         string
	HasUpdate        bool
	NewTemplate      string
	MachineTemplates []*capi.GraphNode
}

// vmTemplateName should be treated as a constant
var vmTemplateName = []string{"spec", "template", "spec", "ovirt", "vmTemplateName"}

// Stage looks at the resources for an OLVM CAPI cluster and generates as
// much of the material necessary to update a cluster from one version to
// another.  This typically includes uploading new custom images if
// necessary, getting the OCIDs of the latest OCI custom images, and then
// creating new OCIMachineTemplates that use them.  Finally, some instructions
// are printed that tell a user how to apply the staged update to their cluster.
func (cad *OlvmDriver) Stage(version string) (string, string, bool, error) {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return "", "", false, err
	}

	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return "", "", false, err
		}

		cad.ClusterResources = cdi
	}

	clusterObj, err := capi.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return "", "", false, err
	}

	// Update OLVMMachineDeployments to use the new images.
	log.Debugf("Getting graph for Cluster %s in namespace %s", clusterObj.GetName(), clusterObj.GetNamespace())
	graph, err := capi.GetClusterGraph(restConfig, clusterObj.GetNamespace(), clusterObj.GetName())
	if err != nil {
		return "", "", false, err
	}

	// Make sure that there is a control plane defined.  Also check to see
	// if the minor version changed.
	currentKubeVersion, found, err := unstructured.NestedString(graph.ControlPlane.Object.Object, capi.ControlPlaneVersion...)
	if err != nil {
		return "", "", false, err
	} else if !found {
		return "", "", false, fmt.Errorf("%s/%s %s in %s does not have a version", graph.ControlPlane.Object.GroupVersionKind().String(), graph.ControlPlane.Object.GroupVersionKind().String(), graph.ControlPlane.Object.GetName(), graph.ControlPlane.Object.GetNamespace())
	}

	// Apply necessary control plane modifications whether there
	// is an update or not.
	err = capi.PatchControlPlane(restConfig, graph.ControlPlane.Object)
	if err != nil {
		return "", "", false, err
	}

	minorVersionCmp, err := versions.CompareKubernetesVersions(currentKubeVersion, version)
	if err != nil {
		return "", "", false, err
	}
	minorVersionChanged := minorVersionCmp != 0

	// Get the collection of vmTemplateNames in use
	ociImages, err := graphToVMTemplates(graph)
	if err != nil {
		return "", "", false, err
	}
	log.Debugf("%v", ociImages)

	/*
		err = findUpdates(ociImages, version, cad.ClusterConfig.BootVolumeContainerImage, cad.ClusterConfig.Providers.Oci.Profile)
		if err != nil {
			return "", "", false, err
		}

		imageImports := map[string]string{}
		for id, img := range ociImages {
			if img.HasUpdate {
				log.Debugf("OCI image %s with architecture %s has an update", id, img.Arch)
			} else if minorVersionChanged {
				log.Debugf("Updating image OCID due to Kubernetes minor version change")
				// If the minor version has changed, go get the latest
				// image for the new minor version.
				existingImg, found, err := oci.GetImage(*img.Image.DisplayName, version, img.Arch, *img.Image.CompartmentId, cad.ClusterConfig.Providers.Oci.Profile)
				if !found {
					// In theory this is impossible because the
					// check to see if a new image must be uploaded
					// has already found one, but impossible things
					// happen every day.
					return "", "", false, fmt.Errorf("Could not find latest OCI image for Kubernetes version %s and architecture %s", version, img.Arch)
				}
				if err != nil {
					return "", "", false, err
				}

				img.HasUpdate = true
				img.NewId = *existingImg.Id
				continue
			} else {
				log.Debugf("OCI image %s with architecture %s does not have an update", id, img.Arch)
				continue
			}

			oldBv := cad.ClusterConfig.BootVolumeContainerImage
			imgXport := alltransports.TransportFromImageName(cad.ClusterConfig.BootVolumeContainerImage)
			if imgXport == nil {
				cad.ClusterConfig.BootVolumeContainerImage = fmt.Sprintf("docker://%s", cad.ClusterConfig.BootVolumeContainerImage)
			}
			cad.ClusterConfig.BootVolumeContainerImage, err = cmdutil.EnsureBootImageVersion(version, cad.ClusterConfig.BootVolumeContainerImage)
			img.NewId, img.WorkRequestId, err = cad.ensureImage(*img.Image.DisplayName, img.Arch, version, true)
			cad.ClusterConfig.BootVolumeContainerImage = oldBv
			if err != nil {
				return "", "", false, err
			}

			imageImports[img.WorkRequestId] = fmt.Sprintf("Importing updated image for %s", *img.Image.DisplayName)
		}

		err = oci.WaitForWorkRequests(imageImports, cad.ClusterConfig.Providers.Oci.Profile)
		if err != nil {
			return "", "", false, err
		}

		for _, img := range ociImages {
			if img.WorkRequestId != "" {
				err = upload.EnsureImageDetails(*img.Image.CompartmentId, cad.ClusterConfig.Providers.Oci.Profile, img.NewId, img.Arch)

				if err != nil {
					return "", "", false, err
				}
			}
		}

		// Make new machine templates.  This is done by creating a new
		// OCIMachineTemplate for each existing one that uses an existing
		// OCI custom image.
	*/
	updatedMts := map[*capi.GraphNode]*unstructured.Unstructured{}

	for _, img := range ociImages {
		log.Debugf("Creating template for %s", img.NewTemplate)
		newId := img.NewTemplate

		// Template updates can be forced for testing purposes.
		// This is useful because the templates are generated only
		// if there is a reasonable update to perform.
		// This calculation is made by looking at resources, timestamps, and other
		// durable data challenging to set up within the
		// context of a test harness.
		if os.Getenv("OCNE_OLVM_STAGE_FORCE_TEMPLATES") != "" {
			newId = img.NewTemplate
		} else if !img.HasUpdate {
			continue
		}

		for _, mtNode := range img.MachineTemplates {
			mt := mtNode.Object.DeepCopy()
			name := util.IncrementCount(mt.GetName(), "-")
			mt.SetName(name)

			err = unstructured.SetNestedField(mt.Object, newId, "spec", "template", "spec", "ovirt", "vmTemplateName")
			if err != nil {
				return "", "", false, err
			}

			err = k8s.CreateResource(restConfig, mt)
			if err != nil {
				return "", "", false, err
			}

			updatedMts[mtNode] = mt
		}
	}

	// Display some state information and instructions.  The new machine
	// templates that were generated need to get propagated into the
	// MachineDeployments and KubeadmControlPlanes in the cluster.
	var helpMessages []string
	err = graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{}) error {
		updatedMts := arg.(map[*capi.GraphNode]*unstructured.Unstructured)

		var umt *unstructured.Unstructured
		umt, ok := updatedMts[mtNode]
		if !ok {
			return nil
		}

		if parent == graph.ControlPlane {
			kubeVersions, err := versions.GetKubernetesVersions(version)
			if err != nil {
				return err
			}

			patches, err := capi.GetControlPlanePatches(parent.Object, kubeVersions.Kubernetes, umt.GetName())
			if err != nil {
				return err
			}

			helpMessages = append(helpMessages, fmt.Sprintf("To update KubeadmControlPlane %s in %s, run:\n    kubectl patch -n %s kubeadmcontrolplane %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches))
		} else {
			kubeVersions, err := versions.GetKubernetesVersions(version)
			if err != nil {
				return err
			}

			patches := (&util.JsonPatches{}).Replace(capi.MachineDeploymentVersion, kubeVersions.Kubernetes).Replace(append(capi.MachineDeploymentInfrastructureRef, "name"), umt.GetName()).String()
			helpMessages = append(helpMessages, fmt.Sprintf("To update MachineDeployment %s in %s, run:\n    kubectl patch -n %s machinedeployment %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches))
		}

		return nil
	}, updatedMts)
	if err != nil {
		return "", "", false, err
	}
	/*
			// Hand back the kubeconfig for the managed cluster.
			clusterName, _ := clusterObj.GetLabels()[ClusterNameLabel]
			kcfg, err := cad.waitForKubeconfig(client, clusterName)
			kcfgPath, err := util.InMemoryFile(fmt.Sprintf("kcfg.%s", clusterName))

			f, err := os.OpenFile(kcfgPath, os.O_RDWR, 0)
			if err != nil {
				return "", "", false, err
			}
			_, err = f.Write([]byte(kcfg))
			f.Close()
			if err != nil {
				return "", "", false, err
			}
		return kcfgPath, strings.Join(helpMessages, "\n"), true, nil

	*/
	return "", strings.Join(helpMessages, "\n"), minorVersionChanged, nil
}

// imageFromMachineTemplate gets a vmTemplateName from an OLVMMachineTemplate
func imageFromMachineTemplate(mt *unstructured.Unstructured) (string, error) {
	templateName, found, err := unstructured.NestedString(mt.Object, vmTemplateName...)
	if !found {
		err = fmt.Errorf("OLVMMachineTemplate %s in %s has no vmTemplateName", mt.GetName(), mt.GetNamespace())
	}
	if err != nil {
		return "", err
	}
	log.Debugf("OLVMMachineTemplate %s in %s has vmTemplateName %s", mt.GetName(), mt.GetNamespace(), templateName)

	return templateName, nil
}

// doUpdate calculates if there is reason to upload a new OCI custom image
// for a given existing image.
func doUpdate(img *core.Image, arch string, version string, bvImage string, profile string) (bool, error) {
	// Update the image if:
	// - An environment variable forces the update (typically for testing)
	// - The minor version is changing and an image for that version
	//   does not exist
	// - The minor version is not changing and the container image is
	//   newer than the OCI image
	//
	// An image can be forcibly created and uploaded to make it easier
	// to test the upload process.  This logic is reliant on existing
	// resources and timestamps that are difficult to control.  Instead of
	// forcing testers to design complex harnesses, this environment
	// variable makes it easy to simulate an available update.
	if os.Getenv("OCNE_OCI_STAGE_FORCE_UPLOAD") != "" {
		return true, nil
	}

	imgKubeVersion, ok := img.FreeformTags[constants.OCIKubernetesVersionTag]
	if !ok {
		return false, fmt.Errorf("OCI Custom image %s does not have a Kubernetes version tag", *img.Id)
	}

	imgName := *img.DisplayName
	compartmentId := *img.CompartmentId

	existingImg, found, err := oci.GetImage(imgName, version, arch, compartmentId, profile)
	if err != nil {
		return false, err
	}
	if found {
		log.Debugf("Found existing OCI Image for version %s with OCID %s", version, *existingImg.Id)
	}

	// If the versions are the same, check for newer OCI images first
	kubeCmp, err := versions.CompareKubernetesVersions(imgKubeVersion, version)
	if err != nil {
		return false, err
	} else if kubeCmp == 0 {
		// GetImage returns the newest image with the same version and
		// arch.  If the OCIDs of the image from the template and the
		// newest image are the same, then it must be the latest.
		// Otherwise it must not.
		if *existingImg.Id == *img.Id {
			// Check for a new image

			containerImg, err := cmdutil.EnsureBootImageVersion(version, bvImage)
			if err != nil {
				return false, nil
			}

			imgXport := alltransports.TransportFromImageName(containerImg)
			if imgXport == nil {
				containerImg = fmt.Sprintf("docker://%s", containerImg)
			}

			ockImgSpec, err := image.GetImageSpec(containerImg, arch)
			if err != nil {
				return false, err
			}
			log.Debugf("Have container image spec for %s", containerImg)

			ockImgInfo, err := ockImgSpec.Inspect(context.Background())
			if err != nil {
				return false, err
			}
			log.Debugf("Inspecting container image info for %s", containerImg)

			log.Debugf("Checking %v against %v", ockImgInfo.Created, existingImg.TimeCreated.Time)
			if ockImgInfo.Created.After(existingImg.TimeCreated.Time) {
				// Upload the new image
				return true, nil
			}
		}
	} else if found {
		// Don't upload the new image
		return false, nil
	} else {
		// Upload the new image
		return true, nil
	}

	return false, nil
}

// graphToVMTemplates scrapes the graph of CAPI resources and extracts the
// template names from the OLVMMachineTemplates.
func graphToVMTemplates(graph *capi.ClusterGraph) (map[string]*TemplateData, error) {
	ret := map[string]*TemplateData{}

	err := graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{}) error {
		mt := mtNode.Object
		retVal := arg.(map[string]*TemplateData)
		template, err := imageFromMachineTemplate(mt)
		if err != nil {
			return err
		}

		imgData, ok := retVal[template]
		if !ok {
			retVal[template] = &TemplateData{
				Template:         template,
				HasUpdate:        true,
				NewTemplate:      "mgianata-1-31",
				MachineTemplates: []*capi.GraphNode{mtNode},
			}
		} else {
			imgData.MachineTemplates = append(imgData.MachineTemplates, mtNode)
		}
		return nil
	}, ret)
	return ret, err
}

// findUpdates checks to see if an update is available for a set of images.
/*
func findUpdates(imgs map[string]*TemplateData, version string, bvImage string, profile string) error {
	for _, img := range imgs {
		var err error
		img.HasUpdate, err = doUpdate(img.Image, img.Arch, version, bvImage, profile)
		if err != nil {
			return err
		}
	}
	return nil
}
*/
