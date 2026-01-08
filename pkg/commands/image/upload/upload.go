// Copyright (c) 2024, 2026, Oracle and/or its affiliates.
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
	"strings"

	"github.com/containers/image/v5/copy"
	"github.com/oracle-cne/ocne/cmd/flags"
	file2 "github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	log "github.com/sirupsen/logrus"
)

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
	// Hack to determine if OCI config is for a PCA.  PCA will only have one region.
	regions, err := oci.ListRegions(options.Profile)
	if err != nil {
		return "", "", err
	}
	if len(regions) == 1 {
		options.PCA = true
	}

	err = setCompartmentId(&options)
	if err != nil {
		return "", "", err
	}

	qcow2Image, err := file2.AbsDir(options.ImagePath)
	if err != nil {
		return "", "", err
	}

	file, err := os.Open(qcow2Image)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	// Create the image capabilities file
	capabilitiesFileSpec := getImageCapabilitiesFileSpec(qcow2Image)
	if err := createImageCapabilitiesFile(capabilitiesFileSpec, options.ImageArchitecture, options.PCA); err != nil {
		return "", "", err
	}

	// Create tarball
	tarballName := getTarballName(qcow2Image)
	if err := createTarballFile(qcow2Image, capabilitiesFileSpec, tarballName); err != nil {
		return "", "", err
	}

	// Upload the tarball
	err = uploadTarballFile(tarballName, &options)
	if err != nil {
		return "", "", err
	}

	return oci.ImportImage(options.ImageName, options.KubernetesVersion, options.ImageArchitecture, options.compartmentId, options.ClusterConfig.Providers.Oci.ImageBucket, options.filename, options.Profile)
}

func uploadTarballFile(tarballName string, options *UploadOptions) error {
	tarballFile, err := os.Open(tarballName)
	if err != nil {
		return err
	}
	defer tarballFile.Close()
	stat, err := tarballFile.Stat()
	if err != nil {
		return err
	}
	options.size = stat.Size()
	options.filename = "ocne_" + filepath.Base(tarballName)
	options.file = tarballFile
	failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Args:    options,
			Message: fmt.Sprintf("Uploading %s of size %d bytes to object storage", options.filename, options.size),
			WaitFunction: func(uIface interface{}) error {
				uo, _ := uIface.(*UploadOptions)
				return oci.UploadObject(uo.ClusterConfig.Providers.Oci.ImageBucket, options.filename, uo.ClusterConfig.Providers.Oci.Profile, uo.size, uo.file, nil)
			},
		},
	})
	if failed {
		return fmt.Errorf("failed to upload %s to object storage", options.filename)
	}
	return nil
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

	log.Infof("Image OCID is %s", imageId)

	return nil
}

func UploadOstree(options UploadOptions) error {
	return image.Copy(fmt.Sprintf("oci-archive:%s", options.ImagePath), options.Destination, "", copy.CopySystemImage)
}

var providers = map[string]func(UploadOptions) error{
	flags.ProviderTypeOCI:    UploadOci,
	flags.ProviderTypeOstree: UploadOstree,
	flags.ProviderTypeOlvm:   UploadOlvm,
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

// createTarballFile - Create a .tar.gz of the input file
func createTarballFile(qcow2Image string, capabilitiesFileSpec string, archiveName string) error {
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

	// List of files to add
	files := []string{qcow2Image, capabilitiesFileSpec}

	for _, filename := range files {
		if err := addFileToTarWriter(filename, tw); err != nil {
			return err
		}
	}

	log.Infof("Created archive: %s", archiveName)
	return nil
}

func addFileToTarWriter(filename string, tw *tar.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Rename files based on what PCA requires
	name := info.Name()
	if strings.HasSuffix(name, ".oci") {
		name = "output.QCOW2"
	} else {
		name = "image_metadata.json"
	}

	header := &tar.Header{
		Name:    name, // Entry name in archive
		Size:    info.Size(),
		Mode:    int64(info.Mode()), // File permissions
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// createImageCapabilitiesFile - create an image capabilities JSON file based on the architecture passed in
func createImageCapabilitiesFile(filePath string, imageArchitecture string, isPCA bool) error {
	capabilities := oci.NewImageCapability(oci.ImageArch(imageArchitecture), isPCA)

	// Marshal the struct to JSON
	data, err := json.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON data to a file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	log.Infof("Created file %s", filePath)

	return nil
}

func getTarballName(filePath string) string {
	return fmt.Sprintf("%s.upload.oci", strings.TrimSuffix(filePath, ".oci"))
}

func getImageCapabilitiesFileSpec(filePath string) string {
	return fmt.Sprintf("%s.json", filePath)
}
