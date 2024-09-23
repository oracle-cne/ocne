// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture/sanitize"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
)

const (
	controlPlaneRole = "control plane"
	workerRole       = "worker"
)

// Options are the options for the info command
// If you want to dump more info from pods, change the dump/scripts.go/dumpSubsetScript script, then
// change extractNodeInfo and display.go to display the information
type Options struct {
	// KubeConfigPath is the path to the optional kubeconfig file
	KubeConfigPath string

	// KubeClient is the optional client-go client
	KubeClient kubernetes.Interface

	// SkipCluster true means to show the details of the cluster
	SkipCluster bool

	// SkipNodes true means to show the details of each node
	SkipNodes bool

	// NodeNames are the names of the nodes to info
	NodeNames []string

	// RootDumpDir contains the node dump files used by cluster info
	RootDumpDir string

	// Writer writes the cluster info data. Default is os.Stdout
	Writer io.Writer
}

type clusterInfo struct {
	kubernetesVersion            string
	nodeInfos                    []*nodeInfo
	numControlPlaneNodes         int
	numWorkerNodes               int
	numNodesWithUpdatesAvailable int
}

type nodeInfo struct {
	// Node is the Kubernetes node information
	node *corev1.Node

	// updateAvailable is true if an update is available
	updateAvailable bool

	// True if control plane node
	controlPlane bool

	// role is control plane or worker
	role string

	// nodeDump contains the node dump subset
	nodeDump *nodeDumpData
}

// nodeDumpData is data that was dump from the node (by a pod)
type nodeDumpData struct {
	updateYAML string

	ostreeRefs string
}

// Info gets the cluster info
func Info(o Options) error {
	if err := validate(&o); err != nil {
		return err
	}

	// get a kubernetes client
	kubeClient := o.KubeClient
	if kubeClient == nil {
		var err error
		_, kubeClient, err = client.GetKubeClient(o.KubeConfigPath)
		if err != nil {
			return err
		}
	}

	// sanity check to make sure we can access cluster
	if _, err := k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// ensure the Namespace exists
	if err := k8s.CreateNamespaceIfNotExists(kubeClient, constants.OCNESystemNamespace); err != nil {
		return err
	}

	// Get the nodes
	ci := clusterInfo{}
	nodeList, err := k8s.GetNodeList(kubeClient)
	if err != nil {
		return err
	}
	for i, node := range nodeList.Items {
		nodeDumpInfo, err := extractNodeInfo(o.SkipNodes, o.RootDumpDir, node.Name)
		if err != nil {
			return err
		}

		cp, role := k8s.GetRole(&node)
		ci.nodeInfos = append(ci.nodeInfos, &nodeInfo{
			node:            &nodeList.Items[i],
			updateAvailable: k8s.IsUpdateAvailable(&node),
			controlPlane:    cp,
			role:            role,
			nodeDump:        nodeDumpInfo,
		})
	}

	loadSummaryInfo(&ci)
	displayAllInfo(&ci, o.Writer)

	return nil
}

// validate the options, this will make the paths absolute
func validate(o *Options) error {
	if o.Writer == nil {
		o.Writer = os.Stdout
	}
	return nil
}

// extractNodeInfo extracts info from node dump files.  If the node directory doesn't exist then return nil
func extractNodeInfo(skipNodes bool, outDir string, nodeName string) (*nodeDumpData, error) {
	// This checks to see both filepaths, covering the cases where a redacted dump has occurred and when a non-redacted dump has occurred
	nodeDir, err := getNodePath(outDir, nodeName)
	if err != nil {
		return nil, err
	}
	// Read the files downloaded from the node
	updateYAML, err := os.ReadFile(filepath.Join(nodeDir, "update.yaml"))
	if err != nil {
		// if update.yaml is missing then the cluster is not OCNE 2.0, just skip it
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	ostreeRefs, err := os.ReadFile(filepath.Join(nodeDir, "ostree-refs.out"))
	if err != nil {
		// if ostree is missing then the cluster is not OCNE 2.0, just skip it
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	ni := nodeDumpData{
		updateYAML: string(updateYAML),
		ostreeRefs: string(ostreeRefs),
	}

	return &ni, nil
}

// getNodePath returns a path for a node in the dump, if it exists and an error indicating whether it is a valid path
func getNodePath(outDir string, nodeName string) (string, error) {
	unSanitizedPath := filepath.Join(outDir, "nodes", nodeName)
	_, err := os.Stat(unSanitizedPath)
	if err == nil {
		return unSanitizedPath, nil
	}
	sanitizedPath := filepath.Join(outDir, "nodes", sanitize.RedactionPrefix+sanitize.GetShortSha256Hash(nodeName))
	_, err = os.Stat(sanitizedPath)
	if err == nil {
		return sanitizedPath, nil
	}
	return "", fmt.Errorf("A valid nodePath for a node was not found in the cluster dump")
}
