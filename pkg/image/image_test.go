// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifest(t *testing.T) {
	ArchTest(t, "aarch64")
	ArchTest(t, "x86_64")
}

func TestImage(t *testing.T) {
	ArchTest(t, "aarch64")
}

// TestImage is a series of tests that interact with real container iamges.
// The format of this tests is non-standard because it requires a significant
// amount of harness, and the setup of the harness itself needs to be tested.
func ArchTest(t *testing.T, arch string) {
	// Override the default image store location to avoid polluting
	// the users environment.
	imageDirectoryPrefix = "./"
	imageName := "docker://container-registry.oracle.com/os/oraclelinux:8-slim"
	filePath := "usr/bin/bash"
	badPath := "fake/path"

	img, err := GetOrPull(imageName, arch)
	assert.NoError(t, err, "Could not pull test image: %v", err)
	assert.NotNil(t, img, "Image was nil")

	// Get the first layer out of the reference
	layers, err := GetImageLayers(img, arch)
	assert.NoError(t, err, "Could not get layers for image: %v", err)
	assert.NotNil(t, layers, "Layers was nil")

	// The OL 8 slim image should have only one layer, but it's still
	// worth checking that it's true so that if that assumption changes
	// it's easier to figure out why.
	assert.Equal(t, len(layers), 1, "Unexpected number of layers in %s: %d", imageName, len(layers))
	layerId := layers[0].Digest.Encoded()

	tarStream, closer, err := GetTarFromLayerById(img, layerId)
	assert.NoError(t, err, "Could not get directory from layer: %v", err)
	assert.NotNil(t, tarStream, "No tar stream found")

	// Get File from the tarchive
	err = AdvanceTarToPath(tarStream, filePath)
	assert.NoError(t, err, "Could not advance tar stream to input path: %v", err)
	closer()

	tarStream, closer, err = GetTarFromLayerById(img, layerId)
	err = AdvanceTarToPath(tarStream, badPath)
	assert.Error(t, err, "No error when looking for bad path in tar stream")
	closer()
}
