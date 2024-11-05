// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"fmt"
	"github.com/gosuri/uitable"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/spf13/cobra"
	"os"
)

const (
	CommandName = "info"
	helpShort   = "Displays version and setting information"
	helpLong    = `Displays settings for options that are not available from individual commands along with version information.`
	helpExample = `
ocne info
`
)

var cliVersion string
var buildDate string
var gitCommit string

type infoStruct struct {
	name  string
	value string
}

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
	cmdutil.SilenceUsage(cmd)
	cmd.Example = helpExample

	return cmd
}

// RunCmd runs the "ocne info" command
func RunCmd(cmd *cobra.Command) error {

	fmt.Printf("CLI Info\n")

	infoArgs := []infoStruct{
		{name: "Version", value: cliVersion},
		{name: "BuildDate", value: buildDate},
		{name: "GitCommit", value: gitCommit},
	}

	infoTable := uitable.New()
	infoTable.Wrap = true
	infoTable.MaxColWidth = 50

	infoTable.AddRow("Name", "Value")
	for _, pair := range infoArgs {
		infoTable.AddRow(pair.name, pair.value)
	}
	fmt.Println(infoTable)

	fmt.Println()

	fmt.Printf("Environment Variables\n")

	envVars := []infoStruct{
		{name: "OCNE_DEFAULTS", value: "Sets the location of the default configuration file."},
		{name: "KUBECONFIG", value: "Sets the location of the kubeconfig file. This behaves the same way as the --kubeconfig option for most ocne commands."},
		{name: "EDITOR", value: "Sets the default document editor."},
	}

	table := uitable.New()
	table.Wrap = true
	table.MaxColWidth = 50

	table.AddRow("Name", "Description", "Current Value")
	for _, pair := range envVars {
		table.AddRow(pair.name, pair.value, os.Getenv(pair.name))
	}
	fmt.Println(table)

	return nil

}
