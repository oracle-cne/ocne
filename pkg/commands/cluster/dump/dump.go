// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dump

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// Options are the options for the dump command
type Options struct {
	// KubeConfigPath is the path to the optional kubeconfig file
	KubeConfigPath string

	// CuratedResources true will dump resource manifests, except Secrets and ConfigMaps
	CuratedResources bool

	// NodeDumpForClusterInfo dumps a subset of node info if true
	NodeDumpForClusterInfo bool

	// IncludeConfigMap true will dump ConfigMaps
	IncludeConfigMap bool

	// SkipPodLogs true will skip pod logs
	SkipPodLogs bool

	// SkipCluster true means to show the details of the cluster
	SkipCluster bool

	// SkipNodes true means to show the details of each node
	SkipNodes bool

	// SkipRedact true mean skip redaction of cluster and node data
	SkipRedact bool

	// NodeNames are the names of the nodes to dump
	NodeNames []string

	// Namespaces are the names of the namespaces to dump
	Namespaces []string

	// Output directory for the dump.  Must be empty or not exist
	OutDir string

	// Quiet means don't log cluster dump info
	Quiet bool

	// ArchiveFile is the file path of the archive file that will be generated
	ArchiveFile string

	// Managed determines whether the managedField metadata is dumped with a resource
	Managed bool

	// JSON determines whether the Kubernetes resources should be outputted in JSON format
	JSON bool
}

// Dump the cluster
func Dump(o Options) error {
	var err error
	if o.OutDir != "" {
		if err = validate(&o); err != nil {
			return err
		}
	} else {
		o.OutDir, err = file.CreateOcneTempDir(string(uuid.NewUUID()))
		if err != nil {
			return err
		}
		defer os.RemoveAll(o.OutDir)
	}

	// get a kubernetes client
	restConfig, kubeClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}

	// sanity check to make sure we can access cluster
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// ensure the Namespace exists
	err = k8s.CreateNamespaceIfNotExists(kubeClient, constants.OCNESystemNamespace)
	if err != nil {
		return err
	}

	// If the user specifies a node name that is not present, return
	nodeNames, err := determineNodeNames(o, kubeClient)
	if err != nil {
		return err
	}

	// Populate the map of the nodes and their hashes, in case sanitization is needed later
	err = populateNodeMap(kubeClient)
	if err != nil {
		return err
	}

	// Dump both the nodes and the cluster in background goroutines.
	// Errors are not fatal, just logged.  This is a best effort command
	// where a partial dump might be produced.
	wg := sync.WaitGroup{}
	if !o.SkipNodes {
		wg.Add(1)
		log.Infof("Collecting node data")
		go func() {
			err := dumpNodes(o, kubeClient, nodeNames)
			if err != nil {
				log.Errorf("Error dumping nodes: %s", err.Error())
			}
			wg.Done()
		}()
	}

	if !o.SkipCluster {
		wg.Add(1)
		if !o.Quiet {
			log.Infof("Collecting cluster data")
		}
		go func() {
			dynCli, err := client.GetDynamicClient(restConfig)
			if err != nil {
				log.Errorf("Error dumping cluster: %s", err.Error())
			}
			err = dumpCluster(o, kubeClient, dynCli)
			if err != nil {
				log.Errorf("Error dumping cluster: %s", err.Error())
			}
			wg.Done()
		}()

	}
	wg.Wait()

	if !o.SkipCluster {
		// cluster info cannot be captured until all the node files have been dumped
		// so do it after all the goroutines are done
		capture.CaptureClusterInfo(o.SkipNodes, kubeClient, restConfig, o.KubeConfigPath, o.OutDir, o.SkipRedact, nodeNames)
	}
	if o.ArchiveFile != "" {
		err = CreateReportArchive(o.OutDir, o.ArchiveFile)
		if err != nil {
			return err
		}
	}
	if !o.Quiet && o.ArchiveFile == "" {
		log.Infof("Cluster dump successfully completed, files written to %s", o.OutDir)
	} else if !o.Quiet && o.ArchiveFile != "" {
		log.Infof("Cluster dump successfully completed, archive file written to %s", o.ArchiveFile)
	}

	return nil
}

// validate the options, this will make the paths absolute
func validate(o *Options) error {
	if o.OutDir == "" {
		return fmt.Errorf("Output directory must be specified")
	}

	d, err := file.AbsDir(o.OutDir)
	if err != nil {
		return err
	}
	o.OutDir = d
	if err := os.MkdirAll(o.OutDir, 0744); err != nil {
		return err
	}

	return nil
}

// CreateReportArchive creates the .tar.gz file specified by the archiveFile from the files in captureDir e
func CreateReportArchive(captureDir string, archiveFilePath string) error {
	directoryPath := filepath.Dir(archiveFilePath)
	if _, err := os.Stat(directoryPath); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(directoryPath, 0744); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	archiveFile, err := os.Create(archiveFilePath)
	if err != nil {
		return err
	}
	err = os.Chmod(archiveFilePath, 0744)
	if err != nil {
		return err
	}
	defer archiveFile.Close()
	// Create new Writers for gzip and tar
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	walkFn := func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo.Mode().IsDir() {
			return nil
		}
		var filePath string
		filePath = path[len(captureDir):]
		fileReader, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		fih, err := tar.FileInfoHeader(fileInfo, filePath)
		if err != nil {
			return err
		}

		fih.Name = filePath
		err = tarWriter.WriteHeader(fih)
		if err != nil {
			return err
		}
		_, err = io.Copy(tarWriter, fileReader)
		if err != nil {
			return err
		}
		return nil
	}

	if err := filepath.Walk(captureDir, walkFn); err != nil {
		return err
	}
	return nil
}
