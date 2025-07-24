// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/filesystem"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/file"
	contimg "github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/disk"

	log "github.com/sirupsen/logrus"
)

type InfoOptions struct {
	Architecture string
	File         string
	Label        string
	Recursive    bool
	Path         string
}

func Info(startConfig *types.Config, clusterConfig *types.ClusterConfig, options InfoOptions) error {
	var imgPath string

	if options.File == "" {
		tmpPath, err := file.CreateOcneTempDir("image-info")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpPath)

		tarStream, closer, err := contimg.EnsureBaseQcow2Image(clusterConfig.BootVolumeContainerImage, options.Architecture)
		if err != nil {
			return err
		}
		defer closer()

		imgPath := filepath.Join(tmpPath, "boot.qcow2")
		err = writeFile(tarStream, imgPath)
		if err != nil {
			return err
		}
	} else {
		imgPath = options.File
	}

	qcowImg, err := disk.OpenQcow2(imgPath)
	if err != nil {
		return err
	}

	log.Debugf("Info: %s", qcowImg.Info())

	disk := qcowImg.GetDisk()

	partTable, err := qcowImg.GetPartitionTable()
	if err != nil {
		return err
	}

	if options.Path == "" {
		stat, _ := qcowImg.Stat()
		log.Infof("Image: %s", clusterConfig.BootVolumeContainerImage)
		log.Infof("Size: %s", util.HumanReadableSize(uint64(stat.Size())))
		log.Infof("Logical Block Size: %d", disk.LogicalBlocksize)
		log.Infof("PhysicalBlockSize: %d", disk.PhysicalBlocksize)
		log.Infof("Partition Table: %s", partTable.UUID())
	}

	for i, pt := range partTable.GetPartitions() {
		gptPart, ok := pt.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("Parition %d is not a GPT partition", i)
		}

		if !(options.Label == "" || options.Label == gptPart.Name) {
			continue
		}


		if options.Path == "" {
			log.Infof("\t%d", i)
			log.Infof("\t\tLabel: %s", gptPart.Name)
			log.Infof("\t\tGUID: %s", gptPart.GUID)
			log.Infof("\t\tType: %s", gptPart.Type)
			log.Infof("\t\tSize: %s", util.HumanReadableSize(gptPart.Size))
			log.Infof("\t\tExtents: %d to %d", gptPart.Start, gptPart.End)
			continue
		}

		thefs, err := qcowImg.GetFilesystem(i)
		if err != nil {
			log.Warnf("EFI partition %s was not a valid filesystem: %v", gptPart.Name, err)
			continue
		}

		printPath(thefs, nil, options.Path, options.Recursive)
	}

	return nil
}

func printPath(thefs filesystem.FileSystem, entries []fs.FileInfo, path string, recursive bool) error {
	var err error

	// Special case the root directory because it cannot
	// be split into dirname and filename
	if path == "/" {
		entries, err := thefs.ReadDir(path)
		if err != nil {
			return fmt.Errorf("could not read %s: %v", path, err)
		}

		for _, fi := range entries {
			fname := filepath.Join(path, fi.Name())
			fmt.Printf("%s\n", fname)

			if recursive && fi.IsDir() {
				printPath(thefs, entries, fname, recursive)
			}
		}
		return nil
	}

	dirname, fname := filepath.Split(path)
	log.Debugf("Checking %s %s %s", path, dirname, fname)

	if entries == nil {
		log.Debugf("Getting entries")
		entries, err = thefs.ReadDir(dirname)
		if err != nil {
			log.Errorf("Could not read %s: %s", path, err)
		}
	}

	for _, fi := range entries {
		if fname != fi.Name() {
			log.Debugf("skipping %s %s", fname, fi.Name())
			continue
		}


		if !fi.IsDir() {
			rdr, err := thefs.OpenFile(path, 0)
			if err != nil {
				return err
			}

			contents, err := io.ReadAll(rdr)
			if err != nil {
				return err
			}

			fmt.Printf("%s", contents)
			return nil
		}

		children, err := thefs.ReadDir(path)
		if err != nil {
			return err
		}

		for _, cfi := range children {
			if cfi.Name() == "." || cfi.Name() == ".." {
				continue
			}

			log.Debugf("Joining %s %s", path, cfi.Name())
			fmt.Printf("%s\n", filepath.Join(path, cfi.Name()))
			if cfi.IsDir()  && recursive {
				printPath(thefs, children, filepath.Join(path, cfi.Name()), recursive)
			}
		}

		break
	}

	return nil
}

func writeFile(reader io.Reader, filePath string) error {
	w, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}

	return nil
}


