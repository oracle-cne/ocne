// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"os"

	"github.com/spf13/cobra"
)

// ArgsCheck - common function for checking args for commands that only have
// sub-commands.  The help for the sub-command will be displayed if no args are passed.
func ArgsCheck(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cmd.Help()
		os.Exit(0)
	}
	cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)
	return nil
}
