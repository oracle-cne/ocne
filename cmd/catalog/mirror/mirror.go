// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mirror

import (
	"errors"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/mirror"
	"github.com/oracle-cne/ocne/pkg/config/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CommandName = "mirror"
	helpShort   = "Mirror container images in a catalog"
	helpLong    = `Clone the container images used by applications in an application catalog and push them to a private registry or download them to a .tgz file`
	helpExample = `
ocne catalog mirror --name mycatalog --destination other-container-registry.io --push --config example-path/OCNE-Configuration-File --download
`
)

var kubeConfig string
var destination string
var catalogName string
var config types.Config
var clusterConfig types.ClusterConfig
var clusterConfigPath string
var defaultRegistry string
var quiet bool
var push bool
var download bool

const (
	flagCatalogName      = "name"
	flagCatalogNameShort = "N"
	flagCatalogNameHelp  = "The name of the catalog to mirror"

	flagConfig      = "config"
	flagConfigShort = "c"
	flagConfigHelp  = "The URI of an Oracle Cloud Native Environment cluster configuration file. If a cluster configuration file is provided, only the applications listed in that file are mirrored"

	flagDestination      = "destination"
	flagDestinationShort = "d"
	flagDestinationHelp  = "The URI of the destination container registry. The images from the application catalog are tagged so that they belong to this registry. Specify --push to push the images"

	flagPush      = "push"
	flagPushShort = "p"
	flagPushHelp  = "Push images from the application catalog to the destination"

	flagQuiet      = "quiet"
	flagQuietShort = "q"
	flagQuietHelp  = "Output only image names and omit all other output"

	flagSource      = "source"
	flagSourceShort = "s"
	flagSourceHelp  = "The source registry to use for images without a registry. By default, this value is container-registry.oracle.com. For example, olcne/headlamp becomes container-registry.oracle.com/olcne/headlamp"

	flagDownload      = "download"
	flagDownloadShort = "t"
	flagDownloadHelp  = "Download images locally to a tar file on the system "
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
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, catalog.InternalCatalog, flagCatalogNameHelp)
	cmd.Flags().StringVarP(&clusterConfigPath, flagConfig, flagConfigShort, "", flagConfigHelp)
	cmd.Flags().StringVarP(&defaultRegistry, flagSource, flagSourceShort, "container-registry.oracle.com", flagSourceHelp)
	cmd.Flags().StringVarP(&destination, flagDestination, flagDestinationShort, "", flagDestinationHelp)
	cmd.Flags().BoolVarP(&push, flagPush, flagPushShort, false, flagPushHelp)
	cmd.Flags().BoolVarP(&quiet, flagQuiet, flagQuietShort, false, flagQuietHelp)
	cmd.Flags().BoolVarP(&download, flagDownload, flagDownloadShort, false, flagDownloadHelp)
	return cmd
}

// RunCmd runs the "ocne catalog mirror" command
func RunCmd(cmd *cobra.Command) error {
	c, cc, err := cmdutil.GetFullConfig(&config, &clusterConfig, clusterConfigPath)
	if err != nil {
		err = errors.New("Configuration error: " + err.Error())
		return err
	}

	if quiet {
		log.SetLevel(log.PanicLevel)
	}

	mo := mirror.Options{
		KubeConfigPath:  kubeConfig,
		CatalogName:     catalogName,
		DestinationURI:  destination,
		ConfigURI:       clusterConfigPath,
		Push:            push,
		Config:          c,
		ClusterConfig:   cc,
		DefaultRegistry: defaultRegistry,
		Download:        download,
	}
	return mirror.Mirror(mo)
}
