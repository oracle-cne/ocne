// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"fmt"

	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/template"
	"github.com/oracle-cne/ocne/pkg/config"
	"github.com/oracle-cne/ocne/pkg/config/types"
	pkgconst "github.com/oracle-cne/ocne/pkg/constants"
	"github.com/spf13/cobra"
)

const (
	CommandName = "template"
	helpShort   = "Outputs a cluster configuration template"
	helpLong    = `Emits a sample cluster configuration that can be customized as needed`
	helpExample = `
ocne cluster template
`
)

var kubeConfig string
var clusterConfigPath string

var opts = template.TemplateOptions{
	Config: types.Config{},
	ClusterConfig: types.ClusterConfig{
		WorkerNodes: config.GenerateUInt16Pointer(pkgconst.WorkerNodes),
	},
}

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
	cmd.Flags().StringVarP(opts.ClusterConfig.Provider, constants.FlagProviderName, constants.FlagProviderNameShort, pkgconst.ProviderTypeOCI, constants.FlagProviderNameHelp)
	cmd.Flags().StringVarP(&clusterConfigPath, constants.FlagConfig, constants.FlagConfigShort, "", constants.FlagConfigHelp)
	return cmd
}

// RunCmd runs the "ocne cluster template" command
func RunCmd(cmd *cobra.Command) error {
	cc, err := cmdutil.GetFullConfig(&opts.Config, &opts.ClusterConfig, clusterConfigPath)
	if err != nil {
		return err
	}

	// if the user has not overridden the osTag and the requested k8s version is not the default, make the osTag
	// match the k8s version
	if *cc.OsTag == pkgconst.KubeVersion && *cc.KubeVersion != pkgconst.KubeVersion {
		cc.OsTag = cc.KubeVersion
	}

	// if cluster name is empty, then default it to ocne
	if *cc.Name == "" {
		*cc.Name = "ocne"
	}

	// if number of control plane nodes is 0, then default it to 1
	if *cc.ControlPlaneNodes == 0 {
		*cc.ControlPlaneNodes = pkgconst.ControlPlaneNodes
	}
	opts.ClusterConfig = *cc
	tmpl, err := template.Template(opts)
	if err != nil {
		return err
	}
	fmt.Println(tmpl)
	return nil
}
