// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/node/update"
)

const (
	CommandName = "update"
	helpShort   = "Update a node on a cluster"
	helpLong    = `Updates the target node, if an update is available. The update is performed
by cordoning the node, draining the node, applying the update, rebooting the
node, and finally uncordoning the node once it has booted up to the new
version`
	helpExample = `
ocne node update --node my-node -t 10m
ocne node update --node my-node --delete-emptydir
`
)

var options update.UpdateOptions

const (
	flagNodeName      = "node"
	flagNodeNameShort = "N"
	flagNodeNameHelp  = "The name of the node to update, as seen from within Kubernetes. That is, the name should be one of the nodes listed in kubectl get nodes"

	flagEmptyDir      = "delete-emptydir-data"
	flagEmptyDirShort = "d"
	flagEmptyDirHelp  = "Delete pods that use emptyDir during node drain"

	flagEviction      = "disable-eviction"
	flagEvictionShort = "c"
	flagEvictionHelp  = "Force pods to be deleted during drain, bypassing PodDisruptionBudget"

	flagTimeout      = "timeout"
	flagTimeoutShort = "t"
	flagTimeoutHelp  = "Node drain timeout, such as 5m"
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
	cmd.Flags().StringVarP(&options.NodeName, flagNodeName, flagNodeNameShort, "", flagNodeNameHelp)
	cmd.MarkFlagRequired(flagNodeName)
	cmd.Flags().StringVarP(&options.Timeout, flagTimeout, flagTimeoutShort, "30m", flagTimeoutHelp)
	cmd.Flags().BoolVarP(&options.DeleteEmptyDir, flagEmptyDir, flagEmptyDirShort, false, flagEmptyDirHelp)
	cmd.Flags().BoolVarP(&options.DisableEviction, flagEviction, flagEvictionShort, false, flagEvictionHelp)
	return cmd
}

// RunCmd runs the "ocne node update" command
func RunCmd(cmd *cobra.Command) error {
	err := update.Update(options)
	return err
}