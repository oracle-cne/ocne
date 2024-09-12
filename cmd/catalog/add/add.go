// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	"github.com/oracle-cne/ocne/pkg/commands/catalog/add"
	"strings"

	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
)

const (
	CommandName = "add"
	helpShort   = "Adds a catalog"
	helpLong    = `Adds a catalog to a Kubernetes cluster`
	helpExample = `
ocne catalog add --name mycatalog --uri my-catalog-uri.org
`
)

var kubeConfig string
var uri string
var catalogName string
var namespace string
var protocol string

const (
	flagCatalogName      = "name"
	flagCatalogNameShort = "N"
	flagCatalogNameHelp  = "The name of the added catalog"

	flagURI      = "uri"
	flagURIShort = "u"
	flagURIHelp  = "The URI of the application catalog to add"

	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The namespace to put the catalog in"

	flagProtocol      = "protocol"
	flagProtocolShort = "p"
	flagProtocolHelp  = "The catalog protocol for the added catalog"
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
	cmd.Flags().StringVarP(&namespace, flagNamespace, flagNamespaceShort, pkgconst.OCNESystemNamespace, flagNamespaceHelp)
	cmd.Flags().StringVarP(&protocol, flagProtocol, flagProtocolShort, "helm", flagProtocolHelp)
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, "", flagCatalogNameHelp)
	cmd.MarkFlagRequired(flagCatalogName)
	cmd.Flags().StringVarP(&uri, flagURI, flagURIShort, "", flagURIHelp)
	cmd.MarkFlagRequired(flagURI)

	return cmd
}

// RunCmd runs the "ocne catalog add" command
func RunCmd(cmd *cobra.Command) error {
	// Generate a service name from the catalog name
	svcName := strings.ReplaceAll(catalogName, " ", "")
	svcName = strings.ToLower(svcName)
	if len(svcName) >= 64 {
		svcName = svcName[:63]
	}
	return add.Add(kubeConfig, svcName, namespace, uri, protocol, catalogName)
}
