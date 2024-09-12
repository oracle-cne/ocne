// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stage

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/stage"
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
	err := cmd.MarkFlagRequired(flagVersion)
	if err != nil {
		return nil
	}
	return cmd
}

// RunCmd runs the "ocne node update" command
func RunCmd(cmd *cobra.Command) error {
	err := stage.Stage(options)
	return err
}
