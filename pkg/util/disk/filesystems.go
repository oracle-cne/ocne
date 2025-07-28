// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"

	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util"
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

type fsCopy struct {
	totalBytes   uint64
	writtenBytes uint64
	lastError    error
}

func CopyFS(inFs fs.FS, outFs filesystem.FileSystem, bytes uint64) error {
	var waitFunc func(interface{})string
	var waitMsg string

	if bytes == 0 {
		waitMsg = "Copying filesystem"
	} else {
		waitFunc = func(fIface interface{})string{
			f, _ := fIface.(*fsCopy)
			percentComplete := float32(f.writtenBytes * 100) / float32(f.totalBytes)
			return fmt.Sprintf("Copying filesystem: %s/%s %s", util.HumanReadableSize(f.writtenBytes), util.HumanReadableSize(f.totalBytes), logutils.ProgressBar(percentComplete))
		}
	}

	fsc := &fsCopy{
		totalBytes: bytes,
	}

	debugFile, err := os.Create("debug.txt")
	if err != nil {
		return err
	}

	failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			MessageFunction: waitFunc,
			Args: fsc,
			WaitFunction: func(fIface interface{})error{
				f, _ := fIface.(*fsCopy)
				return fs.WalkDir(inFs, "/", func(path string, d fs.DirEntry, err error)error{
					if err != nil {
						f.lastError = err
						return err
					}

					inf, err := d.Info()
					if err != nil {
						return nil
					}

					fmt.Fprintf(debugFile, "%d %s -- %v -- %v\n", inf.Size(), path, inf.Mode(), inf.ModTime())

					f.writtenBytes = f.writtenBytes + uint64(inf.Size())
					return nil
				})
			},
		},
	})

	debugFile.Close()

	if failed {
		return fsc.lastError
	}
	return nil
}

func CopyFilesystem(inFs filesystem.FileSystem, outFs filesystem.FileSystem, bytes uint64) error {
	walkFs := filesystem.FS(inFs)
	return CopyFS(walkFs, outFs, bytes)
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
