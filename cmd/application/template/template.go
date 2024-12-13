// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"fmt"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/template"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/spf13/cobra"
)

const (
	CommandName = "template"
	helpShort   = "Generate an application configuration template"
	helpLong    = `The format of the template is depends on the
  style of application served by the target catalog. In general, it will be a
  set of Helm values.`
	helpExample = `
# Generate a template for an application
ocne application template --catalog myCatalog --name myApplication

# Generate a template using catalog built into the CLI
ocne application template --name grafana --catalog embedded
`
)

var kubeConfig string
var interactive bool
var catalog string
var name string
var version string

const (
	flagInteractive      = "interactive"
	flagInteractiveShort = "i"
	flagInteractiveHelp  = "Opens the application defined by the EDITOR environment variable and populates it with the template"

	flagCatalog      = "catalog"
	flagCatalogShort = "c"
	flagCatalogHelp  = "The name of the catalog that contains the application"

	flagName      = "name"
	flagNameShort = "N"
	flagNameHelp  = "The name of the application to templatize"

	flagVersion      = "version"
	flagVersionShort = "v"
	flagVersionHelp  = "The version of the application to templatize"
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
	cmd.PersistentFlags().StringVarP(&catalog, flagCatalog, flagCatalogShort, pkgconst.DefaultCatalogName, flagCatalogHelp)
	cmd.PersistentFlags().StringVarP(&name, flagName, flagNameShort, "", flagNameHelp)
	cmd.MarkFlagRequired(flagName)
	cmd.PersistentFlags().StringVarP(&version, flagVersion, flagVersionShort, "", flagVersionHelp)
	cmd.Flags().BoolVarP(&interactive, flagInteractive, flagInteractiveShort, false, flagInteractiveHelp)

	return cmd
}

// RunCmd runs the "ocne application template" command
func RunCmd(cmd *cobra.Command) error {
	output, err := template.Template(application.TemplateOptions{
		KubeConfigPath: kubeConfig,
		AppName:        name,
		Version:        version,
		Interactive:    interactive,
		Catalog:        catalog,
	})
	if err != nil {
		return err
	}
	if interactive {
		err = template.RunInteractiveMode(name, output)
	} else {
		fmt.Println(string(output))
	}
	if err != nil {
		return err
	}
	return nil
}
