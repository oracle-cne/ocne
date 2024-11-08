// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dump

import (
	"fmt"
	"github.com/oracle-cne/ocne/cmd/constants"
	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump"
	"github.com/oracle-cne/ocne/pkg/util/strutil"
	"github.com/spf13/cobra"
	"strings"
)

const (
	CommandName = "dump"
	helpShort   = "Dump the cluster"
	helpLong    = `Dump cluster and node data into a local directory. By default, all cluster resources are included, except Secrets and ConfigMaps.`
	helpExample = `
  # dump all cluster manifests and node data
  ocne cluster dump --output-directory /tmp/dump

  # dump all cluster manifests and node data for nodes node1 and node2
  ocne cluster dump -d /tmp/dump -N node1,node2

  # dump all cluster manifests from the default and sales namespaces, along with cluster-wide resources and node data.
  ocne cluster dump -d /tmp/dump -n default,sales

  # dump curated cluster manifests, skipping the node data.
  ocne cluster dump -d /tmp/dump -c --skip-nodes

  # dump all cluster manifests in default namespace, skipping the node data.
  ocne cluster dump -d /tmp/dump -n "default" --skip-nodes

  # dump all cluster manifests in default namespace to an archive file, skipping the node data.
  ocne cluster dump -z /tmp/dump.tgz -n "default" --skip-nodes
`
)

var options dump.Options
var nodes string
var namespaces string

const (
	flagCurated      = "curated-resources"
	flagCuratedShort = "c"
	flagCuratedHelp  = "Dump manifests from a curated subset of cluster resources. By default, all cluster resources are dumped, except Secrets and ConfigMaps"

	flagConfigMap      = "include-configmaps"
	flagConfigMapShort = "m"
	flagConfigMapHelp  = "Include ConfigMaps in the cluster dump, not valid when the curated-resources flag is set"

	flagNamespaces      = "namespaces"
	flagNamespacesShort = "n"
	flagNamespacesHelp  = "A comma separated list of namespaces, default is all namespaces"

	flagNodes      = "nodes"
	flagNodesShort = "N"
	flagNodesHelp  = "A comma separated list of nodes, default is all nodes"

	flagOut      = "output-directory"
	flagOutShort = "d"
	flagOutHelp  = "The output directory where the data will be written"

	flagSkipCluster      = "skip-cluster"
	flagSkipClusterShort = "r"
	flagSkipClusterHelp  = "Skip data from cluster"

	flagSkipNodes      = "skip-nodes"
	flagSkipNodesShort = "s"
	flagSkipNodesHelp  = "Skip data from nodes"

	flagSkipPodLogs      = "skip-pod-logs"
	flagSkipPodLogsShort = "p"
	flagSkipPodLogsHelp  = "Skip pod logs in the cluster dump"

	flagSkipRedaction      = "skip-redaction"
	flagSkipRedactionShort = "t"
	flagSkipRedactionHelp  = "Skip redaction of sensitive data"

	flagGenerateArchive      = "generate-archive"
	flagGenerateArchiveShort = "z"
	flagGenerateClusterHelp  = "Generate an archive instead of dumping files to an output directory.  The filename must end with .tgz or .tar.gz"

	flagManaged      = "managed"
	flagManagedShort = "f"
	flagManagedHelp  = "Remove managedField data from Kubernetes resource output"

	flagToJSON      = "to-json"
	flagToJSONShort = "j"
	flagToJSONHelp  = "Output Kubernetes resources in JSON format"
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

	cmd.Flags().BoolVarP(&options.CuratedResources, flagCurated, flagCuratedShort, false, flagCuratedHelp)
	cmd.Flags().BoolVarP(&options.IncludeConfigMap, flagConfigMap, flagConfigMapShort, false, flagConfigMapHelp)
	cmd.Flags().StringVarP(&namespaces, flagNamespaces, flagNamespacesShort, "", flagNamespacesHelp)
	cmd.Flags().StringVarP(&nodes, flagNodes, flagNodesShort, "", flagNodesHelp)
	cmd.Flags().StringVarP(&options.OutDir, flagOut, flagOutShort, "", flagOutHelp)
	cmd.Flags().BoolVarP(&options.SkipCluster, flagSkipCluster, flagSkipClusterShort, false, flagSkipClusterHelp)
	cmd.Flags().BoolVarP(&options.SkipNodes, flagSkipNodes, flagSkipNodesShort, false, flagSkipNodesHelp)
	cmd.Flags().BoolVarP(&options.SkipPodLogs, flagSkipPodLogs, flagSkipPodLogsShort, false, flagSkipPodLogsHelp)
	cmd.Flags().BoolVarP(&options.SkipRedact, flagSkipRedaction, flagSkipRedactionShort, false, flagSkipRedactionHelp)
	cmd.Flags().StringVarP(&options.ArchiveFile, flagGenerateArchive, flagGenerateArchiveShort, "", flagGenerateClusterHelp)
	cmd.Flags().BoolVarP(&options.Managed, flagManaged, flagManagedShort, false, flagManagedHelp)
	cmd.Flags().BoolVarP(&options.ToJSON, flagToJSON, flagToJSONShort, false, flagToJSONHelp)

	cmd.MarkFlagsMutuallyExclusive(flagOut, flagGenerateArchive)
	cmd.MarkFlagsMutuallyExclusive(flagSkipCluster, flagCurated)
	cmd.MarkFlagsMutuallyExclusive(flagSkipCluster, flagNamespaces)
	cmd.MarkFlagsMutuallyExclusive(flagSkipCluster, flagConfigMap)
	cmd.MarkFlagsMutuallyExclusive(flagCurated, flagConfigMap)

	return cmd
}

// RunCmd runs the "ocne cluster dump" command
func RunCmd(cmd *cobra.Command) error {
	if options.OutDir == "" && options.ArchiveFile == "" {
		return fmt.Errorf("An output directory or an archive file path must be specified")
	}

	if options.ArchiveFile != "" && !strings.HasSuffix(options.ArchiveFile, ".tgz") && !strings.HasSuffix(options.ArchiveFile, ".tar.gz") {
		return fmt.Errorf("An archive file path must end in .tgz or .tar.gz")
	}

	if len(nodes) > 0 {
		options.NodeNames = strutil.TrimArray(strings.Split(strings.Trim(nodes, "\""), ","))
	}
	if len(namespaces) > 0 {
		options.Namespaces = strutil.TrimArray(strings.Split(strings.Trim(namespaces, "\""), ","))
	}
	err := dump.Dump(options)
	return err
}
