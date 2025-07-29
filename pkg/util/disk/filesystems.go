// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"

	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util"
)

type File struct {
	IsDir bool
	IsSymlink bool
	IsHardlink bool
	LinkTarget string
	Entries map[string]*File
	UID int
	GID int
	Mode os.FileMode
	Contents []byte
}

func (f *File) AddFile(path string, content []byte, fileUid int, fileGid int, fileMode os.FileMode, dirUid int, dirGid int, dirMode os.FileMode) *File {
	dirs := strings.Split(path, string(filepath.Separator))

	if f.Entries == nil {
		f.Entries = map[string]*File{}
	}

	// The last element of the path must be a file
	if len(dirs) == 1 {
		f.Entries[path] = &File{
			UID: fileUid,
			GID: fileGid,
			Mode: fileMode,
			Contents: content,
		}
		return f.Entries[path]
	}

	// If it's not a file, then it's a directory
	dir := dirs[0]
	d, ok := f.Entries[dir]
	if !ok {
		f.Entries[dir] = &File{
			IsDir: true,
			UID: dirUid,
			GID: dirGid,
			Mode: dirMode,
		}
		d = f.Entries[dir]
	}

	return d.AddFile(filepath.Join(dirs[1:]...), content, fileUid, fileGid, fileMode, dirUid, dirGid, dirMode)
}

func MakeISO9660(path string, size int64) (*disk.Disk, *iso9660.FileSystem, error) {
	bkend, err := file.CreateFromPath(path, size)
	if err != nil {
		return nil, nil, err
	}

	oDisk, err := diskfs.OpenBackend(bkend)
	if err != nil {
		return nil, nil, err
	}

	oDisk.LogicalBlocksize = 4096

	ofs, err := oDisk.CreateFilesystem(disk.FilesystemSpec{
		Partition: 0,
		FSType: filesystem.TypeISO9660,
		VolumeLabel: "OCK",
	})
	if err != nil {
		return nil, nil, err
	}
	oIso, ok := ofs.(*iso9660.FileSystem)
	if !ok {
		return nil, nil, fmt.Errorf("ISO-9660 filesystem is not an ISO-9660 filesystem")
	}
	return oDisk, oIso, nil
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

// AbsolutePath takes a path and converts it to the true
// path by resolving and traversing any symlinks.  If the
// target of a symlink does not exist, an error is returned.
//
// Loops are not handled, and will result in infinite recursion.
func RealPath(in filesystem.FileSystem, path string) (string, error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("searching virtual disk filesystems requires an absolute path")
	}

	realPath := string(filepath.Separator)
	for _, d := range strings.Split(path, string(filepath.Separator)) {
		if d == "" {
			continue
		}

		ents, err := in.ReadDir(realPath)
		if err != nil {
			return "", err
		}

		var fi fs.FileInfo
		for _, e := range ents {
			if e.Name() == d {
				fi = e
				break
			}
		}

		if fi == nil {
			return "", fmt.Errorf("Could not find %s", filepath.Join(realPath, d))
		}

		if (fi.Mode() & fs.ModeSymlink) != 0 {
			xfs, ok := in.(*XfsFilesystem)
			if !ok {
				return "", fmt.Errorf("symlink resolution is only supported for xfs filesystems")
			}

			// Resolve symlink.
			tgt, err := xfs.GetSymlinkTarget(fi)
			if err != nil {
				return "", err
			}

			if filepath.IsAbs(tgt) {
				realPath = tgt
			} else {
				realPath = filepath.Join(realPath, tgt)
			}
			realPath = filepath.Clean(realPath)

			// There may be symlinks in the resolved target path.
			realPath, err = RealPath(in, realPath)
			if err != nil {
				return "", err
			}
			//return "", fmt.Errorf("symlink")
		} else {
			realPath = filepath.Join(realPath, d)
		}
	}

	return realPath, nil
}

func CopyFS(inFs fs.FS, outFs filesystem.FileSystem, root string, bytes uint64) error {
	var waitFunc func(interface{})string
	var waitMsg string

	if root == "" {
		root = "/"
	}

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

	fmt.Println("Walking filesystem from", root)

	failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			MessageFunction: waitFunc,
			Args: fsc,
			WaitFunction: func(fIface interface{})error{
				f, _ := fIface.(*fsCopy)
				return fs.WalkDir(inFs, root, func(path string, d fs.DirEntry, err error)error{
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

func CopyFilesystem(inFs filesystem.FileSystem, outFs filesystem.FileSystem, root string, bytes uint64) error {
	walkFs := filesystem.FS(inFs)
	return CopyFS(walkFs, outFs, root, bytes)
}

func CopyFiles(outFs filesystem.FileSystem, tree map[string]*File, root string) error {
	for name, dst := range tree {
		path := filepath.Join(root, name)

		if dst.IsDir {
			err := outFs.Mkdir(path)
			if err != nil {
				return err
			}
		} else if dst.IsSymlink {
			err := outFs.Symlink(path, dst.LinkTarget)
			if err != nil {
				return err
			}
		} else if dst.IsHardlink {
			err := outFs.Link(path, dst.LinkTarget)
			if err != nil {
				return err
			}
		} else {
			f, err := outFs.OpenFile(path, os.O_CREATE | os.O_RDWR)
			if err != nil {
				return err
			}

			_, err = io.Copy(f, bytes.NewBuffer(dst.Contents))
			if err  != nil {
				return err
			}

			f.Close()
		}

		outFs.Chmod(path, dst.Mode)
		outFs.Chown(path, dst.UID, dst.GID)

		err := CopyFiles(outFs, dst.Entries, path)
		if err != nil {
			return err
		}

	}
	return nil
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

func GetFileInFilesystem(src filesystem.FileSystem, file string) ([]byte, error) {
	res, err := FindFilesInFilesystem(src, []string{file})
	if err != nil {
		return nil, err
	}

	return res[file], err
}
