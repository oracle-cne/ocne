// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package copy

import (
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/copy"
)

const (
	CommandName = "copy"
	helpShort   = "Copy images from one domain to another"
	helpLong    = `Copy images from one domain to another`
	helpExample = `
ocne catalog copy --filepath sourcefile.txt --destinationfilepath destinationfile.txt --destination new.domain.name.com
`
)

var kubeConfig string
var filePath string
var destinationfilepath string
var destination string

const (
	flagFilePath      = "filepath"
	flagFilePathShort = "f"
	flagFilePathHelp  = "The name of the source file populated with the original container images"

	flagDestinationFilePath      = "destinationfilepath"
	flagDestinationFilePathShort = "e"
	flagDestinationFilePathHelp  = "The name of the file to contain new container images"

	flagDestination      = "destination"
	flagDestinationShort = "d"
	flagDestinationHelp  = "The new domain name of the images"
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
	// flag for source, destination, and new domain name
	cmd.Flags().StringVarP(&filePath, flagFilePath, flagFilePathShort, "", flagFilePathHelp)
	cmd.Flags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&destination, flagDestination, flagDestinationShort, "", flagDestinationHelp)
	cmd.Flags().StringVarP(&destinationfilepath, flagDestinationFilePath, flagDestinationFilePathShort, "", flagDestinationFilePathHelp)
	cmd.MarkFlagRequired(flagFilePath)
	cmd.MarkFlagRequired(flagDestination)
	cmd.MarkFlagRequired(flagDestinationFilePathHelp)
	return cmd
}

// RunCmd runs the "ocne catalog copy" command
func RunCmd(cmd *cobra.Command) error {
	err := copy.Copy(catalog.CopyOptions{
		KubeConfigPath:      kubeConfig,
		FilePath:            filePath,
		DestinationFilePath: destinationfilepath,
		Destination:         destination,
	})
	return err
}
