// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmdutil

import (
	"fmt"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/config"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/image"
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

	// Be as friendly as can be
	img, err := image.MakeOstreeReference(cc.OsRegistry)
	if err != nil {
		return nil, nil, err
	}
	cc.OsRegistry = img

	return &ndf, &cc, nil
}

// EnsureBootImageVersion appends an image tag consisting of the Kubernetes version ,if the image string does not currently have a tag.
// It returns the updated image string.
func EnsureBootImageVersion(kubeVersion string, imageForContainer string) (string, error) {
	// if the user already specified a tag at the end of the image, use that tag and return
	imgInfo, err := image.SplitImage(imageForContainer)
	if err != nil {
		return imageForContainer, err
	}

	// Since the early days, the cluster cache has stored the post-processed
	// value for BootVolumeContainerImage.  That is, the value that is
	// returned by this function.  The value will always contain a tag.
	// Often, like during an update, the tag has to change.  But sometimes
	// it does not, such as when a non-standard tag or sha is specified.
	// That creates a bind where the tag needs to be changed but it's not
	// possible to tell if a specific tag was configured or the tag came
	// from this functions auto-tagging facility.  To handle both cases,
	// see if the tag is a typical OCK tag (read: <major>.<minor>).  If it
	// is, then strip the tag.  Otherwise, preserve the tag.  That will
	// allow updates to the tag when appropriate without destroying the
	// ability to specify a custom tag/digest.  There is one corner case.
	// A cluster configuration may have explicity configured a standard
	// tag.  This logic will strip that tag in leiu of a new one.  The
	// problem only applies to times when Kubernetes is being updated to
	// another minor version.  It is diffult to imagine that happening in
	// the real world.  At some point you've gone out of your way to make
	// your own life hard...
	if imgInfo.Tag != "" {
		// Ignore any errors from this call.  Many custom tags are
		// not going to be valid semantic versions.
		ok, _ := versions.IsSupportedKubernetesVersion(imgInfo.Tag)
		if ok {
			imgInfo.Tag = ""

			components := strings.Split(imageForContainer, ":")
			if len(components) < 2 {
				return "", fmt.Errorf("invalid image %s", imageForContainer)
			}

			imageForContainer = strings.Join(components[:len(components)-1], ":")
		}
	}

	if imgInfo.Tag == "" && imgInfo.Digest == "" {
		// if the version contains a "v" prefix, strip it
		ver := strings.TrimPrefix(kubeVersion, "v")
		// add the tag to the image string
		return imageForContainer + ":" + ver, nil
	}
	return imageForContainer, nil
}
