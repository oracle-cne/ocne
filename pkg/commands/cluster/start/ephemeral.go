// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"github.com/oracle-cne/ocne/pkg/cluster/driver/libvirt"
	delete2 "github.com/oracle-cne/ocne/pkg/commands/cluster/delete"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	log "github.com/sirupsen/logrus"
)

// EnsureCluster returns the kubeconfig to a functioning cluster.  If there is
// a kubeconfig provided, it will return that.  If not, it will create a
// cluster using the libvirt provider that mostly uses the default configuration.
func EnsureCluster(kubeconfigPath string, config *types.Config, clusterConfig *types.ClusterConfig) (string, bool, error) {
	// Try to get a valid kubeconfig.  If one exists, then a cluster
	// is available.

	log.Debugf("Ensuring that cluster is available")
	path, isDefault, err := client.GetKubeConfigLocation(kubeconfigPath)
	log.Debugf("Processed kubeconfig at \"%s\": %+v", path, err)
	if !isDefault {
		return path, false, err
	}

	kubeConfigPath, err := StartEphemeralCluster(config, clusterConfig)
	return kubeConfigPath, true, err
}

// StartEphemeralCluster
func StartEphemeralCluster(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
	log.Debugf("Starting ephemeral cluster %s", config.EphemeralConfig.Name)

	// Make local copies of the configuration that is passed
	// in so that it can be edited.
	origConfig := config
	c := types.CopyConfig(config)
	cc := types.CopyClusterConfig(clusterConfig)
	config = &c
	clusterConfig = &cc

	// Force some settings to what is required for
	// the ephemeral cluster.
	clusterConfig.Provider = libvirt.DriverName
	clusterConfig.WorkerNodes = 0
	clusterConfig.Headless = true
	clusterConfig.Catalog = false
	clusterConfig.CommunityCatalog = false
	clusterConfig.CNI = constants.CNIFlannel
	clusterConfig.Name = config.EphemeralConfig.Name
	clusterConfig.Providers.Libvirt.ControlPlaneNode = config.EphemeralConfig.Node
	config.Quiet = true
	config.QuietPtr = &config.Quiet
	config.BootVolumeContainerImage = clusterConfig.BootVolumeContainerImage
	config.KubeVersion = constants.KubeVersion
	clusterConfig.KubeVersion = constants.KubeVersion
	config.OsTag = clusterConfig.OsTag

	// If there was no valid kubeconfig, then a cluster is needed.  Make
	// an ephemeral cluster that has basic functionality.  That is, there
	// is a basic CNI in the form of flannel and nothing else.
	kubeConfigPath, err := Start(config, clusterConfig)
	if err != nil {
		return "", err
	}

	// Set the configuration kubeconfig to this kubeconfig.  This
	// is done so that callers can automatically pick up the new
	// path and to scratch down that the ephemeral cluster is being
	// used.
	origConfig.KubeConfig = kubeConfigPath

	return kubeConfigPath, nil
}

// StopEphemeralCluster
func StopEphemeralCluster(config *types.Config, clusterConfig *types.ClusterConfig) error {
	// If the cluster is supposed to be preserved, then nothing else matters.
	if config.EphemeralConfig.Preserve {
		return nil
	}

	c := types.CopyConfig(config)
	cc := types.CopyClusterConfig(clusterConfig)
	config = &c
	clusterConfig = &cc

	clusterConfig.Provider = libvirt.DriverName
	clusterConfig.Name = config.EphemeralConfig.Name
	config.Quiet = true
	return delete2.Delete(config, clusterConfig)
}
