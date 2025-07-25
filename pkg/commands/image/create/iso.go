// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util/disk"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
)

func CreateIso(startConfig *otypes.Config, clusterConfig *otypes.ClusterConfig, options CreateOptions) error {
	// Get the image
	tmpPath, err := file.CreateOcneTempDir(tempDir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpPath)

	// Get the tarstream of the boot qcow2 image
	log.Infof("Getting local boot image for architecture: %s", options.Architecture)
	tarStream, closer, err := image.EnsureBaseQcow2Image(clusterConfig.BootVolumeContainerImage, options.Architecture)
	if err != nil {
		return err
	}
	defer closer()

	// Write the local image. e.g. ~/.ocne/tmp/create-images.xyz/boot.oci
	localImagePath := filepath.Join(tmpPath, localVMImage+".oci")
	err = writeFile(tarStream, localImagePath)
	if err != nil {
		return err
	}

	qcowImg, err := disk.OpenQcow2(localImagePath)
	if err != nil {
		return err
	}

	// Get filesystems
	var efiFs filesystem.FileSystem
	var rootFs filesystem.FileSystem
	var bootFs filesystem.FileSystem

	partTable, err := qcowImg.GetPartitionTable()
	if err != nil {
		return err
	}

	for i, pt := range partTable.GetPartitions() {
		gptPart, ok := pt.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("Parition %d is not a GPT partition", i)
		}

		thefs, err := qcowImg.GetFilesystem(i)
		if err != nil {
			return err
		}

		if strings.Contains(gptPart.Name, "EFI") {
			efiFs = thefs
		} else if gptPart.Name == "root" {
			rootFs = thefs
		} else if gptPart.Name == "boot" {
			bootFs = thefs
		}
	}
	log.Debugf("Filesystems: %v %v %v", efiFs, rootFs, bootFs)


	// Dump the root filesystem into a squashfs
	tmpDir, err := os.MkdirTemp("", "ocneIso")
	if err != nil {
		return err
	}

	rootSquashDiskPath := filepath.Join(tmpDir, "rootSquash")

	defer os.RemoveAll(tmpDir)
	rootSquashDisk, rootSquashFs, err := disk.MakeSquashfs(rootSquashDiskPath, 8 * 1024 * 1024 * 1024)
	if err != nil {
		return err
	}

	err = disk.CopyFilesystem(rootFs, rootSquashFs)
	if err != nil {
		return nil
	}


	err = rootSquashFs.Finalize(squashfs.FinalizeOptions{})
	if err != nil {
		return err
	}

	err = rootSquashDisk.Close()
	if err != nil {
		return err
	}

	return nil
}
