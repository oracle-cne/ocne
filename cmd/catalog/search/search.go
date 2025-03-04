// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package search

import (
	"fmt"
	"sort"

	"github.com/gosuri/uitable"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/spf13/cobra"
)

const (
	CommandName = "search"
	helpShort   = "Discover applications in a catalog"
	helpLong    = `Discover applications in an application catalog that follow a specific pattern in a Kubernetes cluster`
	helpExample = `
# Search a catalog for a matching pattern
ocne catalog search --name mycatalog --pattern *

# Search the catalog that is built into the CLI
ocne catalog search --name embedded
`
)

var kubeConfig string
var pattern string
var catalogName string

const (
	flagCatalogName      = "name"
	flagCatalogNameShort = "N"
	flagCatalogNameHelp  = "The name of the catalog to search. The catalog named \"embedded\" can be searched without creating a cluster"

	flagPattern      = "pattern"
	flagPatternShort = "p"
	flagPatternHelp  = "The terms to search for. The patterns must be a valid RE2 regular expression"
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
	cmd.Flags().StringVarP(&pattern, flagPattern, flagPatternShort, "", flagPatternHelp)

	return cmd
}

// RunCmd runs the "ocne catalog search" command
func RunCmd(cmd *cobra.Command) error {
	cat, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: kubeConfig,
		CatalogName:    catalogName,
		Pattern:        pattern,
	})
	if err != nil {
		return err
	}

	// Get the names of all the charts.
	// ChartEntries map[string][]ChartMeta `json:"entries"`
	var charts []catalog.ChartMeta
	for _, chartMetas := range cat.ChartEntries {
		for i := range chartMetas {
			charts = append(charts, chartMetas[i])
		}
	}

	// The entries that come back from search.Search() are sorted.
	// It's hard to sort the random nonsense that can come from
	// catalogs.  Use a stable sort to ensure that the versions
	// remain sorted.
	sort.SliceStable(charts, func(i, j int) bool {
		return charts[i].Name < charts[j].Name
	})

	table := uitable.New()
	table.AddRow("APPLICATION", "VERSION")
	for _, chart := range charts {
		table.AddRow(chart.Name, chart.Version)
	}
	fmt.Println(table)

	return nil
}
