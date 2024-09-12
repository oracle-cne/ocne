// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/uuid"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/info"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/util/strutil"
)

const (
	CommandName = "info"
	helpShort   = "Get cluster information"
	helpLong    = `Get overall cluster information along with node level information.`
	helpExample = `
  ocne cluster info 

  # get cluster info, including data from all nodes.
  ocne cluster info

  # get cluster info, including data from node1 and node2.
  ocne cluster info -N node1,node2

  # get cluster info, skipping the node data.
  ocne cluster info --skip-nodes
`
)

var options info.Options
var nodes string

const (
	flagNodes      = "nodes"
	flagNodesShort = "N"
	flagNodesHelp  = "A comma separated list of nodes, default is all nodes"

	flagSkipNodes      = "skip-nodes"
	flagSkipNodesShort = "s"
	flagSkipNodesHelp  = "Skip data from nodes"
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

	cmd.Flags().StringVarP(&options.KubeConfigPath, constants.FlagKubeconfig, constants.FlagKubeconfigShort, "", constants.FlagKubeconfigHelp)

	cmd.Flags().StringVarP(&nodes, flagNodes, flagNodesShort, "", flagNodesHelp)
	cmd.Flags().BoolVarP(&options.SkipNodes, flagSkipNodes, flagSkipNodesShort, false, flagSkipNodesHelp)

	return cmd
}

// RunCmd runs the "ocne cluster info" command
func RunCmd(cmd *cobra.Command) error {
	var nodeNames []string
	if len(nodes) > 0 {
		nodeNames = strutil.TrimArray(strings.Split(strings.Trim(nodes, "\""), ","))
	}
	// Dump the node info into a temp directory.  This is needed for cluster info
	if !options.SkipNodes {
		tmpDir, err := file.CreateOcneTempDir(string(uuid.NewUUID()))
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)
		options.RootDumpDir = tmpDir
		if err := dumpNodes(nodeNames, tmpDir); err != nil {
			return err
		}
	}

	// Execute the cluster info command
	options.NodeNames = nodeNames
	err := info.Info(options)
	return err
}

// dump a subset of node info
func dumpNodes(nodeNames []string, outDir string) error {
	dumpOptions := dump.Options{
		KubeConfigPath:         options.KubeConfigPath,
		NodeDumpForClusterInfo: true,
		NodeNames:              nodeNames,
		OutDir:                 outDir,
		Quiet:                  true,
		SkipCluster:            true,
		SkipNodes:              options.SkipNodes,
		SkipPodLogs:            true,
		SkipRedact:             true,
	}
	return dump.Dump(dumpOptions)
}
