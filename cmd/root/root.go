// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/oracle-cne/ocne/cmd/application"
	"github.com/oracle-cne/ocne/cmd/catalog"
	"github.com/oracle-cne/ocne/cmd/cluster"
	"github.com/oracle-cne/ocne/cmd/image"
	"github.com/oracle-cne/ocne/cmd/info"
	"github.com/oracle-cne/ocne/cmd/node"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CommandName = "ocne"
	helpShort   = "The ocne tool manages an ocne environment"
	helpLong    = `The ocne tool manages an ocne environment`

	flagLogLevel      = "log-level"
	flagLogLevelShort = "l"
	flagLogLevelHelp  = "Sets the log level.  Valid values are \"error\", \"info\", \"debug\", and \"trace\"."
)

var logLevel string

func stringToLogLevel(level string) log.Level {
	switch level {
	case "error":
		return log.ErrorLevel
	case "info":
		return log.InfoLevel
	case "debug":
		return log.DebugLevel
	case "trace":
		return log.TraceLevel
	default:
		log.Fatalf("%s is not a valid log level", level)
	}
	return log.InfoLevel
}

// NewRootCmd - create the root cobra command
func NewRootCmd() *cobra.Command {
	cmd := NewCommand(CommandName, helpShort, helpLong)

	// Add commands
	cmd.AddCommand(application.NewCmd())
	cmd.AddCommand(catalog.NewCmd())
	cmd.AddCommand(cluster.NewCmd())
	cmd.AddCommand(node.NewCmd())
	cmd.AddCommand(image.NewCmd())
	cmd.AddCommand(info.NewCmd())

	cmd.PersistentFlags().StringVarP(&logLevel, flagLogLevel, flagLogLevelShort, "info", flagLogLevelHelp)

	return cmd
}

// NewCommand - utility method to create cobra commands
func NewCommand(use string, short string, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetLevel(stringToLogLevel(logLevel))
		},
	}

	// Disable usage output on errors
	cmd.SilenceUsage = true
	return cmd
}
