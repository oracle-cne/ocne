// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/ls"
)

const (
	CommandName = "list"
	Alias       = "ls"
	helpShort   = "List installed applications"
	helpLong    = `
List applications that are installed in a Kubernetes cluster from the application catalog`
	helpExample = `
ocne application list
`
)

var kubeConfig string
var namespace string
var all bool

const (
	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The Kubernetes namespace with applications to list. If this value is not provided, the namespace from the current context of the kubeconfig is used"

	flagAll      = "all"
	flagAllShort = "A"
	flagAllHelp  = "List applications in all kubernetes namespaces"
)

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

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&namespace, flagNamespace, flagNamespaceShort, "", flagNamespaceHelp)
	cmd.Flags().BoolVarP(&all, flagAll, flagAllShort, false, flagAllHelp)

	return cmd
}

// RunCmd runs the "ocne application ls" command
func RunCmd(cmd *cobra.Command) error {

	// Run the function to gather the release information as Go Structs
	releases, err := ls.List(application.LsOptions{
		KubeConfigPath: kubeConfig,
		Namespace:      namespace,
		All:            all,
	})

	if err != nil {
		return err
	}
	fmt.Printf("Releases\n")

	table := uitable.New()

	table.AddRow("NAME", "NAMESPACE", "CHART", "STATUS", "REVISION", "APPVERSION")
	for _, release := range releases {
		table.AddRow(release.Name, release.Namespace, release.Chart.Name(), release.Info.Status, release.Version, release.Chart.AppVersion())
	}
	fmt.Println(table)

	return nil
}
