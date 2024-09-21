// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"github.com/containers/image/v5/transports/alltransports"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/cmd/flags"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/image/create"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/image"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
)

const (
	CommandName = "create"
	helpShort   = "Create an image for a specific provider"
	helpLong    = `Create an image for a specific ignition provider, such as OCI.`
	helpExample = `
ocne image create --arch amd64
ocne image create --arch arm64 --type oci
`
)

var config types.Config
var clusterConfig types.ClusterConfig
var clusterConfigPath string
var createOptions create.CreateOptions
var flagArchitectureHelp = "The architecture of the image to create, allowed values: " + strings.Join(flags.ValidArchs, ", ")

const (
	flagProviderType      = "type"
	flagProviderTypeShort = "t"
	flagProviderTypeHelp  = "The provider type, default is oci"
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

	cmd.Flags().StringVarP(&config.KubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&createOptions.ProviderType, flagProviderType, flagProviderTypeShort, create.ProviderTypeOCI, flagProviderTypeHelp)
	cmd.Flags().StringVarP(&createOptions.Architecture, flags.FlagArchitecture, flags.FlagArchitectureShort, "amd64", flagArchitectureHelp)
	cmd.Flags().StringVarP(&clusterConfig.KubeVersion, constants.FlagVersionName, constants.FlagVersionShort, "", constants.FlagKubernetesVersionHelp)

	return cmd
}

// RunCmd runs the "ocne image create" command
func RunCmd(cmd *cobra.Command) error {
	if err := flags.ValidateArchitecture(createOptions.Architecture); err != nil {
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

	// Try to be polite by accepting "regular" container registry formats.
	// Not everyone is familiar with the requirements for ostree.
	_, _, _, err = image.ParseOstreeReference(cc.OsRegistry)
	if err != nil {
		cc.OsRegistry = fmt.Sprintf("ostree-unverified-registry:%s", cc.OsRegistry)
	}

	// Make sure we create the new image using the base image that goes with the requested version of k8s.
	// Note that c.BootVolumeContainerImage has the image that will be used to create the ephemeral cluster where
	// we spin up a pod to create the custom image (which might be different than the base image we use to
	// create the custom image).
	cc.BootVolumeContainerImage = cmdutil.EnsureBootImageVersion(cc.KubeVersion, cc.BootVolumeContainerImage)

	// if the user has not overridden the osTag and the requested k8s version is not the default, make the osTag
	// match the k8s version
	if cc.OsTag == pkgconst.KubeVersion && cc.KubeVersion != pkgconst.KubeVersion {
		cc.OsTag = cc.KubeVersion
	}

	log.Info("Creating Image")
	err = create.Create(c, cc, createOptions)

	return err
}
