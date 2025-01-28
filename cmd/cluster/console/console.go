// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/console"
	constantspkg "github.com/oracle-cne/ocne/pkg/constants"
)

const (
	CommandName = "console"
	helpShort   = "Launch a console on a node"
	helpLong    = `Launch an administration console on nodes in a Kubernetes cluster. Use the --direct option
  to access the local filesystem of the target node.`
	helpExample = `
ocne cluster console --node mynode --toolbox
`
)

var kubeConfig string
var nodeName string
var toolbox bool
var chroot bool
var defaultRegistry string

const (
	flagNodeName      = "node"
	flagNodeNameShort = "N"
	flagNodeNameHelp  = "The Kubernetes cluster node where the console is to be launched"

	flagToolbox      = "toolbox"
	flagToolboxShort = "t"
	flagToolboxHelp  = "Create the console using a container image that contains a variety of tools that are useful for diagnosing a Linux system"

	flagChrootName  = "direct"
	flagChrootShort = "d"
	flagChrootHelp  = "Access the node's root directory directly."
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CommandName,
		Short: helpShort,
		Long:  helpLong,
		Args:  cobra.MatchAll(cobra.OnlyValidArgs),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.Flags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&nodeName, flagNodeName, flagNodeNameShort, "", flagNodeNameHelp)
	cmd.MarkFlagRequired(flagNodeName)
	cmd.Flags().BoolVarP(&toolbox, flagToolbox, flagToolboxShort, false, flagToolboxHelp)
	cmd.Flags().BoolVarP(&chroot, flagChrootName, flagChrootShort, false, flagChrootHelp)
	cmd.Flags().StringVarP(&defaultRegistry, constants.FlagSource, constants.FlagSourceShort, constantspkg.ContainerRegistry, constants.FlagSourceHelp)

	return cmd
}

// RunCmd runs the "ocne cluster console" command
func RunCmd(cmd *cobra.Command) error {
	// Get any command that was provided
	cmds := []string{}
	if cmd.ArgsLenAtDash() >= 0 {
		cmds = cmd.Flags().Args()[cmd.ArgsLenAtDash():]
	}

	options := console.Options{
		KubeConfigPath:  kubeConfig,
		NodeName:        nodeName,
		DefaultRegistry: defaultRegistry,
		Toolbox:         toolbox,
		Chroot:          chroot,
		Commands:        cmds,
	}

	err := console.Console(options)
	return err
}
