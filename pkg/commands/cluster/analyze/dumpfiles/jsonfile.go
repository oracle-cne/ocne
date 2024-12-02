// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dumpfiles

import (
	"io/fs"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"slices"
)

// ReadClusterWideJSONOrYAMLFile reads cluster-wide json/yaml data files and un-marshals them into resources.
// For example, read nodes.json
func ReadClusterWideJSONOrYAMLFile[T any](clusterWideDir string, fileNames []string) (*T, error) {
	var rObj *T
	if err := readJSONOrYAMLFromDirTree[T](clusterWideDir, fileNames, func(_ string, obj *T) {
		rObj = obj
	}); err != nil {
		return nil, err
	}
	return rObj, nil
}

// ReadJSONOrYAMLFiles reads namespaced or node-specific json/yaml data files and unmarshals them into resources.
// The unmarshalled objects are then put into a map where the namespace is the key.
// For example, read pods.json in all namespaces.
func ReadJSONOrYAMLFiles[T any](rootDir string, fileNames []string) (map[string]T, error) {
	// Read the json from each namespace directory into a map resource list then put the list into
	// the map, indexed by namespace
	rMap := make(map[string]T)
	if err := readJSONOrYAMLFromDirTree[T](rootDir, fileNames, func(nameSpace string, obj *T) {
		rMap[nameSpace] = *obj
	}); err != nil {
		return nil, err
	}
	return rMap, nil
}

// readJSONorYAMLFromDirTree read matching JSON or YAML files in a directory tree, including files in all nested subdirectories
func readJSONOrYAMLFromDirTree[T any](rootDir string, targetFileNames []string, f func(parentDir string, obj *T)) error {
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
			if !slices.Contains(targetFileNames, dirEntry.Name()) {
				return nil
			}
			dataStr, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var obj T
			if err = yaml.Unmarshal([]byte(dataStr), &obj); err != nil {
				return err
			}

			parent := filepath.Base(filepath.Dir(path))
			f(parent, &obj)
			return nil
		})
	return err
}
