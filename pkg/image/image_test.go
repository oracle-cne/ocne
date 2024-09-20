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

// TestImage is a series of tests that interact with real container images.
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

func TestSplitImage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testName      string
		testImage     string
		tag           string
		digest        string
		imageOnly     string
		expectedError bool
	}{
		{"test no tag non-canonical", "ock", "", "", "ock", false},
		{"test no tag non-canonical 2", "olcne/ock", "", "", "olcne/ock", false},
		{"test no tag with canonical", "container-registry.oracle.com/ock", "", "", "container-registry.oracle.com/ock", false},
		{"test no tag with canonical 2", "container-registry.oracle.com/olcne/ock", "", "", "container-registry.oracle.com/olcne/ock", false},
		{"test non-canonical with latest", "ock:latest", "latest", "", "ock", false},
		{"test non-canonical with latest 2", "olcne/ock:latest", "latest", "", "olcne/ock", false},
		{"test canonical with latest", "container-registry.oracle.com/ock:latest", "latest", "", "container-registry.oracle.com/ock", false},
		{"test canonical with latest 2", "container-registry.oracle.com/olcne/ock:latest", "latest", "", "container-registry.oracle.com/olcne/ock", false},
		{"test non-canonical with tag", "ock:olcne-65513-v1.2678", "olcne-65513-v1.2678", "", "ock", false},
		{"test non-canonical with tag 2", "olcne/ock:long-tag-example-7715", "long-tag-example-7715", "", "olcne/ock", false},
		{"test canonical with tag", "container-registry.oracle.com/ock:olcne-65513-v1.2678", "olcne-65513-v1.2678", "", "container-registry.oracle.com/ock", false},
		{"test canonical with tag 2", "container-registry.oracle.com/olcne/ock:long-tag-example-7715", "long-tag-example-7715", "", "container-registry.oracle.com/olcne/ock", false},
		{"test non-canonical with digest", "ock@sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "", "sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "ock", false},
		{"test non-canonical with digest 2", "olcne/ock@sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "", "sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "olcne/ock", false},
		{"test canonical with digest", "container-registry.oracle.com/ock@sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "", "sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "container-registry.oracle.com/ock", false},
		{"test canonical with digest 2", "container-registry.oracle.com/olcne/ock@sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "", "sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "container-registry.oracle.com/olcne/ock", false},
		{"test digest and tag in one image", "container-registry.oracle.com/ock:sometaghere@sha256:ce7395d681afeb6afd68e73a8044e4a965ede52cd0799de7f97198cca6ece7ed", "", "", "", true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			t.Parallel()
			tagRes, digestRes, imageOnlyRes, err := SplitImage(testCase.testImage)
			if testCase.expectedError {
				assert.Errorf(t, err, "expected test case to return an error but got result %s", imageOnlyRes)
				return
			}
			assert.NoError(t, err, "expected no error to be returned")
			assert.EqualValues(t, testCase.tag, tagRes, "returned an incorrect tag")
			assert.EqualValues(t, testCase.digest, digestRes, "returned an incorrect digest")
			assert.EqualValues(t, testCase.imageOnly, imageOnlyRes, "image result without tag and digest is wrong")
		})
	}

}
