// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"fmt"
	"io"
	"io/fs"

	log "github.com/sirupsen/logrus"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"
)

func MakeISO9660(path string, size int64) (*disk.Disk, filesystem.FileSystem, error) {
	bkend, err := file.CreateFromPath(path, size)
	if err != nil {
		return nil, nil, err
	}

	oDisk, err := diskfs.OpenBackend(bkend)
	if err != nil {
		return nil, nil, err
	}

	ofs, err := oDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition: 0,
		FSType: filesystem.TypeISO9660,
		VolumeLabel: "OCK",
	})
	if err != nil {
		return nil, nil, err
	}
	return oDisk, ofs, nil
}

func MakeSquashfs(path string, size int64) (*disk.Disk, *squashfs.FileSystem, error) {
	bkend, err := file.CreateFromPath(path, size)
	if err != nil {
		return nil, nil, err
	}

	oDisk, err := diskfs.OpenBackend(bkend, diskfs.WithSectorSize(diskfs.SectorSize4k))
	if err != nil {
		return nil, nil, err
	}

	ofs, err := oDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition: 0,
		FSType: filesystem.TypeSquashfs,
		VolumeLabel: "OCK",
	})
	if err != nil {
		return nil, nil, err
	}
	oSquash, ok := ofs.(*squashfs.FileSystem)
	if !ok {
		// This should never happen, and indicates a bug in go-diskfs
		return nil, nil, fmt.Errorf("Squashfs isn't a squashfs")
	}
	return oDisk, oSquash, nil
}

func CopyFS(inFs fs.FS, outFs filesystem.FileSystem) error {
	err := fs.WalkDir(inFs, "/", func(path string, d fs.DirEntry, err error) error {
		// Propagate any error that has occurred
		if err != nil {
			return err
		}
		log.Debugf("Processing %s", path)
		return nil
	})
	return err
}

func CopyFilesystem(inFs filesystem.FileSystem, outFs filesystem.FileSystem) error {
	walkFs := filesystem.FS(inFs)
	return CopyFS(walkFs, outFs)
}


func FindFilesInFilesystem(src filesystem.FileSystem, files []string) (map[string][]byte, error) {
	ret := map[string][]byte{}

	for _, file := range files {
		f, err := src.OpenFile(file, 0)
		if err != nil {
			return nil, err
		}

		contents, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		ret[file] = contents
	}

	return ret, nil
}
