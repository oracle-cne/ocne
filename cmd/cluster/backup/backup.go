// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/backup"
)

const (
	CommandName = "backup"
	helpShort   = "Backup the etcd database"
	helpLong    = `Backup the contents of the etcd database that stores data for a target cluster`
	helpExample = `
ocne cluster backup --out example-path
`
)

var kubeConfig string
var out string

const (
	flagOut      = "out"
	flagOutShort = "o"
	flagOutHelp  = "The location where the backup materials are written"
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
	cmd.Flags().StringVarP(&out, flagOut, flagOutShort, "", flagOutHelp)
	cmd.MarkFlagRequired(flagOut)

	return cmd
}

// RunCmd runs the "ocne cluster backup" command
func RunCmd(cmd *cobra.Command) error {
	err := backup.Backup(kubeConfig, out)
	return err
}
