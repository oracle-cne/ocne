// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/copy"
	file2 "github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	log "github.com/sirupsen/logrus"
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

	// Create tarball
	if err = compressFile(file, getTarballName(fpath)); err != nil {
		return "", "", err
	}

	// Create image capabilities file
	capabilitiesFile, err := imageCapabilitiesFile(getImageCapabilitiesName(fpath), options.ImageArchitecture)
	if err != nil {
		return "", "", err
	}
	log.Infof("created file %s", capabilitiesFile.Name())
	os.Exit(1)

	options.size = stat.Size()
	options.filename = "ocne_" + filepath.Base(fpath)
	options.file = file
	failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
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

func compressFile(file *os.File, archiveName string) error {
	var err error

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create archive for writing
	outFile, err := os.Create(archiveName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Wrap output in gzip and tar writers
	gw := gzip.NewWriter(outFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Create tar header from file info
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = file.Name() // Name in archive

	// Write header and file content
	if err = tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err = io.Copy(tw, file); err != nil {
		return err
	}
	log.Infof("Created archive: %s", archiveName)
	return nil
}

func imageCapabilitiesFile(filePath string, imageArchitecture string) (*os.File, error) {
	capabilities := oci.NewImageCapability(oci.ImageArch(imageArchitecture))

	// Marshal the struct to JSON
	data, err := json.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		return nil, err
	}

	// Write JSON data to a file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(data)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func getTarballName(filePath string) string {
	return fmt.Sprintf("%s.tar.gz", filePath)
}

func getImageCapabilitiesName(filePath string) string {
	return fmt.Sprintf("%s.json", filePath)
}
