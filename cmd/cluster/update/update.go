// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
)

const (
	CommandName = "update"
	helpShort   = "Updates the version of a cluster"
	helpLong    = `Updates the version of a Kubernetes cluster. Only the cluster version is updated. The ocne node update is used to update individual cluster nodes`
	helpExample = `
ocne cluster update --version 1.29.0
`
)

var kubeConfig string
var version string

const (
	flagVersion      = "version"
	flagVersionShort = "v"
	flagVersionHelp  = "The target Kubernetes version"
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
	cmd.Flags().StringVarP(&version, flagVersion, flagVersionShort, "", flagVersionHelp)
	cmd.MarkFlagRequired(flagVersion)

	return cmd
}

// RunCmd runs the "ocne cluster update" command
func RunCmd(cmd *cobra.Command) error {
	log.Info("ocne cluster update command not yet implemented")
	return nil
}
