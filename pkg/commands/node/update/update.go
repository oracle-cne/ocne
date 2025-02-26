// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/oracle-cne/ocne/pkg/cluster/update"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util/script"
)

const (
	envNodeName        = "NODE_NAME"
	envCoreDNSImageTag = "CORE_DNS_IMAGE_TAG"

	PreUpdateModeDefault = "default"
	PreUpdateModeOnly = "only"
	PreUpdateModeSkip = "skip"
)

// UpdateOptions are the options for the update command
type UpdateOptions struct {
	// KubeConfigPath is the path to the optional kubeconfig file
	KubeConfigPath string

	// NodeName is the name of the node to update
	NodeName string

	// DeleteEmptyDir specifies that the node should be drained even if pods are using emptydir
	DeleteEmptyDir bool

	// DisableEviction forces drain to use delete, ignoring PodDisruptionBudget
	DisableEviction bool

	// Timeout to wait for the node to drain
	Timeout string

	// PreUpdateMode determines how to handle the preupdate process
	PreUpdateMode string
}

// Update updates a cluster node with a new CrateOS image and restarts the node
func Update(o UpdateOptions) error {
	namespace := constants.OCNESystemNamespace

	// get a kubernetes client
	restConfig, kubeClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}

	// sanity check to make sure we can access cluster
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// Does check to see whether node requires update
	nodeList, err := k8s.GetNodeList(kubeClient)
	if err != nil {
		return err
	}

	// Do any pre-upgrade work
	if o.PreUpdateMode != PreUpdateModeSkip {
		err = update.Update(restConfig, kubeClient, o.KubeConfigPath, nodeList)
		if err != nil {
			return err
		}
	}

	if o.PreUpdateMode == PreUpdateModeOnly {
		return nil
	}

	// Check whether the desired node to upgrade is a worker node and if the control plane nodes are up-to-date
	// Make sure that current version of worker is less that the control planes
	controlPlaneNodesAcceptable, err := areControlPlaneNodesAcceptable(nodeList, o.NodeName, restConfig, kubeClient, o.KubeConfigPath)
	if err != nil {
		return err
	}
	if !controlPlaneNodesAcceptable {
		return fmt.Errorf("the upgrade on %s cannot be performed, since it either has the same or greater version than some control-plane nodes, or some control-plane nodes have updates available  ", o.NodeName)
	}
	// ensure the Namespace exists
	err = k8s.CreateNamespaceIfNotExists(kubeClient, namespace)

	ignores := []string{}

	// If a control plane node is being updated, there is a chance that
	// it is the one servicing its own update.  There isn't a way to
	// predict that in advance, so close our eyes and hope that any
	// connections errors are for that reason.
	//
	// The error string is something like "next reader: websocket: close 1006 (abnormal closure): unexpected EOF"
	for _, n := range nodeList.Items {
		if n.Name != o.NodeName {
			continue
		}

		if k8s.IsControlPlane(&n) {
			ignores = append(ignores, "unexpected EOF")

			log.Infof("When updating control plane nodes, it is possible to lose connection to the Kubernetes API Server temporarily.  Any upcoming log messages about connection errors can be ignored.")
		}
		break
	}


	// get config needed to use kubectl
	ignores = append(ignores, "The --rebuild-if-modules-changed option is deprecated. Use --refresh instead.")
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, o.KubeConfigPath, namespace, ignores, false)
	if err != nil {
		return err
	}

	err = prepareNode(&o, kubeClient, kcConfig)
	if err != nil {
		return err
	}

	// Reset the error buffer in the kcConfig to avoid catching ignored
	// errors from the previous command.
	kcConfig.ErrBuf.Reset()

	log.Info("Running node update")
	err = script.RunScript(kubeClient, kcConfig, o.NodeName, namespace, "update-node", updateNodeScript, []corev1.EnvVar{
		{Name: envNodeName, Value: o.NodeName},
	})
	if err != nil {
		return err
	}

	// The reboot happens 3 seconds after the script finishes.
	// Wait until the cluster can be accessed, sleep first
	log.Infof("Node %s has been updated and rebooted", o.NodeName)
	time.Sleep(5 * time.Second)
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// wait until the node is ready.
	if err := k8s.WaitUntilNodeIsReady(kubeClient, o.NodeName); err != nil {
		return err
	}
	// un-cordon the node so that pods can be scheduled on it
	if err := uncordonNode(&o, kcConfig); err != nil {
		return err
	}

	if err := deleteUpdatePod(kubeClient, namespace, o.NodeName); err != nil {
		return err
	}

	log.Infof("Node %s successfully updated", o.NodeName)

	return nil
}

