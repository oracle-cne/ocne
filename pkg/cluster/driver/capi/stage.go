// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	"os"

	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/image/upload"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/capi"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type OciImageData struct {
	Image *core.Image
	HasUpdate bool
	Arch string
	NewId string
	WorkRequestId string
	MachineTemplates []*capi.GraphNode
}

// These should be treated as constants
var MachineTemplateImageId []string = []string{"spec", "template", "spec", "imageId"}
var MachineTemplateShape []string = []string{"spec", "template", "spec", "shape"}

func imageFromMachineTemplate(mt *unstructured.Unstructured) (*core.Image, error) {
	imageId, found, err := unstructured.NestedString(mt.Object, MachineTemplateImageId...)
	if !found {
		err = fmt.Errorf("MachineTemplate %s in %s has no imageId", mt.GetName(), mt.GetNamespace())
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("MachineTemplate %s in %s has imageId %s", mt.GetName(), mt.GetNamespace(), imageId)

	img, err := oci.GetImageById(imageId)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func doUpdate(img *core.Image, arch string, version string, bvImage string) (bool, error) {

	// Update the image if:
	// - An environment variable forces the update (typically for testing)
	// - The minor version is changing and an image for that version
	//   does not exist
	// - The minor version is not changing and the container image is
	//   newer than the OCI image
	if os.Getenv("OCNE_OCI_STAGE_FORCE_UPLOAD") != "" {
		return true, nil
	}

	imgKubeVersion, ok := img.FreeformTags[constants.OCIKubernetesVersionTag]
	if !ok {
		return false, fmt.Errorf("OCI Custom image %s does not have a Kubernetes version tag", *img.Id)
	}

	imgName := *img.DisplayName
	compartmentId := *img.CompartmentId

	existingImg, found, err := oci.GetImage(imgName, version, arch, compartmentId)
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

func graphToImages(graph *capi.ClusterGraph) (map[string]*OciImageData, error) {
	ret := map[string]*OciImageData{}

	err := graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{})error{
		mt := mtNode.Object
		retVal := arg.(map[string]*OciImageData)
		img, err := imageFromMachineTemplate(mt)
		if err != nil {
			return err
		}


		shape, found, err := unstructured.NestedString(mt.Object, MachineTemplateShape...)
		if !found {
			err = fmt.Errorf("MachineTemplate %s in %s has no shape", mt.GetName(), mt.GetNamespace())
		}
		if err != nil {
			return err
		}

		arch := oci.ArchitectureFromShape(shape)
		log.Debugf("MachineTemplate %s in %s has shape %s of architecture %s", mt.GetName(), mt.GetNamespace(), shape, arch)

		imgData, ok := retVal[*img.Id]
		if !ok {
			retVal[*img.Id] = &OciImageData{
				Image: img,
				HasUpdate: false,
				Arch: arch,
				MachineTemplates: []*capi.GraphNode{mtNode},
			}
		} else {
			imgData.MachineTemplates = append(imgData.MachineTemplates, mtNode)
		}
		return nil
	}, ret)
	return ret, err
}

func findUpdates(imgs map[string]*OciImageData, version string, bvImage string) error {
	for _, img := range imgs {
		var err error
		img.HasUpdate, err = doUpdate(img.Image, img.Arch, version, bvImage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cad *ClusterApiDriver) Stage(version string) error {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}

		cad.ClusterResources = cdi
	}

	clusterObj, err := cad.getClusterObject()
	if err != nil {
		return err
	}

	// Change any cluster resource state that may be required to move
	// from one version to the next.

	// Update OCIMachineDeployments to use the new images.
	log.Debugf("Getting graph for Cluster %s in namespace %s", clusterObj.GetName(), clusterObj.GetNamespace())
	graph, err := capi.GetClusterGraph(restConfig, clusterObj.GetNamespace(), clusterObj.GetName())
	if err != nil {
		return err
	}

	// Make sure that there is a control plane defined.  Also check to see
	// if the minor version changed.
	currentKubeVersion, found, err := unstructured.NestedString(graph.ControlPlane.Object.Object, capi.ControlPlaneVersion...)
	if err != nil {
		return err
	} else if !found {
		return fmt.Errorf("%s/%s %s in %s does not have a version", graph.ControlPlane.Object.GroupVersionKind().String(), graph.ControlPlane.Object.GetName(), graph.ControlPlane.Object.GetNamespace())
	}
	minorVersionCmp, err := versions.CompareKubernetesVersions(currentKubeVersion, version)
	if err != nil {
		return err
	}
	minorVersionChanged := minorVersionCmp != 0

	// Check the existing images for the machine templates and see if there
	// are updates available.  If so, upload the new images and generate
	// new machine templates that consume them.
	ociImages, err := graphToImages(graph)
	if err != nil {
		return err
	}

	err = findUpdates(ociImages, version, cad.ClusterConfig.BootVolumeContainerImage)
	if err != nil {
		return err
	}

	imageImports := map[string]string{}
	for id, img := range ociImages {
		if img.HasUpdate {
			log.Debugf("OCI image %s with architecture %s has an update", id, img.Arch)
		} else if minorVersionChanged {
			log.Debugf("Updating image OCID due to Kubernetes minor version change")
			// If the minor version has changed, go get the latest
			// image for the new minor version.
			existingImg, found, err := oci.GetImage(*img.Image.DisplayName, version, img.Arch, *img.Image.CompartmentId)
			if !found {
				// In theory this is impossible because the
				// check to see if a new image must be uploaded
				// has already found one, but impossible things
				// happen every day.
				return fmt.Errorf("Could not find latest OCI image for Kubernetes version %s and architecture %s", version, img.Arch)
			}
			if err != nil {
				return err
			}

			img.HasUpdate = true
			img.NewId = *existingImg.Id
			continue
		} else {
			log.Debugf("OCI image %s with architecture %s does not have an update", id, img.Arch)
			continue
		}

		img.NewId, img.WorkRequestId, err = cad.ensureImage(*img.Image.DisplayName, img.Arch, version, true)
		if err != nil {
			return err
		}

		imageImports[img.WorkRequestId] = fmt.Sprintf("Importing updated image for %s", *img.Image.DisplayName)
	}

	err = oci.WaitForWorkRequests(imageImports)
	if err != nil {
		return err
	}

	for _, img := range ociImages {
		if img.WorkRequestId != "" {
			err = upload.EnsureImageDetails(*img.Image.CompartmentId, img.NewId, img.Arch)

			if err != nil {
				return err
			}
		}
	}

	// Make new machine templates
	updatedMts := map[*capi.GraphNode]*unstructured.Unstructured{}
	for _, img := range ociImages {
		log.Debugf("Creating template for %s", *img.Image.Id)
		newId := img.NewId
		if  os.Getenv("OCNE_OCI_STAGE_FORCE_TEMPLATES") != "" {
			newId = *img.Image.Id
		} else if !img.HasUpdate {
			continue
		}

		for _, mtNode := range img.MachineTemplates {
			mt := mtNode.Object.DeepCopy()
			name := util.IncrementCount(mt.GetName(), "-")
			mt.SetName(name)

			err = unstructured.SetNestedField(mt.Object, newId, "spec", "template", "spec", "imageId")
			if err != nil {
				return err
			}

			err = k8s.CreateResource(restConfig, mt)
			if err != nil {
				return err
			}

			updatedMts[mtNode] = mt
		}
	}

	// Spit out some state information and instructions.  The new machine
	// templates that were generated need to get propagated into the
	// MachineDeployments and KubeadmControlPlanes in the cluster.
	err = graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{})error{
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

			patches := (&util.JsonPatches{}).Replace(capi.ControlPlaneVersion, kubeVersions.Kubernetes).Replace(append(capi.ControlPlaneMachineTemplateInfrastructureRef, "name"), umt.GetName()).String()

			fmt.Printf("To update KubeadmControlPlane %s in %s, run: kubectl patch -n %s kubeadmcontrolplane %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches)
		} else {
			kubeVersions, err := versions.GetKubernetesVersions(version)
			if err != nil {
				return err
			}

			patches := (&util.JsonPatches{}).Replace(capi.MachineDeploymentVersion, kubeVersions.Kubernetes).Replace(append(capi.MachineDeploymentInfrastructureRef, "name"), umt.GetName()).String()
			fmt.Printf("To update MachineDeployment %s in %s, run: kubectl patch -n %s machinedeployment %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches)
		}

		return nil
	}, updatedMts)

	return nil
}
