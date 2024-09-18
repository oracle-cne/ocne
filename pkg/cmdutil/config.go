// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmdutil

import (
	image2 "github.com/oracle-cne/ocne/pkg/image"
	"strings"

	"github.com/oracle-cne/ocne/pkg/config"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

func GetFullConfig(defaultConfig *types.Config, clusterConfig *types.ClusterConfig, clusterConfigPath string) (*types.Config, *types.ClusterConfig, error) {
	// Read the cluster config file, if it was specified
	var err error
	if clusterConfigPath != "" {
		ncc, err := config.ParseClusterConfigFile(clusterConfigPath)
		if err != nil {
			return nil, nil, err
		}

		mcc := types.MergeClusterConfig(clusterConfig, ncc)
		clusterConfig = &mcc
	}
	df, err := config.GetDefaultConfig()
	if err != nil {
		return nil, nil, err
	}

	ndf := types.MergeConfig(df, defaultConfig)
	cc := types.OverlayConfig(clusterConfig, &ndf)

	return &ndf, &cc, nil
}

// EnsureBootImageVersion appends an image tag consisting of the Kubernetes version ,if the image string does not currently have a tag.
// It returns the updated image string.
func EnsureBootImageVersion(kubeVersion string, image string) string {
	// if the user already specified a tag at the end of the image, use that tag and return
	image, err := image2.WithoutTag(image)
	if err != nil {
		return image
	}
	// if the version contains a "v" prefix, strip it
	ver := strings.TrimPrefix(kubeVersion, "v")
	// add the tag to the image string
	return image + ":" + ver
}
