// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"errors"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"github.com/oracle-cne/ocne/pkg/util/strutil"
	"regexp"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

type olvmData struct {
	Config                     *types.Config
	ClusterConfig              *types.ClusterConfig
	ExtraConfigControlPlane    string
	ExtraConfigWorker          string
	KubeVersions               *versions.KubernetesVersions
	VolumePluginDir            string
	CipherSuite                string
	PodSubnetCidrBlocks        []string
	ServiceSubnetCidrBlocks    []string
	ControlPlaneIPV4CidrBlocks []string
	ControlPlaneIPV6CidrBlocks []string
	WorkerIPV4CidrBlocks       []string
	WorkerIPV6CidrBlocks       []string
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
	olvm := &clusterConfig.Providers.Olvm
	return util.TemplateToStringWithFuncs(string(tmplBytes), &olvmData{
		Config:                     config,
		ClusterConfig:              clusterConfig,
		ExtraConfigControlPlane:    cpIgn,
		ExtraConfigWorker:          workerIgn,
		KubeVersions:               &kubeVer,
		VolumePluginDir:            ignition.VolumePluginDir,
		CipherSuite:                clusterConfig.CipherSuites,
		PodSubnetCidrBlocks:        strutil.SplitAndTrim(clusterConfig.PodSubnet, ","),
		ServiceSubnetCidrBlocks:    strutil.SplitAndTrim(clusterConfig.ServiceSubnet, ","),
		ControlPlaneIPV4CidrBlocks: strutil.SplitAndTrim(olvm.ControlPlaneMachine.VirtualMachine.Network.IPV4.CIDRBlocks, ","),
		ControlPlaneIPV6CidrBlocks: strutil.SplitAndTrim(olvm.ControlPlaneMachine.VirtualMachine.Network.IPV6.CIDRBlocks, ","),
		WorkerIPV4CidrBlocks:       strutil.SplitAndTrim(olvm.WorkerMachine.VirtualMachine.Network.IPV4.CIDRBlocks, ","),
		WorkerIPV6CidrBlocks:       strutil.SplitAndTrim(olvm.WorkerMachine.VirtualMachine.Network.IPV6.CIDRBlocks, ","),
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
