// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package list

import (
	"github.com/spf13/cobra"

	"github.com/oracle-cne/ocne/pkg/cmdutil"
	cmdlist "github.com/oracle-cne/ocne/pkg/commands/cluster/ls"
)

const (
	CommandName = "list"
	Alias       = "ls"
	helpShort   = "List clusters"
	helpLong    = "Lists all known clusters as well as their complete configuration"
	helpExample = `
ocne cluster list
`
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

	return cmd
}

// RunCmd runs the "ocne cluster list" command
func RunCmd(cmd *cobra.Command) error {
	err := cmdlist.List()
	return err
}
