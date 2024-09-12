// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"os"
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
