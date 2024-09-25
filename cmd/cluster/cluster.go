// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/oracle-cne/ocne/cmd/cluster/analyze"
	"github.com/oracle-cne/ocne/cmd/cluster/backup"
	"github.com/oracle-cne/ocne/cmd/cluster/console"
	"github.com/oracle-cne/ocne/cmd/cluster/delete"
	"github.com/oracle-cne/ocne/cmd/cluster/dump"
	"github.com/oracle-cne/ocne/cmd/cluster/info"
	"github.com/oracle-cne/ocne/cmd/cluster/join"
	"github.com/oracle-cne/ocne/cmd/cluster/list"
	"github.com/oracle-cne/ocne/cmd/cluster/show"
	"github.com/oracle-cne/ocne/cmd/cluster/stage"
	"github.com/oracle-cne/ocne/cmd/cluster/start"
	"github.com/oracle-cne/ocne/cmd/cluster/template"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CommandName = "cluster"
	helpShort   = "Manage ocne clusters"
	helpLong    = `Manage the lifecycle of ocne clusters and application deployment.`
	helpExample = `
ocne cluster <subcommand>
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       CommandName,
		Short:     helpShort,
		Long:      helpLong,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{analyze.CommandName, backup.CommandName, console.CommandName, dump.CommandName, info.CommandName, join.CommandName, delete.CommandName, template.CommandName, stage.CommandName},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	cmd.AddCommand(analyze.NewCmd())
	cmd.AddCommand(backup.NewCmd())
	cmd.AddCommand(console.NewCmd())
	cmd.AddCommand(dump.NewCmd())
	cmd.AddCommand(info.NewCmd())
	cmd.AddCommand(join.NewCmd())
	cmd.AddCommand(list.NewCmd())
	cmd.AddCommand(delete.NewCmd())
	cmd.AddCommand(show.NewCmd())
	cmd.AddCommand(start.NewCmd())
	cmd.AddCommand(template.NewCmd())
	cmd.AddCommand(stage.NewCmd())
	return cmd
}

// RunCmd runs the "ocne cluster" command
func RunCmd(cmd *cobra.Command) error {
	log.Info("ocne cluster command not yet implemented")
	return nil
}