// prepareNode checks if an update is available then cordons and drains the node.
func prepareNode(o *UpdateOptions, kubeClient kubernetes.Interface, kcConfig *kubectl.KubectlConfig) error {
	const key = "ocne.oracle.com/update-available"

	// check if node is ready to be updated, the annotation will be set to true
	node, err := k8s.GetNode(kubeClient, o.NodeName)
	if err != nil {
		return err
	}
	readyToUpdate := false
	if node.Annotations != nil {
		v, ok := node.Annotations[key]
		if ok && strings.ToLower(v) == "true" {
			readyToUpdate = true
		}
	}
	if !readyToUpdate {
		return fmt.Errorf("Node %s has no updates available", o.NodeName)
	}

	// don't drain the node if this is a single node cluster
	// TODO: we should drain if there is >1 worker nodes
	nodelist, err := k8s.GetNodeList(kubeClient)
	if len(nodelist.Items) == 1 {
		return nil
	}

	log.Infof("Draining node %s before updating it", o.NodeName)

	// cordon and drain the node
	if err := cordonAndDrainNode(o, kcConfig); err != nil {
		return err
	}

	return nil
}

// isWorkerLessOrEqualToControlPlane takes in the major.minor of the worker node that we are trying to update and of the control plane node that it is being compared against
// If the worker node has a major.minor that is less than the control plane node, it returns true
func isWorkerLessOrEqualToControlPlane(workerVersion string, controlPlaneVersion string) (bool, error) {
	controlPlaneSemver, err := semver.NewVersion(controlPlaneVersion)
	if err != nil {
		return false, err
	}
	workerSemver, err := semver.NewVersion(workerVersion)
	if err != nil {
		return false, err
	}
	controlPlaneSanitized := fmt.Sprintf("%d.%d", controlPlaneSemver.Major(), controlPlaneSemver.Minor())
	workerSanitized := fmt.Sprintf("%d.%d", workerSemver.Major(), workerSemver.Minor())
	workerSanitizedSemver, err := semver.NewVersion(workerSanitized)
	if err != nil {
		return false, err
	}
	c, err := semver.NewConstraint("<= " + controlPlaneSanitized)
	if err != nil {
		return false, err
	}
	return c.Check(workerSanitizedSemver), nil

}

