// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle/oci-go-sdk/v65/common"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/containers/image/v5/copy"
	file2 "github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
)

const ProviderTypeOCI = "oci"
const ProviderTypeOstree = "ostree"

func setCompartmentId(options *UploadOptions, ociConfig common.ConfigurationProvider) error {
	// Compartment ID is already resolved.
	if options.compartmentId != "" {
		return nil
	}

	compartmentId, err := oci.GetCompartmentId(options.CompartmentName, ociConfig)
	if err != nil {
		return err
	}

	options.compartmentId = compartmentId
	return nil
}

// UploadAsync uploads a VM image to object storage and then begins
// the import process.  A work request is returned for the import.
func UploadAsync(options UploadOptions, ociConfig common.ConfigurationProvider) (string, string, error) {
	err := setCompartmentId(&options, ociConfig)
	if err != nil {
		return "", "", err
	}

	fpath, err := file2.AbsDir(options.ImagePath)
	if err != nil {
		return "", "", err
	}

	file, err := os.Open(fpath)
	if err != nil {
		return "", "", err
	} else {
		defer file.Close()
	}

	stat, err := file.Stat()
	if err != nil {
		return "", "", err
	}

	options.size = stat.Size()
	options.filename = "ocne_" + filepath.Base(fpath)
	options.file = file
	failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Args:    &options,
			Message: "Uploading image to object storage",
			WaitFunction: func(uIface interface{}) error {
				uo, _ := uIface.(*UploadOptions)
				return oci.UploadObject(uo.BucketName, options.filename, uo.size, uo.file, nil, ociConfig)
			},
		},
	})
	if failed {
		return "", "", fmt.Errorf("Failed to upload image to object storage")
	}

	return oci.ImportImage(options.ImageName, options.KubernetesVersion, options.ImageArchitecture, options.compartmentId, options.BucketName, options.filename, ociConfig)
}

// EnsureImageDetails sets important configuration options for the custom image.
// In particular, it sets the image schema to allow EFI and sets the image shapes
// to match the architecture.
func EnsureImageDetails(compartmentId string, imageId string, arch string, ociConfig common.ConfigurationProvider) error {
	// Set schema.  compartmentId is set by UploadAsync
	if err := oci.CreateEFIImageSchema(compartmentId, imageId, ociConfig); err != nil {
		return err
	}

	if err := oci.EnsureCompatibleImageShapes(imageId, arch, ociConfig); err != nil {
		return err
	}
	return nil
}

// UploadOci uploads a boot image to an OCI bucket and imports
// it as a custom compute image.
func UploadOci(options UploadOptions) error {
	// TODO, inject OCIProfile into UploadOption
	ociConfig, _ := oci.GetOCIConfig(types.OCIProfile{})
	err := setCompartmentId(&options, ociConfig)
	if err != nil {
		return err
	}

	imageId, workRequestId, err := UploadAsync(options, ociConfig)
	if err != nil {
		return err
	}

	if workRequestId == "" {
		log.Infof("Image OCID is %s", imageId)
		return nil
	}

	// Wait for the operation to complete
	if err = oci.WaitForWorkRequest(workRequestId, "Importing compute image", ociConfig); err != nil {
		return err
	}

	// Set schema.  compartmentId is set by UploadAsync
	if err = EnsureImageDetails(options.compartmentId, imageId, options.ImageArchitecture, ociConfig); err != nil {
		return err
	}

	log.Infof("Image OCID is %s", imageId)

	return nil
}

func UploadOstree(options UploadOptions) error {
	return image.Copy(fmt.Sprintf("oci-archive:%s", options.ImagePath), options.Destination, "", copy.CopySystemImage)
}

var providers map[string]func(UploadOptions) error = map[string]func(UploadOptions) error{
	ProviderTypeOCI:    UploadOci,
	ProviderTypeOstree: UploadOstree,
}

// Upload uploads a VM image to OCI Object storage and then imports
// it into a compute image.
func Upload(options UploadOptions) error {
	pf, ok := providers[options.ProviderType]
	if !ok {
		return fmt.Errorf("%s is not a supported provider", options.ProviderType)
	}

	return pf(options)
}
