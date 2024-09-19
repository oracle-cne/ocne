// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmdutil

import (
	"github.com/Masterminds/semver/v3"
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
		mcc := types.MergeClusterConfig(ncc, clusterConfig)
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

// ensureBootImageVersion patches the boot image tag as necessary based on the desired version
// of Kubernetes. It returns the updated image string.
func EnsureBootImageVersion(requestedVersion string, image string) string {
	// if the user already specified a tag at the end of the image, use that tag and return
	parts := strings.Split(image, ":")
	_, err := semver.NewVersion(parts[len(parts)-1])
	if err == nil {
		return image
	}
	// if the version contains a "v" prefix, strip it
	ver := strings.TrimPrefix(requestedVersion, "v")
	// add the tag to the image string
	return image + ":" + ver
}
