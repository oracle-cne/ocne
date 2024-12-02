// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"errors"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"regexp"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/oci"
)

type olvmData struct {
	Config          *types.Config
	ClusterConfig   *types.ClusterConfig
	ExtraConfig     string
	KubeVersions    *versions.KubernetesVersions
	VolumePluginDir string
	CipherSuite     string
}

func imageFromShape(shape string, imgs *types.OciImageSet) string {
	arch := oci.ArchitectureFromShape(shape)
	if arch == "arm64" {
		return imgs.Arm64
	}
	return imgs.Amd64
}

func GetOlvmTemplate(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
	tmplBytes, err := template.ReadTemplate("capi-olvm.yaml")

	if err != nil {
		return "", err
	}

	if clusterConfig.ControlPlaneNodes%2 == 0 {
		return "", errors.New("the number of control plane nodes must be odd")
	}

	// Get the Kubernetes version configuration
	kubeVer, err := versions.GetKubernetesVersions(clusterConfig.KubeVersion)
	if err != nil {
		return "", err
	}

	// Build up the extra ignition structures
	ign, err := getExtraIgnition(config, clusterConfig)
	if err != nil {
		return "", err
	}

	return util.TemplateToStringWithFuncs(string(tmplBytes), &olvmData{
		Config:          config,
		ClusterConfig:   clusterConfig,
		ExtraConfig:     ign,
		KubeVersions:    &kubeVer,
		VolumePluginDir: ignition.VolumePluginDir,
		CipherSuite:     clusterConfig.CipherSuites,
	}, nil)
}

// ValidateClusterResources performs basic validation on cluster resources.
func ValidateClusterResources(clusterResources string) error {
	// validate that image OCIDs are not empty and have the correct prefix
	imageRegex := regexp.MustCompile(`imageId:(.*)`)

	matches := imageRegex.FindAllStringSubmatch(clusterResources, -1)
	for _, match := range matches {
		ocid := strings.Trim(match[1], `" `)
		if len(ocid) == 0 || !strings.HasPrefix(ocid, "ocid1.image") {
			return fmt.Errorf("Image ids in cluster resources must be valid OCI image OCIDs")
		}
	}
	return nil
}
