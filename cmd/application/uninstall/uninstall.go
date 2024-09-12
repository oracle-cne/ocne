// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/uninstall"
)

const (
	CommandName = "uninstall"
	helpShort   = "Uninstall an application."
	helpLong    = `Uninstall an application that was installed from the catalog.`
	helpExample = `
ocne application uninstall --release appRelease
`
)

var kubeConfig string
var namespace string
var releaseName string

const (
	flagRelease      = "release"
	flagReleaseShort = "r"
	flagReleaseHelp  = "The name of the release of the application"

	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The Kubernetes namespace that the application is installed into. If this value is not provided, the namespace from the current context of the kubeconfig is used"
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
	cmd.Flags().StringVarP(&namespace, flagNamespace, flagNamespaceShort, "", flagNamespaceHelp)
	cmd.Flags().StringVarP(&releaseName, flagRelease, flagReleaseShort, "", flagReleaseHelp)
	cmd.MarkFlagRequired(flagRelease)

	return cmd
}

// RunCmd runs the "ocne application install" command
func RunCmd(cmd *cobra.Command) error {
	// Run the function to gather the release information as Go Structs
	err := uninstall.Uninstall(application.UninstallOptions{
		KubeConfigPath: kubeConfig,
		Namespace:      namespace,
		ReleaseName:    releaseName,
	})
	if err != nil {
		return err
	}
	log.Infof("%s uninstalled successfully", releaseName)
	return nil
}
