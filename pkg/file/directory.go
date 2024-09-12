// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package file

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle-cne/ocne/pkg/constants"
)

// CreateOcneTempDir creates a temp dir to hold the manifests files
func CreateOcneTempDir(nameOfTempDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use homedir for temp files since root might own temp dir on OSX and we get
	// errors trying to create temp files
	hidden := filepath.Join(home, ".ocne/tmp")
	err = os.MkdirAll(hidden, 0700)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	topDir, err := os.MkdirTemp(hidden, nameOfTempDir)
	if err != nil {
		return "", err
	}
	return topDir, nil
}

// GetOcneDir gets the absolute path of ~/.ocne dir
func GetOcneDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, constants.UserConfigDir), nil
}

// EnsureOcneDir ensures that ~/.ocne dir exists
func EnsureOcneDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, constants.UserConfigDir)
	err = os.MkdirAll(dir, 0700)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	return dir, err
}

// EnsureOcneImagesDir ensures that ~/.ocne/images dir exists
func EnsureOcneImagesDir() (string, error) {
	ocneDir, err := EnsureOcneDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(ocneDir, constants.UserImageCacheDir)
	err = os.MkdirAll(dir, 0700)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	return dir, err
}

// AbsDir returns the absolute director of the string, expanding ~/ prefix if needed.
func AbsDir(dir string) (string, error) {
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(dir, "~")), nil
	}

	return filepath.Abs(dir)
}
