// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/constants"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

const (
	controlPlaneRole = "control plane"
	workerRole       = "worker"
)

// GetNode returns the specified node.
func GetNode(client kubernetes.Interface, name string) (*v1.Node, error) {
	node, err := client.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})

	return node, err
}

// GetNodeList returns a list of Kubernetes nodes.
func GetNodeList(client kubernetes.Interface) (*v1.NodeList, error) {
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	return nodeList, err
}

// WaitUntilGetNodesSucceeds until Kubernetes returns node list
func WaitUntilGetNodesSucceeds(client kubernetes.Interface) (*v1.NodeList, error) {
	var nodeList *v1.NodeList

	// Check once before  waiting to avoid log spew
	nodeList, err := GetNodeList(client)
	if err == nil {
		return nodeList, nil
	}

	// Wait until get nodes works.
	waitors := []*logutils.Waiter{
		&logutils.Waiter{
			Message: "Waiting for the Kubernetes cluster to be ready",
			WaitFunction: func(ignored interface{}) error {
				// wait for 10 min
				maxTime := time.Now().Add(10 * time.Minute)
				for {
					var err error
					nodeList, err = GetNodeList(client)
					if err == nil && len(nodeList.Items) > 0 {
						return nil
					}
					if time.Now().After(maxTime) {
						return fmt.Errorf("Timeout waiting for get nodes to succeed")
					}
					log.Debugf("Error getting node list: %+v", err)
					time.Sleep(5 * time.Second)
				}
			},
		},
	}
	haveError := logutils.WaitFor(logutils.Info, waitors)
	if haveError == true {
		return nil, fmt.Errorf("Failed to access Kubernetes cluster: %v", waitors[0].Error)
	}
	return nodeList, nil
}

// WaitUntilNodeIsReady until Kubernetes node is ready
func WaitUntilNodeIsReady(client kubernetes.Interface, nodeName string) error {
	waitors := []*logutils.Waiter{
		&logutils.Waiter{
			Message: fmt.Sprintf("Waiting for the node %s to be ready", nodeName),
			WaitFunction: func(ignored interface{}) error {
				// wait for 10 min
				maxTime := time.Now().Add(10 * time.Minute)
				for {
					var err error
					node, err := GetNode(client, nodeName)
					if err != nil {
						return err
					}
					if IsNodeReady(node.Status) {
						return nil
					}

					if time.Now().After(maxTime) {
						return fmt.Errorf("Timeout waiting for node %s to be ready", nodeName)
					}
					time.Sleep(5 * time.Second)
				}
			},
		},
	}
	haveError := logutils.WaitFor(logutils.Info, waitors)
	if haveError == true {
		return fmt.Errorf("Error waiting for node %s to be ready: %v", nodeName, waitors[0].Error)
	}
	return nil
}

func IsNodeReady(status v1.NodeStatus) bool {
	for _, cond := range status.Conditions {
		if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

// GetNodeNames returns the cluster node naems
func GetNodeNames(cli kubernetes.Interface) ([]string, error) {
	nodeNames := []string{}
	nodeList, err := GetNodeList(cli)
	if err != nil {
		return nil, err
	}

	for i, _ := range nodeList.Items {
		nodeNames = append(nodeNames, nodeList.Items[i].Name)
	}
	return nodeNames, nil
}

// GetControlPlaneNodes gets all nodes that run control plane components.
// It looks for data other than labels and annotations to make a reasonable
// guess at what those nodes are.
func GetControlPlaneNodes(cli kubernetes.Interface) (*v1.NodeList, error) {
	// Find the control plane pods
	controlPlanePods, err := GetPodsBySelector(cli, "kube-system", "tier=control-plane")
	if err != nil {
		return nil, err
	}

	// Make a set of all unique node names
	nodes := map[string]bool{}
	for _, p := range controlPlanePods.Items {
		nodes[p.Spec.NodeName] = true
	}

	// Get the nodes and cobble together a list of them
	ret := &v1.NodeList{}
	for n, _ := range nodes {
		node, err := GetNode(cli, n)
		if err != nil {
			return nil, err
		}
		ret.Items = append(ret.Items, *node)
	}

	return ret, nil
}

// GetRole returns true if the node is a control-plane node, along with a string indicating the node's role
func GetRole(node *v1.Node) (bool, string) {
	cp := false
	if node.ObjectMeta.Labels != nil {
		if _, ok := node.ObjectMeta.Labels[constants.K8sLabelControlPlane]; ok {
			cp = true
		}
	}
	if cp {
		return true, controlPlaneRole
	}
	return false, workerRole
}

// IsControlPlane returns true if the node is a control-plane node
func IsControlPlane(node *v1.Node) bool {
	cp, _ := GetRole(node)
	return cp
}

// IsUpdateAvailable returns true if an update is available on that node
func IsUpdateAvailable(node *v1.Node) bool {
	update := false
	if node.ObjectMeta.Annotations != nil && node.ObjectMeta.Annotations[constants.OCNEAnnoUpdateAvailable] == "true" {
		update = true
	}
	return update
}
