// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dump

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture/sanitize"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
)

const (
	podNamePrefix = "ocne-dump-node"
	cmName        = "ocne-dump-node"
	remoteOutPath = "/tmp/ocne/out"
	envNodeName   = "NODE_NAME"
	envDumpFull   = "OCNE_DUMP_FULL"
)

// dumpNodes dumps data from the specified nodes in the cluster
func dumpNodes(o Options, kubeClient kubernetes.Interface, nodeNames []string) error {
	// create a set to keep track of which nodes are dumped
	podSet := make(map[string]*k8s.PodOptions)
	namespace := constants.OCNESystemNamespace

	// create the configmap with the scripts that will run on the pod. First delete cm if exists.
	// the pod mounts this configmap
	if err := k8s.DeleteConfigmap(kubeClient, namespace, cmName); err != nil {
		return err
	}
	defer k8s.DeleteConfigmap(kubeClient, namespace, cmName)
	if err := k8s.CreateConfigMapWithData(kubeClient, namespace, cmName,
		map[string]string{dumpScriptName: dumpScript,
			dumpFullScriptName:   dumpFullScript,
			dumpSubsetScriptName: dumpSubsetScript},
	); err != nil {
		return err
	}

	// create a pod on each node
	for i, _ := range nodeNames {
		nodeName := nodeNames[i]
		podName := fmt.Sprintf("%s-%s", podNamePrefix, nodeName)

		dumpFull := "TRUE"
		if o.NodeDumpForClusterInfo {
			dumpFull = "FALSE"
		}
		dumpImage := k8s.GenerateOLImageToUse()

		p := &k8s.PodOptions{
			NodeName:      nodeName,
			Namespace:     namespace,
			PodName:       podName,
			ImageName:     dumpImage,
			ConfigMapName: cmName,
			HostPID:       true,
			HostNetwork:   true,
			Env: []corev1.EnvVar{
				{Name: envNodeName, Value: nodeName},
				{Name: envDumpFull, Value: dumpFull},
			},
		}

		if err := k8s.DeletePod(kubeClient, namespace, podName); err != nil {
			return err
		}
		defer k8s.DeletePod(kubeClient, p.Namespace, podName)
		if err := k8s.CreateScriptsPod(kubeClient, p); err != nil {
			return err
		}
		podSet[nodeName] = p
	}

	// Do parallel dump of each node
	wg := sync.WaitGroup{}
	wg.Add(len(podSet))
	for _, p := range podSet {
		go func() {
			var outDir string
			if o.SkipRedact {
				outDir = filepath.Join(o.OutDir, "nodes", p.NodeName)
			} else {
				outDir = filepath.Join(o.OutDir, "nodes", sanitize.RedactionPrefix+sanitize.GetShortSha256Hash(p.NodeName))
			}

			// Dump the node using a pod.  Just log if error, it is not fatal
			if err := waitForPodThenDump(o, *p, outDir); err != nil {
				log.Errorf("Error dumping node %s: %s", p.NodeName, err.Error())
			}
			if !o.SkipRedact {
				if err := sanitize.SanitizeFilesInDirTree(outDir); err != nil {
					log.Errorf("Error accessing files in directory %s: %s.  Manually delete this directory and its contents, it may have sensitive data", outDir, err.Error())
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return nil
}

// waitForPodThenDump waits for a pod to be ready then dumps it
func waitForPodThenDump(o Options, p k8s.PodOptions, outDir string) error {
	// A new kcConfig for each node.  This needs to be done for each node
	restConfig, kubeClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}
	// get config needed to use kubectl, each thread needs its own kcConfig since it has the error/output buffer
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, o.KubeConfigPath, constants.OCNESystemNamespace, nil, false)
	if err != nil {
		return err
	}
	log.Debugf("Collecting data on node %s", p.NodeName)
	if err := k8s.WaitForPod(kubeClient, p.Namespace, p.PodName); err != nil {
		return err
	}

	// run the script from the pod to get info
	if err := kubectl.RunScript(kcConfig, p.PodName, constants.ScriptMountPath, dumpScriptName); err != nil {
		return err
	}

	// copy the output files from the pod to the local directory for the node
	if err := os.MkdirAll(outDir, 0744); err != nil {
		return err
	}
	cc := &kubectl.CopyConfig{
		KubectlConfig: kcConfig,
		FilePaths:     []kubectl.FilePath{{LocalPath: outDir, RemotePath: remoteOutPath}},
		PodName:       p.PodName,
	}
	if err := kubectl.CopyFilesFromPod(cc, fmt.Sprintf("Copying data from node %s to the local system", p.NodeName)); err != nil {
		return err
	}
	return nil
}

// determineNodeNames determines which nodes should be dumped. Return either the user provided node names or
// the names of all the nodes in the cluster user provided node names are validated.
func determineNodeNames(o Options, cli kubernetes.Interface) ([]string, error) {
	existingNodes, err := k8s.GetNodeNames(cli)
	if err != nil {
		return nil, err
	}

	return determineNames(existingNodes, o.NodeNames, "node")
}
