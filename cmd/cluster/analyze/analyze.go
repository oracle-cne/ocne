// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"os"

	"github.com/oracle-cne/ocne/pkg/cmdutil"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	CommandName = "analyze"
	helpShort   = "Analyze the cluster and report problems"
	helpLong    = `Analyze a live cluster or a cluster dump.  If no dump directory or archive file is specified, then the live cluster will be analyzed.`
	helpExample = `
ocne cluster analyze
`
)

const (
	flagDumpDir      = "dump-directory"
	flagDumpDirShort = "d"
	flagDumpDirHelp  = "The directory which has the cluster dump. Mutually-exclusive with archive flag."

	flagSkipNodes      = "skip-nodes"
	flagSkipNodesShort = "s"
	flagSkipNodesHelp  = "Skip data from nodes. Only valid for live cluster analyze. This is only valid for a live cluster analysis."

	flagSkipPodLogs      = "skip-pod-logs"
	flagSkipPodLogsShort = "p"
	flagSkipPodLogsHelp  = "Skip collecting pod logs for the analysis. This is only valid for a live cluster analysis."

	flagVerbose      = "verbose"
	flagVerboseShort = "v"
	flagVerboseHelp  = "Display additional detailed information related to the analysis."
)

var dumpOptions dump.Options
var options analyze.Options
var kubeConfig string

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

	// Analyze options
	cmd.Flags().StringVarP(&options.RootDumpDir, flagDumpDir, flagDumpDirShort, "", flagDumpDirHelp)
	cmd.Flags().BoolVarP(&options.Verbose, flagVerbose, flagVerboseShort, false, flagVerboseHelp)

	// Dump options
	cmd.Flags().BoolVarP(&dumpOptions.SkipNodes, flagSkipNodes, flagSkipNodesShort, false, flagSkipNodesHelp)
	cmd.Flags().BoolVarP(&dumpOptions.SkipPodLogs, flagSkipPodLogs, flagSkipPodLogsShort, false, flagSkipPodLogsHelp)

	cmd.MarkFlagsMutuallyExclusive(flagSkipNodes, flagDumpDir)
	cmd.MarkFlagsMutuallyExclusive(flagSkipPodLogs, flagDumpDir)
	cmd.MarkFlagsMutuallyExclusive(flagSkipPodLogs, flagSkipNodes)

	return cmd
}

// RunCmd runs the "ocne cluster analyze" command
func RunCmd(cmd *cobra.Command) error {
	if options.ArchiveFilePath != "" {
		return fmt.Errorf("The archive flag is not yet implemented")
	}
	if options.RootDumpDir == "" && options.ArchiveFilePath == "" {
		tmpDir, err := file.CreateOcneTempDir(string(uuid.NewUUID()))
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)
		options.RootDumpDir = tmpDir
		if err := dumpCluster(tmpDir); err != nil {
			return err
		}
	}
	err := analyze.Analyze(options)

	return err
}

// dump the cluster and optionally the nodes (skipNodes is already set in flags)
func dumpCluster(outDir string) error {
	dumpOptions.KubeConfigPath = options.KubeConfigPath
	dumpOptions.Quiet = true
	dumpOptions.IncludeConfigMap = true
	dumpOptions.OutDir = outDir
	dumpOptions.SkipRedact = true

	return dump.Dump(dumpOptions)
}
