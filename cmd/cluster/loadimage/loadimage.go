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
		Use:   CommandName + " SOURCE",
		Short: "Load a container image onto cluster nodes",
		Long:  "Load a container image into the node storage used by CRI-O.\nSOURCE accepts a Docker or OCI archive path, or an image transport reference such as\ndocker://REGISTRY/IMAGE:TAG, docker-archive:PATH[:IMAGE:TAG], oci:PATH[:NAME],\noci-archive:PATH[:NAME], or dir:PATH. Archive references select one image.\nBy default, load onto every cluster node; --node selects one exact node name.\nManifest lists select the Linux image matching each node architecture. Single images\nmust match all selected nodes. All required images are prepared before loaders start.\nSelected source names are retained; unnamed OCI sources load by digest.\nAdministrative cluster access is required.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadimage.Load(kubeconfig, node, args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmdutil.SilenceUsage(cmd)
	cmd.Flags().StringVarP(&kubeconfig, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)
	cmd.Flags().StringVarP(&node, "node", "N", "", "Load the image onto this node instead of every cluster node")
	return cmd
}
