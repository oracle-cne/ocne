// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"bytes"
	"compress/gzip"
	"maps"
	"path/filepath"
	"slices"

	"github.com/cavaliergopher/cpio"
)

type CpioHeader = cpio.Header

func MakeCpio(files map[*CpioHeader][]byte, compress bool) ([]byte, error) {
	dirHeaders := map[string]*cpio.Header{}

	// Cache any explicit directories so they are made
	// as specified.  The rest are auto-generated with
	// reasonable defaults.
	for h, _ := range files {
		if h.Mode.IsDir() {
			dirHeaders[h.Name] = h
		}
	}

	// Create any missing directories
	for h, _ := range files {
		dirName := filepath.Dir(h.Name)
		_, ok := dirHeaders[dirName]
		if ok {
			continue
		}

		dirHeaders[dirName] = &cpio.Header{
			Name: dirName,
			Mode: cpio.TypeDir | 0755,
			ModTime: h.ModTime,
		}
	}

	out := &bytes.Buffer{}
	cw := cpio.NewWriter(out)

	// write directories first, so that files can go in willy nilly.
	// Soring the paths lexicographically is cheeky, but a sub directory
	// will always have a longer name than its parent so it will
	// all work out.
	dirs := slices.Sorted(maps.Keys(dirHeaders))
	for _, d := range dirs {
		err := cw.WriteHeader(dirHeaders[d])
		if err != nil {
			return nil, err
		}
	}

	// With the directories written, it is possible to write
	// files in any order.  Skip directories because they are
	// already there
	for h, c := range files {
		if h.Mode.IsDir() {
			continue
		}

		nh := *h
		nh.Size = int64(len(c))

		err := cw.WriteHeader(&nh)
		if err != nil {
			return nil, err
		}

		if c != nil {
			_, err = cw.Write(c)
			if err != nil {
				return nil, err
			}
		}
	}

	cw.Close()

	if compress {
		cmpOut := &bytes.Buffer{}
		gzw := gzip.NewWriter(cmpOut)
		_, err := gzw.Write(out.Bytes())
		if err != nil {
			return nil, err
		}
		gzw.Close()
		out = cmpOut
	}

	return out.Bytes(), nil
}

func CpioEntry(path string, mode cpio.FileMode) *CpioHeader {
	return &cpio.Header{
		Name: path,
		Mode: mode,
	}
}

func CpioFile(path string, mode cpio.FileMode) *CpioHeader {
	return CpioEntry(path, (mode | cpio.TypeReg))
}

func CpioDirectory(path string, mode cpio.FileMode) *CpioHeader {
	return CpioEntry(path, (mode | cpio.TypeDir))
}

func CpioSymlink(path string, target string, mode cpio.FileMode) *CpioHeader {
	hdr := CpioEntry(path, (mode & cpio.TypeSymlink))
	hdr.Linkname = target
	return hdr
}
