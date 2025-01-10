// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stage

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/stage"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

const (
	CommandName = "stage"
	helpShort   = "Stage a cluster update to a specified k8s version"
	helpLong    = `Sets the kubernetes version of all nodes and updates the Kubernetes version of the cluster. 
Staging an update prompts each node to download the requested update. 
Once the update is available, each node update must be manually applied.`
	helpExample = `
ocne cluster stage --kubeconfig config.yaml --version 1.28
`
)

var options stage.StageOptions
var clusterName string
var clusterConfigPath string

const (
	flagVersion      = "version"
	flagVersionShort = "v"
	flagVersionHelp  = "The version of Kubernetes to update to"

	flagTransport      = "transport"
	flagTransportShort = "t"
	flagTransportHelp  = "The type of transport to use during an upgrade"

	flagOsRegistry      = "os-registry"
	flagOsRegistryShort = "r"
	flagOsRegistryHelp  = "The name of the os registry to use during an upgrade"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: helpShort,
		Long:  helpLong,
		Args:  cobra.MatchAll(cobra.ExactArgs(0), cobra.OnlyValidArgs),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.Flags().StringVarP(&options.KubeConfigPath, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&options.KubeVersion, flagVersion, flagVersionShort, "", flagVersionHelp)
	cmd.Flags().StringVarP(&options.Transport, flagTransport, flagTransportShort, "", flagTransportHelp)
	cmd.Flags().StringVarP(&options.OsRegistry, flagOsRegistry, flagOsRegistryShort, "", flagOsRegistryHelp)
	cmd.Flags().StringVarP(&clusterName, constants.FlagClusterName, constants.FlagClusterNameShort, "", constants.FlagClusterNameHelp)
	cmd.Flags().StringVarP(&clusterConfigPath, constants.FlagConfig, constants.FlagConfigShort, "", constants.FlagConfigHelp)
	cmd.MarkFlagsMutuallyExclusive(constants.FlagConfig, constants.FlagClusterName)
	err := cmd.MarkFlagRequired(flagVersion)
	if err != nil {
		return nil
	}
	return cmd
}

// RunCmd runs the "ocne cluster stage" command
func RunCmd(cmd *cobra.Command) error {
	// Only try to look in the cache if a name was explicitly given.
	// This protects against accidentally staging the "ocne" cluster
	// when supplying only a kubeconfig.
	if clusterName != "" {
		clusterCache, err := cache.GetCache()
		if err != nil {
			return err
		}

		cached := clusterCache.Get(clusterName)
		if cached == nil {
			return fmt.Errorf("Cluster %s is not in the cache", clusterName)
		}

		c := &types.Config{}
		cc := &types.ClusterConfig{}
		c, cc, err = cmdutil.GetFullConfig(c, &cached.ClusterConfig, clusterConfigPath)
		if err != nil {
			return err
		}

		options.ClusterConfig = cc
		options.Config = c
	}

	if clusterConfigPath != "" {
		c := &types.Config{}
		cc := &types.ClusterConfig{}

		c, cc, err := cmdutil.GetFullConfig(c, cc, clusterConfigPath)
		if err != nil {
			return err
		}

		options.ClusterConfig = cc
		options.Config = c
	}

	err := stage.Stage(options)
	return err
}
