// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
)

const (
	CommandName = "list"
	Alias       = "ls"
	helpShort   = "Lists the catalogs"
	helpLong    = `Lists the application catalogs configured for a particular Kubernetes cluster`
	helpExample = `
ocne catalog list
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     CommandName,
		Aliases: []string{Alias},
		Short:   helpShort,
		Long:    helpLong,
		Args:    cobra.MatchAll(cobra.ExactArgs(0), cobra.OnlyValidArgs),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmd.Example = helpExample
	cmdutil.SilenceUsage(cmd)

	cmd.Flags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	return cmd
}

// RunCmd runs the "ocne catalog list" command
func RunCmd(cmd *cobra.Command) error {
	catalogs, err := ls.Ls(kubeConfig)
	if err != nil {
		return err
	}

	table := uitable.New()
	table.AddRow("CATALOG", "NAMESPACE", "PROTOCOL", "URI")

	for _, catalog := range catalogs {
		table.AddRow(catalog.CatalogName, catalog.ServiceNsn.Namespace, catalog.Protocol, catalog.Uri)
	}
	fmt.Println(table)

	return nil
}
