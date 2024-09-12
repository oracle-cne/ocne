// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/catalog/add"
	"github.com/oracle-cne/ocne/cmd/catalog/copy"
	"github.com/oracle-cne/ocne/cmd/catalog/get"
	"github.com/oracle-cne/ocne/cmd/catalog/list"
	"github.com/oracle-cne/ocne/cmd/catalog/mirror"
	"github.com/oracle-cne/ocne/cmd/catalog/remove"
	"github.com/oracle-cne/ocne/cmd/catalog/search"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
)

const (
	CommandName = "catalog"
	helpShort   = "Manage ocne catalogs"
	helpLong    = `Manage ocne catalogs`
	helpExample = `
ocne catalog <subcommand>
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       CommandName,
		Short:     helpShort,
		Long:      helpLong,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{add.CommandName, get.CommandName, list.CommandName, mirror.CommandName, remove.CommandName, search.CommandName, copy.CommandName},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	cmd.AddCommand(add.NewCmd())
	cmd.AddCommand(get.NewCmd())
	cmd.AddCommand(list.NewCmd())
	cmd.AddCommand(mirror.NewCmd())
	cmd.AddCommand(remove.NewCmd())
	cmd.AddCommand(search.NewCmd())
	cmd.AddCommand(copy.NewCmd())

	return cmd
}

// RunCmd runs the "ocne catalog" command
func RunCmd(cmd *cobra.Command) error {
	log.Info("ocne catalog command not yet implemented")
	return nil
}