// areControlPlaneNodesAcceptable returns false if the node to upgrade is a worker node and at least one control plane node is not actively running the desired version to upgrade to
// For example, if a worker node was running 1.27, and the user wanted to upgrade this worker node to 1.28, all of the control plane nodes must be running 1.28
// This command returns true otherwise, along with any potential errors along the way
func areControlPlaneNodesAcceptable(nodeList *corev1.NodeList, nodeNameBeingUpgraded string, restConfig *rest.Config, kubeClient kubernetes.Interface, kubeConfigPath string) (bool, error) {
	var nodeToUpgrade *corev1.Node
	for _, node := range nodeList.Items {
		if node.Name == nodeNameBeingUpgraded {
			nodeToUpgrade = &node
			break
		}
	}

	if nodeToUpgrade == nil {
		return false, errors.New("node " + nodeNameBeingUpgraded + " not found")
	}

	// if the node we want to update is a control plane node, return true
	if k8s.IsControlPlane(nodeToUpgrade) {
		return true, nil
	}

	// It's a worker node.  Get the staged update via the tag in the
	// update configuration file.  Use that to decide if the update
	// that is available is suitable.  If the tag is not a major.minor
	// then assume that the user knows what they are doing and allow
	// the ugprade to proceed
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeConfigPath, constants.OCNESystemNamespace, []string{}, false)
	if err != nil {
		return false, err
	}

	kcConfig.Streams.Out = bytes.NewBuffer([]byte{})
	log.Debugf("Getting update target for %s", nodeNameBeingUpgraded)
	err = script.RunScript(kubeClient, kcConfig, nodeNameBeingUpgraded, constants.OCNESystemNamespace, "check-update", getUpdateInfo, []corev1.EnvVar{})
	if err != nil {
		return false, err
	}
	err = k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, fmt.Sprintf("check-update-%s", nodeNameBeingUpgraded))
	if err != nil {
		return false, err
	}

	// Try really hard to get a reasonable tag, so tolerate some corruption.
	// Rather than parse the document as yaml, treat it as text
	updateInfo := kcConfig.Streams.Out.(*bytes.Buffer).String()
	tag := ""
	for _, line := range strings.Split(updateInfo, "\n") {
		// Split into fields with as much goop removed as possible
		ui := strings.TrimSpace(line)
		fields := strings.Fields(ui)

		// need "tag: val"
		if len(fields) < 2 || fields[0] != "tag:" {
			continue
		}

		// get rid of any quotes
		tag = strings.Trim(fields[1], "\"'")
	}

	if tag == "" {
		return false, fmt.Errorf("The target Kubernetes version is not set")
	}

	// If the tag is not a semantic version, allow the update
	_, err = semver.NewVersion(tag)
	if err != nil {
		return true, nil
	}

	// Iterate through all the nodes,
	// If a control plane node has an update available or a control plane node is running a major.minor.patch version greater than worker node being looked at, return false
	for _, node := range nodeList.Items {
		if node.Name != nodeNameBeingUpgraded {
			if k8s.IsControlPlane(&node) {
				if k8s.IsUpdateAvailable(&node) {
					// In this scenario, a worker is attempting to be upgraded and a control plane node has an update available
					return false, nil
				}
				workerLessOrEqualToControlPlane, err := isWorkerLessOrEqualToControlPlane(tag, node.Status.NodeInfo.KubeletVersion)
				if err != nil {
					return false, err
				}
				if !workerLessOrEqualToControlPlane {
					// In this scenario, a worker is attempting to be upgraded and a control plane node has a major.minor version that is less than the target worker node version.
					return false, nil
				}
			}
		}
	}
	return true, nil
}

// getDesiredVersionFromKubeadmConfig gets the Kubernetes version in the kubeadm-config Config Map, parses it, and returns it to the caller in a semver object
// This version is used to represent the version that the user wants to upgrade the node to, for the purposes of checking
func getVersionsFromKubeadmConfigMap(kubeClient kubernetes.Interface) (*semver.Version, error) {
	kubeadmConfigMap, err := k8s.GetConfigmap(kubeClient, constants.KubeNamespace, constants.KubeCMName)
	if err != nil {
		return nil, err
	}
	data := kubeadmConfigMap.Data["ClusterConfiguration"]

	compile, err := regexp.Compile("kubernetesVersion:.*[$\\n]")
	if err != nil {
		return nil, err
	}
	newData := compile.FindString(data)
	sanitizedNewData := strings.ReplaceAll(newData, "\n", "")
	splitNewData := strings.Split(sanitizedNewData, ": v")
	desiredMajorMinorPatch := splitNewData[1]
	desiredVersion, err := semver.NewVersion(desiredMajorMinorPatch)
	if err != nil {
		return nil, err
	}
	return desiredVersion, nil
}

// deleteUpdatePod ensures that the pod used to run the script to update and reboot the node is deleted
// When this pod is created on a node in a cluster, it doesn't clean itself up
func deleteUpdatePod(client kubernetes.Interface, namespace string, nodeName string) error {
	podName := "update-node-" + nodeName + "-pod"
	return k8s.DeletePod(client, namespace, podName)
}
