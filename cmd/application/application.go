// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package application

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/application/install"
	"github.com/oracle-cne/ocne/cmd/application/list"
	"github.com/oracle-cne/ocne/cmd/application/show"
	"github.com/oracle-cne/ocne/cmd/application/template"
	"github.com/oracle-cne/ocne/cmd/application/uninstall"
	"github.com/oracle-cne/ocne/cmd/application/update"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
)

const (
	CommandName = "application"
	helpShort   = "Manage ocne applications"
	helpLong    = `Manage ocne applications`
	helpExample = `
ocne application install
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       CommandName,
		Short:     helpShort,
		Long:      helpLong,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{install.CommandName, list.CommandName, show.CommandName, template.CommandName, uninstall.CommandName, update.CommandName},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	cmd.AddCommand(install.NewCmd())
	cmd.AddCommand(list.NewCmd())
	cmd.AddCommand(show.NewCmd())
	cmd.AddCommand(template.NewCmd())
	cmd.AddCommand(uninstall.NewCmd())
	cmd.AddCommand(update.NewCmd())

	return cmd
}

// RunCmd runs the "ocne application" command
func RunCmd(cmd *cobra.Command) error {
	log.Info("ocne application command not yet implemented")
	return nil
}
