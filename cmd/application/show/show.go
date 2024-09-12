// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package show

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/show"
)

const (
	CommandName = "show"
	helpShort   = "Show application details"
	helpLong    = `Show details about a particular application installed into a Kubernetes
cluster`
	helpExample = `
ocne application show --release appRelease
`
)

var kubeConfig string
var releaseName string
var namespace string
var computed bool
var difference bool

const (
	flagRelease      = "release"
	flagReleaseShort = "r"
	flagReleaseHelp  = "The name of the release of the application"

	flagNamespace      = "namespace"
	flagNamespaceShort = "n"
	flagNamespaceHelp  = "The Kubernetes namespace that the application is installed into. If this value is not provided, the namespace from the current context of the kubeconfig is used"

	flagComputed      = "computed"
	flagComputedShort = "c"
	flagComputedHelp  = "If this flag is set, the complete configuration for the application is displayed"

	flagDifference      = "difference"
	flagDifferenceShort = "d"
	flagDifferenceHelp  = "If this flag is set, then the overrides/user-supplied values along with the complete configuration is displayed for the application"
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
	cmd.Flags().BoolVarP(&computed, flagComputed, flagComputedShort, false, flagComputedHelp)
	cmd.Flags().BoolVarP(&difference, flagDifference, flagDifferenceShort, false, flagDifferenceHelp)
	cmd.MarkFlagRequired(flagRelease)

	return cmd
}

// RunCmd runs the "ocne application show" command
func RunCmd(cmd *cobra.Command) error {
	output, err := show.Show(application.ShowOptions{
		KubeConfigPath: kubeConfig,
		Namespace:      namespace,
		ReleaseName:    releaseName,
		Computed:       computed,
		Difference:     difference,
	})
	if err != nil {
		return err
	}
	fmt.Print(output)

	return nil
}
