// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"fmt"
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
const ProviderTypeOlvm = "olvm"

func setCompartmentId(options *UploadOptions) error {
	// Compartment ID is already resolved.
	if options.compartmentId != "" {
		return nil
	}

	compartmentId, err := oci.GetCompartmentId(options.ClusterConfig.Providers.Oci.Compartment, options.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		log.Debugf("oci.GetCompartmentId failed for compartment: %s", options.ClusterConfig.Providers.Oci.Compartment)
		return err
	}

	options.compartmentId = compartmentId
	return nil
}

// UploadAsync uploads a VM image to object storage and then begins
// the import process.  A work request is returned for the import.
func UploadAsync(options UploadOptions) (string, string, error) {
	err := setCompartmentId(&options)
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
				return oci.UploadObject(uo.ClusterConfig.Providers.Oci.ImageBucket, options.filename, uo.ClusterConfig.Providers.Oci.Profile, uo.size, uo.file, nil)
			},
		},
	})
	if failed {
		return "", "", fmt.Errorf("Failed to upload image to object storage")
	}

	return oci.ImportImage(options.ImageName, options.KubernetesVersion, options.ImageArchitecture, options.compartmentId, options.ClusterConfig.Providers.Oci.ImageBucket, options.filename, options.Profile)
}

// EnsureImageDetails sets important configuration options for the custom image.
// In particular, it sets the image schema to allow EFI and sets the image shapes
// to match the architecture.
func EnsureImageDetails(compartmentId string, profile string, imageId string, arch string) error {
	// Set schema.  compartmentId is set by UploadAsync
	if err := oci.CreateEFIImageSchema(compartmentId, imageId, profile); err != nil {
		return err
	}

	if err := oci.EnsureCompatibleImageShapes(imageId, arch, profile); err != nil {
		return err
	}
	return nil
}

// UploadOci uploads a boot image to an OCI bucket and imports
// it as a custom compute image.
func UploadOci(options UploadOptions) error {
	err := setCompartmentId(&options)
	if err != nil {
		return err
	}

	imageId, workRequestId, err := UploadAsync(options)
	if err != nil {
		return err
	}

	if workRequestId == "" {
		log.Infof("Image OCID is %s", imageId)
		return nil
	}

	// Wait for the operation to complete
	if err = oci.WaitForWorkRequest(workRequestId, options.Profile, "Importing compute image"); err != nil {
		return err
	}

	// Set schema.  compartmentId is set by UploadAsync
	if err = EnsureImageDetails(options.compartmentId, options.Profile, imageId, options.ImageArchitecture); err != nil {
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
	ProviderTypeOlvm:   UploadOlvm,
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
