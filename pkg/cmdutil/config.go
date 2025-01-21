// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmdutil

import (
	"strings"

	"github.com/oracle-cne/ocne/pkg/config"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/image"
)

func GetFullConfig(clusterConfig *types.ClusterConfig, clusterConfigPath string) (*types.ClusterConfig, error) {
	// Read the cluster config file, if it was specified
	var cc types.ClusterConfig
	var err error
	// Merge the compiled config and the defaults.yaml
	df, err := config.GetDefaultConfig()
	if err != nil {
		return nil, err
	}
	// Read the cluster config file, if it was specified
	// Overlay it with the resulting defaulting configuration
	if clusterConfigPath != "" {
		ncc, err := config.ParseClusterConfigFile(clusterConfigPath)
		if err != nil {
			return nil, err
		}
		cc = types.OverlayConfig(ncc, df)
	} else {
		cc = types.OverlayConfig(&cc, df)
	}

	ncc := types.MergeClusterConfig(&cc, clusterConfig)
	// Be as friendly as can be
	img, err := image.MakeOstreeReference(*ncc.OsRegistry)
	if err != nil {
		return nil, err
	}
	*ncc.OsRegistry = img

	return &ncc, nil
}

// EnsureBootImageVersion appends an image tag consisting of the Kubernetes version ,if the image string does not currently have a tag.
// It returns the updated image string.
func EnsureBootImageVersion(kubeVersion string, imageForContainer string) (string, error) {
	// if the user already specified a tag at the end of the image, use that tag and return
	imgInfo, err := image.SplitImage(imageForContainer)
	if err != nil {
		return imageForContainer, err
	}
	if imgInfo.Tag == "" && imgInfo.Digest == "" {
		// if the version contains a "v" prefix, strip it
		ver := strings.TrimPrefix(kubeVersion, "v")
		// add the tag to the image string
		return imageForContainer + ":" + ver, nil
	}
	return imageForContainer, nil
}
