// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package node

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/cmd/node/update"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
)

const (
	CommandName = "node"
	helpShort   = "Manage ocne nodes"
	helpLong    = `Manage ocne nodes`
	helpExample = `
ocne node <subcommand>
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       CommandName,
		Short:     helpShort,
		Long:      helpLong,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{update.CommandName},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	cmd.AddCommand(update.NewCmd())
	return cmd
}

// RunCmdNode - Run the "ocne node" command
func RunCmd(cmd *cobra.Command) error {
	log.Info("ocne node command not yet implemented")
	return nil
}
