// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/masahiro331/go-xfs-filesystem/xfs"
)

type XfsFilesystem struct {
	xfsfs *xfs.FileSystem
}

func GetXfsFilesystem(rdr io.ReaderAt, size int64) (filesystem.FileSystem, error) {
	xfsfs, err := xfs.NewFS(*io.NewSectionReader(rdr, 0, size), nil)
	if err != nil {
		return nil, err
	}
	return &XfsFilesystem{
		xfsfs: xfsfs,
	}, nil
}

func (fs *XfsFilesystem) GetSymlinkTarget(fi fs.FileInfo) (string, error) {
	return fs.xfsfs.GetSymlinkTarget(fi)
}

func (fs *XfsFilesystem) Free() uint64 {
	return uint64(fs.xfsfs.PrimaryAG.SuperBlock.BlockSize) * fs.xfsfs.PrimaryAG.SuperBlock.Fdblocks
}

func (fs *XfsFilesystem) Type() filesystem.Type {
	return 999
}

func (fs *XfsFilesystem) Mkdir(pathname string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Mknod(pathname string, mode uint32, dev int) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Link(oldpath, newpath string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Symlink(oldpath, newpath string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Chmod(name string, mode os.FileMode) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Chown(name string, uid, gid int) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) ReadDir(pathname string) ([]os.FileInfo, error) {
	dirents, err := fs.xfsfs.ReadDir(pathname)
	if err != nil {
		return nil, err
	}
	ret := make([]os.FileInfo, len(dirents), len(dirents))
	for i, de := range dirents {
		fi, err := de.Info()
		if err != nil {
			return nil, err
		}
		ret[i] = fi
	}

	return ret, nil
}

func (fs *XfsFilesystem) OpenFile(pathname string, flag int) (filesystem.File, error) {
	pathname = strings.TrimLeft(pathname, "/")

	file, err := fs.xfsfs.Open(pathname)
	if err != nil {
		return nil, err
	}

	return &xfsFile{
		file: file,
	}, nil
}

func (fs *XfsFilesystem) Rename(oldpath, newpath string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Remove(pathname string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Label() string {
	return "not implemented"
}

func (fs *XfsFilesystem) SetLabel(label string) error {
	return filesystem.ErrReadonlyFilesystem
}

func (fs *XfsFilesystem) Close() error {
	return fs.Close()
}

type xfsFile struct {
	file fs.File
}

func (xf *xfsFile) Stat() (fs.FileInfo, error) {
	return xf.file.Stat()
}

func (xf *xfsFile) Read(buf []byte) (int, error) {
	return xf.file.Read(buf)
}

func (xf *xfsFile) Close() error {
	return xf.file.Close()
}

func (xf *xfsFile) Write(buf []byte) (int, error) {
	return -1, filesystem.ErrReadonlyFilesystem
}

func (xf *xfsFile) Seek(offset int64, whence int) (int64, error) {
	return -1, filesystem.ErrNotImplemented
}
