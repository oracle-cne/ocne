// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"os"
	"path/filepath"

	"fmt"
	cmdconstants "github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"gopkg.in/yaml.v3"
)

// ParseConfig takes a yaml-encoded string and parses it
// into a Config structure.
func ParseConfig(in string) (*types.Config, error) {
	ret := &types.Config{}
	err := yaml.Unmarshal([]byte(in), ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ParseConfigFile takes the path to a file, reads the contents,
// and parses it into a Config structure.
func ParseConfigFile(configPath string) (*types.Config, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	conf, err := ParseConfig(string(configBytes))
	if err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %s", configPath, err.Error())
	}
	return conf, nil
}

// GetDefaultConfig returns the global default config.  It starts
// with a hard-coded set of defaults.  It then attempts to read a
// global overrides file.  If such a file is found, the entries in
// that file are merged into the hard-coded defaults.
func GetDefaultConfig() (*types.Config, error) {
	sessionURI := constants.SessionURI
	storagePool := constants.StoragePool
	network := constants.Network
	controlNodeMemory := constants.NodeMemory
	controlNodeStorage := constants.NodeStorage
	controlNodeCPUs := constants.NodeCPUs
	workerNodeMemory := constants.NodeMemory
	workerNodeStorage := constants.NodeStorage
	workerNodeCPUs := constants.NodeCPUs
	bootVolumeName := constants.BootVolumeName
	bootVolumeContainerImagePath := constants.BootVolumeContainerImagePath
	ociNamespace := "ocne"
	defaultConfig := types.Config{
		Providers: types.Providers{
			Libvirt: types.LibvirtProvider{
				SessionURI:  &sessionURI,
				StoragePool: &storagePool,
				Network:     &network,
				ControlPlaneNode: types.Node{
					Memory:  &controlNodeMemory,
					Storage: &controlNodeStorage,
					CPUs:    &controlNodeCPUs,
				},
				WorkerNode: types.Node{
					Memory:  &workerNodeMemory,
					Storage: &workerNodeStorage,
					CPUs:    &workerNodeCPUs,
				},
				BootVolumeName:               &bootVolumeName,
				BootVolumeContainerImagePath: &bootVolumeContainerImagePath,
			},
			Oci: types.OciProvider{
				Namespace:   &ociNamespace,
				ImageBucket: GenerateStringPointer(constants.OciBucket),
				ControlPlaneShape: types.OciInstanceShape{
					Shape: GenerateStringPointer(constants.OciVmStandardA1Flex),
					Ocpus: GenerateIntegerPointer(constants.OciControlPlaneOcpus),
				},
				WorkerShape: types.OciInstanceShape{
					Shape: GenerateStringPointer(constants.OciVmStandardE4Flex),
					Ocpus: GenerateIntegerPointer(constants.OciWorkerOcpus),
				},
			},
			Olvm: types.OlvmProvider{
				Namespace: GenerateStringPointer(constants.OLVMCAPIResourcesNamespace),
				ControlPlaneMachine: types.OlvmMachine{
					Memory: GenerateStringPointer(constants.OLVMCAPIControlPlaneMemory),
				},
				WorkerMachine: types.OlvmMachine{
					Memory: GenerateStringPointer(constants.OLVMCAPIWorkerMemory),
				},
				LocalAPIEndpoint: types.OlvmLocalAPIEndpoint{
					BindPort: GenerateIntegerPointer(6444),
				},
			},
		},
		PodSubnet:                GenerateStringPointer(constants.PodSubnet),
		ServiceSubnet:            GenerateStringPointer(constants.ServiceSubnet),
		KubeAPIServerBindPort:    GenerateUInt16Pointer(constants.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: GenerateUInt16Pointer(constants.KubeAPIServerBindPortAlt),
		AutoStartUI:              GenerateStringPointer("true"),
		CertificateInformation: types.CertificateInformation{
			Country: GenerateStringPointer(ignition.Country),
			Org:     GenerateStringPointer(ignition.Org),
			OrgUnit: GenerateStringPointer(ignition.OrgUnit),
			State:   GenerateStringPointer(ignition.State),
		},
		OsRegistry:               GenerateStringPointer(cmdconstants.OsRegistry),
		OsTag:                    GenerateStringPointer(constants.KubeVersion),
		KubeProxyMode:            GenerateStringPointer(ignition.KubeProxyMode),
		BootVolumeContainerImage: GenerateStringPointer(constants.BootVolumeContainerImage),
		Registry:                 GenerateStringPointer(constants.ContainerRegistry),
		EphemeralConfig: types.EphemeralClusterConfig{
			Name:     GenerateStringPointer(constants.EphemeralClusterName),
			Preserve: GenerateBooleanPointer(constants.EphemeralClusterPreserve),
			Node: types.Node{
				Memory:  GenerateStringPointer(constants.NodeMemory),
				CPUs:    GenerateIntegerPointer(constants.NodeCPUs),
				Storage: GenerateStringPointer(constants.EphemeralNodeStorage),
			},
		},
		KubeVersion: GenerateStringPointer(constants.KubeVersion),
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshKeyPath := filepath.Join(homedir, ".ssh", "id_rsa.pub")
	_, err = os.Stat(sshKeyPath)
	if os.IsNotExist(err) {
		sshKeyPath = ""
	} else if err != nil {
		return nil, err
	}
	*defaultConfig.SshPublicKeyPath = sshKeyPath

	// Load in the defaults.  Prefer the path set by OCNE_DEFAULTS_FILE.
	// If that is not set, use the default path.
	defaultPath := filepath.Join(homedir, constants.UserConfigDefaults)
	defaultPathOvr := os.Getenv(constants.UserConfigDefaultsEnvironmentVariable)
	if defaultPathOvr != "" {
		defaultPath = defaultPathOvr
	}

	configFileDefaults, err := ParseConfigFile(defaultPath)
	if os.IsNotExist(err) {
		return &defaultConfig, nil
	} else if err != nil {
		return nil, err
	}
	ret := types.MergeConfig(&defaultConfig, configFileDefaults)
	return &ret, err
}

// GenerateStringPointer is a helper function used to generate a string pointer
// This is useful when working with constants
func GenerateStringPointer(s string) *string {
	return &s
}

// GenerateIntegerPointer is a helper function used to generate an integer pointer
// This is useful when working with constants
func GenerateIntegerPointer(i int) *int {
	return &i
}

// GenerateUInt16Pointer is a helper function used to generate an Uint16 pointer
// This is useful when working with constants
func GenerateUInt16Pointer(u uint16) *uint16 {
	return &u
}

// GenerateBooleanPointer is a helper function used to generate an boolean pointer
// This is useful when working with constants
func GenerateBooleanPointer(b bool) *bool {
	return &b
}

// ParseClusterConfig taks a yaml-encoded string and parses it
// into a ClusterConfig structure.
func ParseClusterConfig(in string) (*types.ClusterConfig, error) {
	ret := &types.ClusterConfig{}
	err := yaml.Unmarshal([]byte(in), ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// ParseClusterConfigFile takes the path to a file, reads the contents,
// and parses it into a ClusterConfig structure.
func ParseClusterConfigFile(configPath string) (*types.ClusterConfig, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	ret, err := ParseClusterConfig(string(configBytes))
	if err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %s", configPath, err.Error())
	}

	// If the directory is not set, then set it to the
	// directory containing the config file itself.
	if *ret.WorkingDirectory == "" {
		wd, err := filepath.Abs(configPath)
		if err != nil {
			return nil, err
		}

		*ret.WorkingDirectory = filepath.Dir(wd)
	}
	return ret, nil
}
