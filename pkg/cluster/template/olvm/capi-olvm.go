// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"errors"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"github.com/oracle-cne/ocne/pkg/util/olvmutil"
	"github.com/oracle-cne/ocne/pkg/util/strutil"
	"regexp"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

type olvmData struct {
	Config                    *types.Config
	ClusterConfig             *types.ClusterConfig
	ExtraConfigControlPlane   string
	ExtraConfigWorker         string
	KubeVersions              *versions.KubernetesVersions
	VolumePluginDir           string
	CipherSuite               string
	PodSubnetCidrBlocks       []string
	ServiceSubnetCidrBlocks   []string
	ControlPlaneIPV4Addresses []string
	ControlPlaneIPV6Addresses []string
	WorkerIPV4Addresses       []string
	WorkerIPV6Addresses       []string
}

// GetOlvmTemplate renders the OLVM template that specifies the CAPI resources.
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

	// Get the default CA and Secret (which require cluster name)
	olvm := &clusterConfig.Providers.Olvm
	olvm.OlvmAPIServer.CAConfigMap = *olvmutil.CaConfigMapNsn(clusterConfig)
	olvm.OlvmAPIServer.CredentialsSecret = *olvmutil.CredSecretNsn(clusterConfig)

	// Get the CIDR blocks

	// Build up the extra ignition structures.  Internal LB for control plane only
	cpIgn, err := getExtraIgnition(config, clusterConfig, true)
	if err != nil {
		return "", err
	}
	workerIgn, err := getExtraIgnition(config, clusterConfig, false)
	if err != nil {
		return "", err
	}
	return util.TemplateToStringWithFuncs(string(tmplBytes), &olvmData{
		Config:                    config,
		ClusterConfig:             clusterConfig,
		ExtraConfigControlPlane:   cpIgn,
		ExtraConfigWorker:         workerIgn,
		KubeVersions:              &kubeVer,
		VolumePluginDir:           ignition.VolumePluginDir,
		CipherSuite:               clusterConfig.CipherSuites,
		PodSubnetCidrBlocks:       strutil.SplitAndTrim(clusterConfig.PodSubnet, ","),
		ServiceSubnetCidrBlocks:   strutil.SplitAndTrim(clusterConfig.ServiceSubnet, ","),
		ControlPlaneIPV4Addresses: strutil.SplitAndTrim(olvm.ControlPlaneMachine.VirtualMachine.Network.IPV4.IpAddresses, ","),
		ControlPlaneIPV6Addresses: strutil.SplitAndTrim(olvm.ControlPlaneMachine.VirtualMachine.Network.IPV6.IpAddresses, ","),
		WorkerIPV4Addresses:       strutil.SplitAndTrim(olvm.WorkerMachine.VirtualMachine.Network.IPV4.IpAddresses, ","),
		WorkerIPV6Addresses:       strutil.SplitAndTrim(olvm.WorkerMachine.VirtualMachine.Network.IPV6.IpAddresses, ","),
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
