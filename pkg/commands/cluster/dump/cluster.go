// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dump

import (
	"fmt"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util/strutil"
	"os"
	"path/filepath"
)

const clusterDir = "cluster"

func dumpCluster(o Options, cli kubernetes.Interface, dynCli dynamic.Interface) error {
	clusterDir := filepath.Join(o.OutDir, clusterDir)
	if err := os.MkdirAll(clusterDir, 0744); err != nil {
		return err
	}

	namespaces, err := determineNamespaces(o, cli)
	if err != nil {
		return err
	}
	cp := capture.CaptureParams{
		KubeClient:        cli,
		DynamicClient:     dynCli,
		Namespaces:        namespaces,
		RootDumpDir:       o.OutDir,
		ClusterDumpDir:    clusterDir,
		IncludeConfigMaps: o.IncludeConfigMap,
		SkipPodLogs:       o.SkipPodLogs,
		Redact:            !o.SkipRedact,
	}
	if o.CuratedResources {
		if err = capture.CaptureCuratedResources(cp); err != nil {
			return err
		}
	} else {
		if err = capture.CaptureAllResources(cp); err != nil {
			return err
		}
	}

	return nil
}

// determineNamespaces returns either the user provided namespace or
// the names of all the namespaces in the cluster.
// User provided names are validated.
func determineNamespaces(o Options, cli kubernetes.Interface) ([]string, error) {
	existingNamespaces, err := k8s.GetNamespaces(cli)
	if err != nil {
		return nil, err
	}

	return determineNames(existingNamespaces, o.Namespaces, "namespace")
}

// populateNodeMap gathers the names of nodes in the cluster and populates the node map to be used by the sanitize function
func populateNodeMap(cli kubernetes.Interface) error {
	nodeList, err := k8s.GetNodeList(cli)
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		capture.PutIntoNodeNamesIfNotPresent(node.Name)
	}
	return nil
}

// determineNames returns either the user provided names or names of resources in the cluster
// User provided names are validated.
func determineNames(resNames []string, specifiedNames []string, nameType string) ([]string, error) {
	names := resNames
	if len(specifiedNames) > 0 {
		var missing []string
		set := strutil.SliceToSet(resNames)
		for i, n := range specifiedNames {
			if _, ok := set[n]; !ok {
				missing = append(missing, specifiedNames[i])
			}

		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("The following %s resources do not exist: %v", nameType, missing)
		}
		names = specifiedNames
	}
	return names, nil
}
