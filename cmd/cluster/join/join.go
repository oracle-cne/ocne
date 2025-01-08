// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package join

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	cmdjoin "github.com/oracle-cne/ocne/pkg/commands/cluster/join"
	config2 "github.com/oracle-cne/ocne/pkg/config"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

const (
	CommandName = "join"
	helpShort   = "Join a node to a cluster, or generate the materials required to do so"
	helpLong    = `Join a node to a cluster, or generate the materials required to do so.
This subcommand targets three cases: local virtualization, pre-provisioned compute, and self-provisioned compute. For clusters created on local
virtualization, new virtual machines are created and joined to the target cluster. For pre-provisioned cases, it migrates a node from one
cluster to another cluster. For self-provisioned cases, it generates the materials needed to join a node to a cluster on first boot`
	helpExample = `
ocne cluster join --node mynode --destination example-path/kubernetes-client-configuration --worker-nodes 2
`
)

const (
	flagDestination      = "destination"
	flagDestinationShort = "d"
	flagDestinationHelp  = "The path to a Kubernetes client configuration that describes the cluster that the node will join"

	flagControlPlaneNodes      = "control-plane-nodes"
	flagControlPlaneNodesShort = "n"
	flagControlPlaneNodesHelp  = "The number of control plane nodes to provision for clusters deployed using local virtualization"

	flagWorkerNodes      = "worker-nodes"
	flagWorkerNodesShort = "w"
	flagWorkerNodesHelp  = "The number of worker nodes to provision for clusters deployed using local virtualization"

	flagNode      = "node"
	flagNodeShort = "N"
	flagNodeHelp  = "The name of the node to move from the source cluster to the destination cluster, as seen from within Kubernetes. " +
		"That is, the name should be one the nodes listed in kubectl get nodes"

	flagRoleControlPlane      = "role-control-plane"
	flagRoleControlPlaneShort = "r"
	flagRoleControlPlaneHelp  = "If node and destination are specified, the node will join the destination cluster as a control plane node"
)

var clusterConfigPath string

var options cmdjoin.JoinOptions = cmdjoin.JoinOptions{
	Config:        &types.Config{},
	ClusterConfig: &types.ClusterConfig{},
}

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

	cmd.Flags().StringVarP(&clusterConfigPath, constants.FlagConfig, constants.FlagConfigShort, "", constants.FlagConfigHelp)
	cmd.Flags().StringVarP(&options.KubeConfigPath, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&options.DestKubeConfigPath, flagDestination, flagDestinationShort, "", flagDestinationHelp)
	cmd.Flags().StringVarP(&options.Provider, constants.FlagProviderName, constants.FlagProviderNameShort, "libvirt", constants.FlagProviderNameHelp)
	cmd.Flags().StringVarP(&options.Node, flagNode, flagNodeShort, "", flagNodeHelp)
	cmd.Flags().IntVarP(&options.ControlPlaneNodes, flagControlPlaneNodes, flagControlPlaneNodesShort, 0, flagControlPlaneNodesHelp)
	cmd.Flags().IntVarP(&options.WorkerNodes, flagWorkerNodes, flagWorkerNodesShort, 0, flagWorkerNodesHelp)
	cmd.Flags().BoolVarP(&options.RoleControlPlane, flagRoleControlPlane, flagRoleControlPlaneShort, false, flagRoleControlPlaneHelp)

	return cmd
}

// RunCmd runs the "ocne cluster join" command
func RunCmd(cmd *cobra.Command) error {
	if err := validateOptions(&options); err != nil {
		return err
	}
	clusterCache, err := cache.GetCache()
	if err != nil {
		return err
	}

	var clusterName string

	if clusterConfigPath != "" {
		options.ClusterConfig, err = config2.ParseClusterConfigFile(clusterConfigPath)
		if err != nil {
			return err
		}
		// Make sure the tag matches Kubeversion, unless it is overridden
		if *options.ClusterConfig.OsTag == "" {
			options.ClusterConfig.OsTag = options.ClusterConfig.KubeVersion
		}

		clusterName = *options.ClusterConfig.Name
	}

	cached := clusterCache.Get(clusterName)

	// If the cluster does not exist, fall back to the CLI options.
	// This is a bail-out to make sure all the needful can be done in
	// case of some poorly timed error.
	if cached == nil {
		options.ClusterConfig, err = cmdutil.GetFullConfig(options.Config, options.ClusterConfig, clusterConfigPath)
		if err != nil {
			return err
		}
	} else {
		cc := &cached.ClusterConfig
		if options.ClusterConfig != nil {
			merged := types.MergeClusterConfig(cc, options.ClusterConfig)
			cc = &merged
		}
		options.ClusterConfig = cc
	}

	if options.KubeConfigPath == "" && cached != nil {
		options.KubeConfigPath = cached.KubeconfigPath
	}

	// If both the number of nodes to join are zero, and this
	// is not a node migration, then set either control plane
	// or worker node count to 1 based on node role.
	if options.DestKubeConfigPath == "" && options.WorkerNodes == 0 && options.ControlPlaneNodes == 0 {
		if options.RoleControlPlane {
			options.ControlPlaneNodes = 1
		} else {
			options.WorkerNodes = 1
		}
	}

	return cmdjoin.Join(&options)
}

func validateOptions(options *cmdjoin.JoinOptions) error {
	// This is a workaround for using the Cobra CLI to populate a structure that has fields that are pointers
	options.ClusterConfig.Provider = &options.Provider

	if options.DestKubeConfigPath != "" || options.Node != "" {
		// this is the case where we are joining an existing node to another cluster, both of these options must be specified
		if options.DestKubeConfigPath == "" {
			return fmt.Errorf("A destination client configuration path must be specified when joining an existing node to a cluster")
		}
		if options.Node == "" {
			return fmt.Errorf("A node must be specified when joining an existing node to a cluster")
		}
	}

	return nil
}
