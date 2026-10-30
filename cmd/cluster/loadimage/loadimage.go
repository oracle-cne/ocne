// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loadimage

import (
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/loadimage"
	"github.com/spf13/cobra"
)

const CommandName = "load-image"

func NewCmd() *cobra.Command {
	var kubeconfig, node string
	cmd := &cobra.Command{
		Use:   CommandName + " ARCHIVE",
		Short: "Load a container image archive onto cluster nodes",
		Long:  "Load a local Docker or OCI image archive into the node container storage used by CRI-O.\nBy default, load the archive onto every node in the cluster. Use --node to select one node.\nThe archive must contain images suitable for the selected nodes. Administrative cluster access is required.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadimage.Load(kubeconfig, node, args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmdutil.SilenceUsage(cmd)
	cmd.Flags().StringVarP(&kubeconfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&node, "node", "N", "", "Load the archive onto this node instead of every cluster node")
	return cmd
}
