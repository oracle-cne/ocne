// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package show

import (
	"github.com/spf13/cobra"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	cmdshow "github.com/oracle-cne/ocne/pkg/commands/cluster/show"
)

const (
	CommandName = "show"
	helpShort   = "Show cluster configuration"
	helpLong    = "Show configuration for a cluster"
	helpExample = `
ocne cluster show -C mycluster
ocne cluster show -C mycluster -a
ocne cluster show -C mycluster -f kubeconfig
`

	flagAll      = "all"
	flagAllShort = "a"
	flagAllHelp  = "Show full cluster configuration"

	flagField      = "field"
	flagFieldShort = "f"
	flagFieldHelp  = "The path to the field to show"
)

var clusterName string
var all bool
var field string

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

	cmd.Flags().StringVarP(&clusterName, constants.FlagClusterName, constants.FlagClusterNameShort, "ocne", constants.FlagClusterNameHelp)
	cmd.Flags().BoolVarP(&all, flagAll, flagAllShort, false, flagAllHelp)
	cmd.Flags().StringVarP(&field, flagField, flagFieldShort, "kubeconfig", flagFieldHelp)

	return cmd
}

// RunCmd runs the "ocne cluster show" command
func RunCmd(cmd *cobra.Command) error {
	err := cmdshow.Show(clusterName, all, field)
	return err
}
