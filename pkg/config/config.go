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
	defaultConfig := types.Config{
		Providers: types.Providers{
			Libvirt: types.LibvirtProvider{
				SessionURI:  constants.SessionURI,
				StoragePool: constants.StoragePool,
				Network:     constants.Network,
				ControlPlaneNode: types.Node{
					Memory:  constants.NodeMemory,
					Storage: constants.NodeStorage,
					CPUs:    constants.NodeCPUs,
				},
				WorkerNode: types.Node{
					Memory:  constants.NodeMemory,
					Storage: constants.NodeStorage,
					CPUs:    constants.NodeCPUs,
				},
				BootVolumeName:               constants.BootVolumeName,
				BootVolumeContainerImagePath: constants.BootVolumeContainerImagePath,
			},
			Oci: types.OciProvider{
				Namespace:   "ocne",
				Profile:     constants.OciDefaultProfile,
				ImageBucket: constants.OciBucket,
				ControlPlaneShape: types.OciInstanceShape{
					Shape: constants.OciVmStandardA1Flex,
					Ocpus: constants.OciControlPlaneOcpus,
				},
				WorkerShape: types.OciInstanceShape{
					Shape: constants.OciVmStandardE4Flex,
					Ocpus: constants.OciWorkerOcpus,
				},
			},
			Olvm: types.OlvmProvider{
				Namespace: constants.OLVMCAPIResourcesNamespace,
				ControlPlaneMachine: types.OlvmMachine{
					Memory: constants.OLVMCAPIControlPlaneMemory,
				},
				WorkerMachine: types.OlvmMachine{
					Memory: constants.OLVMCAPIWorkerMemory,
				},
				LocalAPIEndpoint: types.OlvmLocalAPIEndpoint{
					BindPort: 6444,
				},
			},
		},
		PodSubnet:                constants.PodSubnet,
		ServiceSubnet:            constants.ServiceSubnet,
		KubeAPIServerBindPort:    constants.KubeAPIServerBindPort,
		KubeAPIServerBindPortAlt: constants.KubeAPIServerBindPortAlt,
		AutoStartUI:              "true",
		CertificateInformation: types.CertificateInformation{
			Country: ignition.Country,
			Org:     ignition.Org,
			OrgUnit: ignition.OrgUnit,
			State:   ignition.State,
		},
		OsRegistry:               cmdconstants.OsRegistry,
		OsTag:                    constants.KubeVersion,
		KubeProxyMode:            ignition.KubeProxyMode,
		BootVolumeContainerImage: constants.BootVolumeContainerImage,
		Registry:                 constants.ContainerRegistry,
		EphemeralConfig: types.EphemeralClusterConfig{
			Name:     constants.EphemeralClusterName,
			Preserve: constants.EphemeralClusterPreserve,
			Node: types.Node{
				Memory:  constants.NodeMemory,
				CPUs:    constants.NodeCPUs,
				Storage: constants.EphemeralNodeStorage,
			},
		},
		KubeVersion: constants.KubeVersion,
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
	defaultConfig.SshPublicKeyPath = sshKeyPath

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
	if ret.WorkingDirectory == "" {
		wd, err := filepath.Abs(configPath)
		if err != nil {
			return nil, err
		}

		ret.WorkingDirectory = filepath.Dir(wd)
	}
	return ret, nil
}
