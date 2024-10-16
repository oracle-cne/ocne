// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	log "github.com/sirupsen/logrus"
	"github.com/oracle-cne/ocne/pkg/constants"
)

// EnsureImage makes sure that an image with a given display name exists.  If
// it already does, then it returns the OCID of that image.  If not, it will
// start the process of importing the image from the given object storage
// description.  A work request OCID is returned to as well to allow the
// caller to monitor pogress.
func EnsureImage(imageName string, k8sVersion string, arch string, compartmentId string, bucketName string, objectName string) (string, string, error) {
	ocid, err := GetImage(imageName, k8sVersion, arch, compartmentId)
	if err == nil {
		return ocid, "", nil
	}
	return ImportImage(imageName, k8sVersion, arch, compartmentId, bucketName, objectName)
}

// GetImage fetches the OCID of an image by name.
func GetImage(imageName string, k8sVersion string, arch string, compartmentId string) (string, error) {
	ctx := context.Background()
	c, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return "", err
	}

	// Check to see if the image already exists
	lir, err := c.ListImages(ctx, core.ListImagesRequest{
		CompartmentId: &compartmentId,
		DisplayName:   &imageName,
	})
	if err != nil {
		return "", err
	}

	// Find an image with the right tags
	for _, img := range lir.Items {
		imgK8sVer, ok := img.FreeformTags[constants.OCIKubernetesVersionTag]
		if !ok {
			continue
		}
		if imgK8sVer != k8sVersion {
			continue
		}

		imgArch, ok := img.FreeformTags[constants.OCIArchitectureTag]
		if !ok {
			continue
		}
		if imgArch != arch {
			continue
		}

		// An image exists, and all the tags match.  Hand back the OCID
		return *img.Id, nil
	}

	return "", fmt.Errorf("could not find image named %s in %s", imageName, compartmentId)
}

// ImportImage creates a custom compute image from the contents of an
// object storage bucket.
func ImportImage(imageName string, k8sVersion string, arch string, compartmentId string, bucketName string, objectName string) (string, string, error) {
	ctx := context.Background()
	c, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return "", "", err
	}

	namespace, err := GetNamespace()
	if err != nil {
		return "", "", err
	}
	osName := "Oracle Linux"
	osVersion := "8"
	req := core.CreateImageRequest{
		CreateImageDetails: core.CreateImageDetails{
			CompartmentId: &compartmentId,
			DisplayName:   &imageName,
			FreeformTags: map[string]string{
				constants.OCIArchitectureTag:      arch,
				constants.OCIKubernetesVersionTag: k8sVersion,
			},
			ImageSourceDetails: core.ImageSourceViaObjectStorageTupleDetails{
				NamespaceName:          &namespace,
				BucketName:             &bucketName,
				ObjectName:             &objectName,
				SourceImageType:        core.ImageSourceDetailsSourceImageTypeQcow2,
				OperatingSystem:        &osName,
				OperatingSystemVersion: &osVersion,
			},
		},
	}

	resp, err := c.CreateImage(ctx, req)
	if err != nil {
		return "", "", err
	}

	return *resp.Image.Id, *resp.OpcWorkRequestId, nil
}

func CreateEFIImageSchema(compartmentId string, imageId string) error {
	ctx := context.Background()
	c, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return err
	}

	schReq := core.ListComputeGlobalImageCapabilitySchemasRequest{}
	schResp, err := c.ListComputeGlobalImageCapabilitySchemas(ctx, schReq)
	if err != nil {
		return err
	}

	if len(schResp.Items) != 1 {
		return fmt.Errorf("Unexpected number of global image capability schemas")
	}

	efi := "UEFI_64"
	launchMode := "PARAVIRTUALIZED"
	upCapReq := core.CreateComputeImageCapabilitySchemaRequest{
		CreateComputeImageCapabilitySchemaDetails: core.CreateComputeImageCapabilitySchemaDetails{
			ImageId:       &imageId,
			CompartmentId: &compartmentId,
			ComputeGlobalImageCapabilitySchemaVersionName: schResp.Items[0].CurrentVersionName,
			SchemaData: map[string]core.ImageCapabilitySchemaDescriptor{
				"Compute.Firmware": core.EnumStringImageCapabilitySchemaDescriptor{
					Values:       []string{efi},
					DefaultValue: &efi,
					Source:       core.ImageCapabilitySchemaDescriptorSourceImage,
				},
				"Compute.LaunchMode": core.EnumStringImageCapabilitySchemaDescriptor{
					Values:       []string{launchMode},
					DefaultValue: &launchMode,
					Source:       core.ImageCapabilitySchemaDescriptorSourceImage,
				},
			},
		},
	}
	_, err = c.CreateComputeImageCapabilitySchema(ctx, upCapReq)
	return err
}

// EnsureCompatibleImageShapes ensures that the image has the correct list of compatible image shapes,
// based on the image architecture.
func EnsureCompatibleImageShapes(imageId string, arch string) error {
	// amd-based images already have the correct shapes, so we only have work to do if this is arm
	if arch != "arm64" {
		return nil
	}

	c, err := core.NewComputeClientWithConfigurationProvider(common.DefaultConfigProvider())
	if err != nil {
		return err
	}

	// First find all shapes that aren't compatible with ARM
	limit := 50
	var page *string
	shapesToRemove := []string{}

	for {
		req := core.ListImageShapeCompatibilityEntriesRequest{
			ImageId: &imageId,
			Limit:   &limit,
			Page:    page,
		}
		resp, err := c.ListImageShapeCompatibilityEntries(context.TODO(), req)
		if err != nil {
			return err
		}

		page = resp.OpcNextPage
		for _, entry := range resp.Items {
			// A1 are compatible, and Generic are special shape names that can't be removed
			if !strings.Contains(*entry.Shape, ".A1.") && !strings.Contains(*entry.Shape, "Generic") {
				shapesToRemove = append(shapesToRemove, *entry.Shape)
			}
		}

		if page == nil {
			break
		}
	}

	// Remove incompatible shapes from the compatibility list
	for _, shape := range shapesToRemove {
		req := core.RemoveImageShapeCompatibilityEntryRequest{
			ImageId:   &imageId,
			ShapeName: &shape,
		}
		_, err := c.RemoveImageShapeCompatibilityEntry(context.TODO(), req)
		if err != nil {
			// On error, log it but keep going
			log.Warnf("Unable to remove shape entry '%s' from image: %v", shape, err)
		}
	}

	// Now add ARM-compatible shapes (note this operation is idempotent so it's fine if the shape is already on the image)
	for _, shape := range constants.OciArmCompatibleShapes {
		req := core.AddImageShapeCompatibilityEntryRequest{
			ImageId:   &imageId,
			ShapeName: &shape,
		}
		_, err := c.AddImageShapeCompatibilityEntry(context.TODO(), req)
		if err != nil {
			return err
		}
	}

	return nil
}
