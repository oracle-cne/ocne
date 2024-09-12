// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package remove

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/remote"
)

const (
	CommandName = "remove"
	helpShort   = "Removes a catalog"
	helpLong    = `Removes a catalog from a Kubernetes cluster`
	helpExample = `
ocne catalog remove --name mycatalog 
`
)

var kubeConfig string
var catalogName string
var namespace string

const (
	flagCatalogName      = "name"
	flagCatalogNameShort = "N"
	flagCatalogNameHelp  = "The name of the catalog to remove"

	flagCatalogNamespace      = "namespace"
	flagCatalogNamespaceShort = "n"
	flagCatalogNamespaceHelp  = "The namespace of the catalog to remove"
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

	cmd.Flags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, "", flagCatalogNameHelp)
	cmd.Flags().StringVarP(&namespace, flagCatalogNamespace, flagCatalogNamespaceShort, "ocne-system", flagCatalogNamespaceHelp)
	cmd.MarkFlagRequired(flagCatalogName)

	return cmd
}

// RunCmd runs the "ocne catalog remove" command
func RunCmd(cmd *cobra.Command) error {
	return remote.Remove(kubeConfig, catalogName, namespace)
}
