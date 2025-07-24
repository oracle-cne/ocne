// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package image

import (
	"github.com/oracle-cne/ocne/cmd/common"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/cmd/image/create"
	"github.com/oracle-cne/ocne/cmd/image/info"
	"github.com/oracle-cne/ocne/cmd/image/upload"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/spf13/cobra"
)

const (
	CommandName = "image"
	helpShort   = "Manage ocne images"
	helpLong    = `Manage ocne images by doing the following: creating for a specfic provider, uploading to cloud storage, importing images, etc.`
	helpExample = `
ocne image <subcommand>
`
)

var kubeConfig string

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       CommandName,
		Short:     helpShort,
		Long:      helpLong,
		Args:      common.ArgsCheck,
		ValidArgs: []string{create.CommandName},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmd(cmd)
	}
	cmdutil.SilenceUsage(cmd)
	cmd.Example = helpExample

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.AddCommand(create.NewCmd())
	cmd.AddCommand(upload.NewCmd())
	cmd.AddCommand(info.NewCmd())

	return cmd
}

// RunCmd runs the "ocne image" command
func RunCmd(cmd *cobra.Command) error {
	return nil
}
