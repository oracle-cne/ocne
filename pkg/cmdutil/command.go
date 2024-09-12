// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmdutil

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCommand - utility method to create cobra commands
func NewCommand(use string, short string, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}

	// Disable usage output on errors
	cmd.SilenceUsage = true
	return cmd
}

func SilenceUsage(cmd *cobra.Command) {
	// If RunE is not set, it's a fatal error.  This is not because calling
	// this is a problem in and of itself.  It is to protect callers of
	// mistakenly calling this without the RunE set, preventing the
	// wrapper function from being correctly installed.
	if cmd.RunE == nil {
		log.Fatalf("SilenceUsage() called before RunE is set for command %s", cmd.Use)
	}

	// Disable usage output on errors
	cmd.SilenceUsage = true

	// wrap the RunE function so that error
	// messages from cobra are suppressed if
	// the error comes from the function
	runCmd := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		err := runCmd(c, args)
		if err != nil {
			c.SilenceErrors = true
			log.Error(err)
		}
		return err
	}
}
