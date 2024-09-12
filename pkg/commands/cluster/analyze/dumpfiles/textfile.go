// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dumpfiles

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ReadClusterWideTextFile reads cluster-wide text files.
func ReadClusterWideTextFile(clusterWideDir string, fileName string) (string, error) {
	var rObj string
	if err := readTextFromDirTree(clusterWideDir, fileName, func(_ string, text string) {
		rObj = text
	}); err != nil {
		return "", err
	}
	return rObj, nil
}

// ReadTextFiles reads namespaced or node-specific text files.
// The unmarshalled objects are then put into a map where the namespace is the key.
func ReadTextFiles(rootDir string, fileName string) (map[string]string, error) {
	// Read the json from each namespace directory into a map resource list then put the list into
	// the map, indexed by namespace
	rMap := make(map[string]string)
	if err := readTextFromDirTree(rootDir, fileName, func(nameSpace string, text string) {
		rMap[nameSpace] = text
	}); err != nil {
		return nil, err
	}
	return rMap, nil
}

// readTextFromDirTree read matching Text files in a directory tree, including files in all nested subdirectories
func readTextFromDirTree(rootDir string, targetFileName string, f func(parentDir string, text string)) error {
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil
	}
	err := filepath.WalkDir(rootDir,
		// Sanitize each file
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if dirEntry.IsDir() {
				return nil // walk into this dir
			}
			if dirEntry.Name() != targetFileName {
				return nil
			}
			text, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			parent := filepath.Base(filepath.Dir(path))
			f(parent, string(text))
			return nil
		})
	return err
}
