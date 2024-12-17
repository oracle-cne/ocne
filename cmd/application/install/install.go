// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CommandName = "install"
	helpShort   = "Install an application from an application catalog or install the built-in catalog"
	helpLong    = `
Install an application from an application catalog in a Kubernetes cluster.  
This command can also be used to install the built-in catalog into any cluster
where the container runtime has the proper registry name configured.`
	helpExample = `
# Install the application named myApplication from myCatalog with the in-cluster name appRelease
ocne application install --name "myApplication" --release "appRelease" --catalog "myCatalog" 

# Install the built-in catalog.
ocne app install -b

# Install Grafana from the catalog named embedded, which is built into in the CLI
ocne application install --name grafana --release grafana --catalog embedded --namespace grafana
`
)

var kubeConfig string
var appName string
var builtin bool
var releaseName string
var values string
var version string
var catalogName string
var namespace string

const (
	flagAppName      = "name"
	flagAppNameShort = "N"
	flagAppNameHelp  = "The name of the application to install"

	flagRelease      = "release"
	flagReleaseShort = "r"
	flagReleaseHelp  = "The application release name. The same application can be installed multiple times, differentiated by release name."

	flagValues      = "values"
	flagValuesShort = "u"
	flagValuesHelp  = "URI of an application configuration. The format of the configuration depends on the style of application served by the target catalog. In general, it will be a set of Helm values."

	flagVersion      = "version"
	flagVersionShort = "v"
	flagVersionHelp  = "The version of an application to install. By default, the version is the latest stable version of the application."

	flagCatalogName      = "catalog"
	flagCatalogNameShort = "c"
	flagCatalogNameHelp  = "The name of the catalog that contains the application."

	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The Kubernetes namespace that the application is installed into. The namespace is created if it does not already exist. If this value is not provided, the namespace from the current context of the kubeconfig is used."

	flagBuiltIn      = "built-in-catalog"
	flagBuiltInShort = "b"
	flagBuiltInHelp  = "Install the built-in catalog into the ocne-system namespace.  The cluster container runtime must be configured with the image registry name."
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
	cmd.Flags().BoolVarP(&builtin, flagBuiltIn, flagBuiltInShort, false, flagBuiltInHelp)
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, pkgconst.DefaultCatalogName, flagCatalogNameHelp)
	cmd.Flags().StringVarP(&appName, flagAppName, flagAppNameShort, "", flagAppNameHelp)
	cmd.Flags().StringVarP(&releaseName, flagRelease, flagReleaseShort, "", flagReleaseHelp)
	cmd.Flags().StringVarP(&values, flagValues, flagValuesShort, "", flagValuesHelp)
	cmd.Flags().StringVarP(&version, flagVersion, flagVersionShort, "", flagVersionHelp)
	cmd.Flags().StringVarP(&namespace, flagNamespace, flagNamespaceShort, "", flagNamespaceHelp)

	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagCatalogName)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagAppName)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagRelease)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagVersion)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagNamespace)

	return cmd
}

// RunCmd runs the "ocne application install" command
func RunCmd(cmd *cobra.Command) error {
	if builtin {
		return install.InstallInternalCatalog(kubeConfig, pkgconst.OCNESystemNamespace)
	}

	if appName == "" {
		return fmt.Errorf("The name flag must be set")
	}

	// Get the catalog information
	cat, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: kubeConfig,
		CatalogName:    catalogName,
		Pattern:        appName,
	})
	if err != nil {
		return err
	}

	err = install.Install(application.InstallOptions{
		Namespace:      namespace,
		Catalog:        cat,
		KubeConfigPath: kubeConfig,
		AppName:        appName,
		Version:        version,
		ReleaseName:    releaseName,
		Values:         values,
	})
	if err != nil {
		return err
	}
	log.Infof("Application installed successfully")
	return nil
}
