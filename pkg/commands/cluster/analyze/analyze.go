// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze/dumpfiles"
	"github.com/oracle-cne/ocne/pkg/file"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

// Options are the options for the analyze command
type Options struct {
	// KubeConfigPath is the path to the optional kubeconfig file
	KubeConfigPath string

	// RootDumpDir contains the node dump files or the archive
	// Either this field or the ArchiveFilePath must be specified
	RootDumpDir string

	// ArchiveFilePath contains the file path of the archive file
	// Either this field or the RootDumpDir must be specified
	ArchiveFilePath string

	// Verbose controls displaying of analyze details like events
	Verbose bool

	// JSON indicates whether the Kubernetes resources in the cluster dump have Kubernetes Resources in JSON format
	JSON bool
}

// Analyze analyzes a cluster dump
func Analyze(o Options) error {
	dumpDir := o.RootDumpDir
	if o.ArchiveFilePath != "" {
		tmpDir, err := file.CreateOcneTempDir("analyze")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)
		if err := unpackArchive(o.ArchiveFilePath, tmpDir); err != nil {
			return err
		}
		dumpDir = tmpDir
	}

	p := analyzeParams{
		rootDir:        dumpDir,
		clusterDir:     filepath.Join(dumpDir, "cluster"),
		clusterWideDir: filepath.Join(dumpDir, "cluster", "cluster-wide"),
		nameSpacesDir:  filepath.Join(dumpDir, "cluster", "namespaces"),
		nodesDir:       filepath.Join(dumpDir, "nodes"),
		verbose:        o.Verbose,
		writer:         os.Stdout,
		isJSON:         o.JSON,
	}
	if _, err := os.Stat(dumpDir); err != nil {
		return err
	}
	if err := analyzeCluster(&p); err != nil {
		// keep going if error
		log.Errorf("Error analyzing cluster: %v", err)
	}

	if err := analyzeNodes(&p); err != nil {
		// keep going if error
		log.Errorf("Error analyzing nodes: %v", err)
	}

	if err := analyzePods(&p); err != nil {
		// keep going if error
		log.Errorf("Error analyzing pods: %v", err)
	}

	if err := analyzeImages(&p); err != nil {
		// keep going if error
		log.Errorf("Error analyzing images: %v", err)
	}

	fmt.Fprintln(p.writer)
	return nil
}

// unpackArchive unpacks the archive into the dump directory
func unpackArchive(archivePath string, dumpDir string) error {

	return nil
}

// readClusterWideJSONFile reads cluster-wide json or yaml data files and unmarshals them into resources.
// For example, read nodes.json
func readClusterWideJSONOrYAMLFile[T any](p *analyzeParams, fileName string) (*T, error) {
	if p.isJSON {
		return dumpfiles.ReadClusterWideJSONOrYAMLFile[T](p.clusterWideDir, fileName+".json")
	} else {
		return dumpfiles.ReadClusterWideJSONOrYAMLFile[T](p.clusterWideDir, fileName+".yaml")
	}
}

// readNamespacedJSONOrYAMLFiles reads namespaced json or yaml data files and unmarshals them into resources, then
// puts the lists in a map where the namespace is the key.
// For example, read pods.json in all namespaces.
func readNamespacedJSONOrYAMLFiles[T any](p *analyzeParams, fileName string) (map[string]T, error) {
	if p.isJSON {
		return dumpfiles.ReadJSONOrYAMLFiles[T](p.clusterDir, fileName+".json")
	} else {
		return dumpfiles.ReadJSONOrYAMLFiles[T](p.clusterDir, fileName+".yaml")
	}
}

// readNodeSpecificJSONFiles reads json data files for each node and unmarshals them into resources
// For example, podman-inspect-all.json
func readNodeSpecificJSONFiles[T any](p *analyzeParams, fileName string) (map[string]T, error) {
	return dumpfiles.ReadJSONOrYAMLFiles[T](p.nodesDir, fileName)
}

// readClusterWideTextFile reads cluster-wide json data files and unmarshals them into resources.
// For example, read nodes.json
func readClusterWideTextFile[T any](p *analyzeParams, fileName string) (string, error) {
	return dumpfiles.ReadClusterWideTextFile(p.clusterWideDir, fileName)
}

// readNamespacedTextFiles reads namespaced json data files and unmarshals them into resources, then
// puts the lists in a map where the namespace is the key.
// For example, read pods.json in all namespaces.
func readNamespacedTextFiles[T any](p *analyzeParams, fileName string) (map[string]string, error) {
	return dumpfiles.ReadTextFiles(p.clusterDir, fileName)
}

// readNodeSpecificTextFiles reads json data files for each node and unmarshals them into resources
// For example, podman-inspect-all.json
func readNodeSpecificTextFiles[T any](p *analyzeParams, fileName string) (map[string]string, error) {
	return dumpfiles.ReadTextFiles(p.nodesDir, fileName)
}
