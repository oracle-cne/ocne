// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package get

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"sigs.k8s.io/yaml"
)

const (
	CommandName = "get"
	helpShort   = "Gets a catalog from the cluster"
	helpLong    = `Emit a YAML document that contains the complete description of the given application catalog`
	helpExample = `
ocne catalog get --name mycatalog 
`
)

var kubeConfig string
var catalogName string

const (
	flagCatalogName      = "name"
	flagCatalogNameShort = "N"
	flagCatalogNameHelp  = "The name of the catalog to retrieve"
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
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, pkgconst.DefaultCatalogName, flagCatalogNameHelp)

	return cmd
}

// RunCmd runs the "ocne catalog get" command
func RunCmd(cmd *cobra.Command) error {
	cat, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: kubeConfig,
		CatalogName:    catalogName,
	})
	if err != nil {
		return err
	}

	yam, err := yaml.JSONToYAML(cat.RawJSON)

	fmt.Printf("\n")
	fmt.Printf("Catalog\n")
	fmt.Printf("--------\n")
	fmt.Println(string(yam))
	fmt.Println()

	return nil
}
