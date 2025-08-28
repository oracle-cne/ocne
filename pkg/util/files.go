// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"bytes"
	"io/fs"
	"os"
	"time"
)

func FilesFromPath(path string) ([]string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// If the stat is a file, just return it
	if !fi.IsDir() {
		return []string{path}, nil
	}

	// If not, it must be a directory.  Return all the
	// regular files.
	dirents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, de := range dirents {
		if de.IsDir() {
			continue
		}

		ret = append(ret, de.Name())
	}

	return ret, nil
}

type MemoryFile struct {
	bytes.Buffer
	FileMode fs.FileMode
}

func NewMemoryFile(fileMode fs.FileMode, size int64) *MemoryFile {
	return &MemoryFile{
		Buffer: *bytes.NewBuffer(make([]byte, size)),
		FileMode: fileMode,
	}
}

func (mf *MemoryFile) Stat() (fs.FileInfo, error) {
	return mf, nil
}

func (mf *MemoryFile) Name() string {
	return "in-memory-file"
}

func (mf *MemoryFile) Size() int64{
	return int64(mf.Buffer.Cap())
}

func (mf *MemoryFile) Mode() fs.FileMode {
	return mf.FileMode
}

func (mf *MemoryFile) ModTime() time.Time {
	return time.Now()
}

func (mf *MemoryFile) IsDir() bool {
	return false
}

func (mf *MemoryFile) Sys() interface{} {
	return mf
}

func (mf *MemoryFile) Close() error {
	return nil
}
