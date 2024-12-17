// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"strings"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/cmd/flags"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/image/upload"
	"github.com/oracle-cne/ocne/pkg/config/types"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/spf13/cobra"
)

const (
	CommandName = "upload"
	helpShort   = "Upload an image to object storage"
	helpLong    = `Upload an image to object storage for a specific provider, such as OCI.`
	helpExample = `
ocne image upload
ocne image upload --compartment ocid1.compartment.example --bucket images --type oci --file ~/bootfile.qcow2 --image-name boot-amd64 --arch amd64
`
)

var config types.Config
var clusterConfig types.ClusterConfig
var clusterConfigPath string

var uploadOptions = upload.UploadOptions{
	BucketName:        pkgconst.OciBucket,
	ImageName:         pkgconst.OciImageName,
	KubernetesVersion: pkgconst.KubeVersion,
}
var flagArchitectureHelp = "The architecture of the image to upload, allowed values: " + strings.Join(flags.ValidArchs, ", ")

const (
	flagProviderType      = "type"
	flagProviderTypeShort = "t"
	flagProviderTypeHelp  = "The provider type, default is oci"

	flagImagePath      = "file"
	flagImagePathShort = "f"
	flagImagePathHelp  = "The local image file path"

	flagBucket      = "bucket"
	flagBucketShort = "b"
	flagBucketHelp  = "The name of the object storage bucket to upload the VM image into"

	flagCompartment      = "compartment"
	flagCompartmentShort = "c"
	flagCompartmentHelp  = "The name of the compartment to create the image in"

	flagImageName      = "image-name"
	flagImageNameShort = "i"
	flagImageNameHelp  = "The name of the compute image to create"

	flagDestination      = "destination"
	flagDestinationShort = "d"
	flagDestinationHelp  = "The location to upload to"

	flagKubernetesVersionHelp = "The version of Kubernetes embedded in the image"
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

	cmd.Flags().StringVarP(&uploadOptions.KubeConfigPath, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&clusterConfigPath, constants.FlagConfig, "", "", constants.FlagConfigHelp)
	cmd.Flags().StringVarP(&config.KubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&uploadOptions.ProviderType, flagProviderType, flagProviderTypeShort, upload.ProviderTypeOCI, flagProviderTypeHelp)
	cmd.Flags().StringVarP(&uploadOptions.BucketName, flagBucket, flagBucketShort, pkgconst.OciBucket, flagBucketHelp)
	cmd.Flags().StringVarP(&uploadOptions.CompartmentName, flagCompartment, flagCompartmentShort, "", flagCompartmentHelp)
	cmd.Flags().StringVarP(&uploadOptions.ImagePath, flagImagePath, flagImagePathShort, "", flagImagePathHelp)
	cmd.MarkFlagRequired(flagImagePath)
	cmd.Flags().StringVarP(&uploadOptions.ImageName, flagImageName, flagImageNameShort, pkgconst.OciImageName, flagImageNameHelp)
	cmd.Flags().StringVarP(&uploadOptions.ImageArchitecture, flags.FlagArchitecture, flags.FlagArchitectureShort, "", flagArchitectureHelp)
	cmd.Flags().StringVarP(&uploadOptions.KubernetesVersion, constants.FlagVersionName, constants.FlagVersionShort, pkgconst.KubeVersion, flagKubernetesVersionHelp)
	cmd.Flags().StringVarP(&uploadOptions.Destination, flagDestination, flagDestinationShort, "", flagDestinationHelp)

	return cmd
}

// RunCmd runs the "ocne image upload" command
func RunCmd(cmd *cobra.Command) error {
	_, cc, err := cmdutil.GetFullConfig(&config, &clusterConfig, clusterConfigPath)
	if err != nil {
		return err
	}

	uploadOptions.ClusterConfig = cc
	if err := flags.ValidateArchitecture(uploadOptions.ImageArchitecture); err != nil {
		return err
	}

	return upload.Upload(uploadOptions)
}
