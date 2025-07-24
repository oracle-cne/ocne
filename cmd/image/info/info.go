// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"strings"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/cmd/flags"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/image/info"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/config/types"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/spf13/cobra"
)

const (
	CommandName = "info"

	flagImage = "image"
	flagImageShort = "i"
	flagImageHelp = "A container image containing OCK boot media"

	flagFile = "file"
	flagFileShort = "f"
	flagFileHelp = "The path to a qcow2 image to inspect"

	flagPath = "path"
	flagPathShort = "p"
	flagPathHelp = "The path of a file on a filesystem to inspect"

	flagRecursive = "recursive"
	flagRecursiveShort = "r"
	flagRecursiveHelp = "If the inspected path is a directory, recursively list any child directories"

	flagLabel = "label"
	flagLabelShort = "L"
	flagLabelHelp = "A partition label to inspect"

	helpShort = "Display information about OCK boot media"
	helpLong = "Display information about OCK boot media"
	helpExample = "ocne image info --image container-registry.oracle.com/olcne/ock"
)

var config types.Config
var clusterConfig types.ClusterConfig
var clusterConfigPath string
var infoOptions info.InfoOptions
var flagArchitectureHelp = "The architecture of the image to inspect, allowed values: " + strings.Join(flags.ValidArchs, ", ")

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

	cmd.Flags().StringVarP(&infoOptions.Architecture, flags.FlagArchitecture, flags.FlagArchitectureShort, "amd64", flagArchitectureHelp)
	cmd.Flags().StringVarP(&clusterConfig.KubeVersion, constants.FlagVersionName, constants.FlagVersionShort, "", constants.FlagKubernetesVersionHelp)
	cmd.Flags().StringVarP(&clusterConfig.BootVolumeContainerImage, flagImage, flagImageShort, "", flagImageHelp)
	cmd.Flags().StringVarP(&infoOptions.File, flagFile, flagFileShort, "", flagFileHelp)
	cmd.Flags().StringVarP(&infoOptions.Label, flagLabel, flagLabelShort, "", flagLabelHelp)
	cmd.Flags().StringVarP(&infoOptions.Path, flagPath, flagPathShort, "", flagPathHelp)
	cmd.Flags().BoolVarP(&infoOptions.Recursive, flagRecursive, flagRecursiveShort, false, flagRecursiveHelp)

	return cmd
}

// RunCmd runs the "ocne image info" command
func RunCmd(cmd *cobra.Command) error {
	if err := flags.ValidateArchitecture(infoOptions.Architecture); err != nil {
		return err
	}

	c, cc, err := cmdutil.GetFullConfig(&config, &clusterConfig, clusterConfigPath)
	if err != nil {
		return err
	}

	imageTransport := alltransports.TransportFromImageName(cc.BootVolumeContainerImage)
	if imageTransport == nil {
		// No transport protocol detected. Adding docker transport protocol as default.
		cc.BootVolumeContainerImage = "docker://" + cc.BootVolumeContainerImage
	}

	// Fix up the container image name based on the configuration.
	cc.BootVolumeContainerImage, err = cmdutil.EnsureBootImageVersion(cc.KubeVersion, cc.BootVolumeContainerImage)
	if err != nil {
		return err
	}

	// if the user has not overridden the osTag and the requested k8s version is not the default, make the osTag
	// match the k8s version
	if cc.OsTag == pkgconst.KubeVersion && cc.KubeVersion != pkgconst.KubeVersion {
		cc.OsTag = cc.KubeVersion
	}

	err = info.Info(c, cc, infoOptions)

	return err
}
