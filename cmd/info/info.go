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
	helpShort   = "Display usage information"
	helpLong    = `Display usage information that may be difficult to find.`
	helpExample = `
ocne info
`
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
	cmdutil.SilenceUsage(cmd)
	cmd.Example = helpExample

	return cmd
}

// RunCmd runs the "ocne info" command
func RunCmd(cmd *cobra.Command) error {
	fmt.Printf("Environment Variables\n")

	envVars := map[string]string{
		"OCNE_DEFAULTS": "Sets the location of the default configuration file.",
		"KUBECONFIG":    "Sets the location of the kubeconfig file. This behaves the same way as the --kubeconfig option for most ocne commands.",
		"EDITOR":        "Sets the default document editor.",
	}

	table := uitable.New()
	table.Wrap = true
	table.MaxColWidth = 50

	table.AddRow("Name", "Description", "Current Value")
	for envVar, description := range envVars {
		table.AddRow(envVar, description, os.Getenv(envVar))
	}
	fmt.Println(table)

	return nil

}
