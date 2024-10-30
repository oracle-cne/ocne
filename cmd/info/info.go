// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	CommandName = "info"
	helpShort   = "Display information about OCNE "
	helpLong    = `Display information about OCNE that may be difficult to find.`
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
	fmt.Println("The OCNE_DEFAULTS environment variable sets the location of the default configuration file.")
	fmt.Println("The KUBECONFIG environment variable sets the location of the kubeconfig file. This behaves the same way as the --kubeconfig option for most ocne commands.")
	fmt.Println("The EDITOR environment variable sets the default document editor.")
	return nil

}
