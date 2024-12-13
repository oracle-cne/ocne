// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/update"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	CommandName = "update"
	helpShort   = "Update an application from an application catalog or update the built-in catalog"
	helpLong    = `
Update an application from an application catalog in a Kubernetes cluster.  
This command can also be used to update the built-in catalog.`
	helpExample = `
# Update application with the release name appRelease
ocne application update --release appRelease

# Update the built-in catalog.
ocne app update -b

# Update the application Grafana using the catalog built into the CLI binary
ocne application update --release grafana --namespace grafana --catalog embedded
`
)

var kubeConfig string
var release string
var values string
var version string
var namespace string
var builtin bool
var catalogName string
var resetValues bool

const (
	flagRelease      = "release"
	flagReleaseShort = "r"
	flagReleaseHelp  = "The name of the release of the application"

	flagValues      = "values"
	flagValuesShort = "u"
	flagValuesHelp  = "URI of an application configuration. The format of the configuration depends on the style of application served by the target catalog. In general, it will be a set of Helm values."

	flagVersion      = "version"
	flagVersionShort = "v"
	flagVersionHelp  = "The version of an application to install. By default, the version is the latest stable version of the application"

	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The Kubernetes namespace that the application is installed into. If this value is not provided, the namespace from the current context of the kubeconfig is used"

	flagBuiltIn      = "built-in-catalog"
	flagBuiltInShort = "b"
	flagBuiltInHelp  = "Update the built-in catalog in the ocne-system namespace."

	flagCatalogName      = "catalog"
	flagCatalogNameShort = "c"
	flagCatalogNameHelp  = "The name of the catalog that contains the application."

	flagResetValues     = "reset-values"
	flagResetValuesHelp = "Reset the values to the ones built into the chart. If --values is also provided, it will be treated as a new set of overrides."
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

	cmd.PersistentFlags().StringVarP(&kubeConfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().BoolVarP(&builtin, flagBuiltIn, flagBuiltInShort, false, flagBuiltInHelp)
	cmd.Flags().StringVarP(&values, flagValues, flagValuesShort, "", flagValuesHelp)
	cmd.Flags().StringVarP(&namespace, flagNamespace, flagNamespaceShort, "", flagNamespaceHelp)
	cmd.Flags().StringVarP(&release, flagRelease, flagReleaseShort, "", flagReleaseHelp)
	cmd.Flags().StringVarP(&version, flagVersion, flagVersionShort, "", flagVersionHelp)
	cmd.Flags().StringVarP(&catalogName, flagCatalogName, flagCatalogNameShort, pkgconst.DefaultCatalogName, flagCatalogNameHelp)
	cmd.Flags().BoolVarP(&resetValues, flagResetValues, "", false, flagResetValuesHelp)

	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagRelease)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagVersion)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagNamespace)
	cmd.MarkFlagsMutuallyExclusive(flagBuiltIn, flagCatalogName)

	return cmd
}

// RunCmd runs the "ocne application update" command
func RunCmd(cmd *cobra.Command) error {
	if builtin {
		return update.UpdateInternalCatalog(kubeConfig, pkgconst.OCNESystemNamespace)
	}

	if release == "" {
		return fmt.Errorf("The release flag must be set")
	}

	err := update.Update(application.UpdateOptions{
		Namespace:      namespace,
		KubeConfigPath: kubeConfig,
		CatalogName:    catalogName,
		Version:        version,
		ReleaseName:    release,
		Values:         values,
		ResetValues:    resetValues,
	})
	if err != nil {
		return err
	}
	log.Infof("Application updated successfully")
	return nil
}
