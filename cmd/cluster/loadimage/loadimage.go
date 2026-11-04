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
	var kubeconfig, node, selector string
	cmd := &cobra.Command{
		Use:   CommandName + " SOURCE",
		Short: "Load a container image onto cluster nodes",
		Long:  "Load a container image into the node storage used by CRI-O. SOURCE accepts a Docker or OCI archive path, or an image transport reference such as docker://REGISTRY/IMAGE:TAG, docker-archive:PATH[:IMAGE:TAG], oci:PATH[:NAME], oci-archive:PATH[:NAME], or dir:PATH. Archive references select one image. By default, load onto every cluster node. Use --node for one exact node name or --selector for nodes matching a Kubernetes label selector. These options are mutually exclusive. Manifest lists select the Linux image matching each node architecture; single images must match all selected nodes. Selected source names are retained; unnamed OCI sources load by digest. Administrative cluster access is required.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadimage.Load(kubeconfig, node, selector, args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmdutil.SilenceUsage(cmd)
	cmd.Flags().StringVarP(&kubeconfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&node, "node", "N", "", "Load the image onto this node instead of every cluster node")
	cmd.Flags().StringVarP(&selector, "selector", "s", "", "Load onto nodes matching this Kubernetes label selector (for example, pool=development)")
	cmd.MarkFlagsMutuallyExclusive("node", "selector")
	return cmd
}
