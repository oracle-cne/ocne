// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/constants"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util"
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
	if err == nil && len(nodeList.Items) > 0 {
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

// WaitForControlPlaneNodes gets all ndoes that run control plane components.
// Unlike GetControlPlaneNodes, it waits for a bit if it appears that no
// control plane nodes are available.  This is useful early in cluster
// creation when even static pods have not been created.
func WaitForControlPlaneNodes(cli kubernetes.Interface) (*v1.NodeList, error) {
	list, _, err := util.LinearRetry(func(arg interface{})(interface{}, bool, error) {
		nodeList, err := GetControlPlaneNodes(cli)
		if err != nil {
			return nil, true, err
		}

		if len(nodeList.Items) == 0 {
			return nil, false, fmt.Errorf("no control plane nodes")
		}
		return nodeList, false, nil
	}, cli)
	if err != nil {
		return nil, err
	}
	ret, ok := list.(*v1.NodeList)
	if !ok {
		// This shouldn't happen.
		return nil, fmt.Errorf("internal error: functor did not return a *v1.NodeList")
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

func processImage(registry string, best string, secondBest string, img *v1.ContainerImage) (string, bool, bool) {
	haveBest := false
	exactMatch := false
	ret := ""

	imgName := fmt.Sprintf("%s:%s", registry, secondBest)
	imgPrefix := fmt.Sprintf("%s:", registry)
	bestName := fmt.Sprintf("%s:%s", registry, best)
	for _, name := range img.Names {
		if name == bestName {
			haveBest = true
			continue
		}
		if name == imgName {
			exactMatch = true
			ret = name
			continue
		}

		// If there is already an exact match, don't
		// look at this image
		if exactMatch {
			continue
		}
		if strings.HasPrefix(name, imgPrefix) {
			ret = name
		}
	}

	return ret, haveBest, exactMatch
}


// GetImageCandidate returns a reasonable image:tag based on some criteria
// - If the best match is found, the first return value is set to that image
//   and the second return value is true
// - If the second best match is found, the first return value is set to that
//   image:tag and the third value is true
// - If neither are found, an arbitrary image:tag is returned
// - If the image does not exist on the node, the first value is the empty string
//   and the other two are false.
func GetImageCandidate(registry string, best string, secondBest string, node *v1.Node) (string, bool, bool) {
	imgs := node.Status.Images
	log.Debugf("%s has these images: %+v", node.Name, imgs)

	bestImg := ""
	ret := ""
	foundExact := false
	foundBest := false
	for _, img := range imgs {
		log.Debugf("Checking %+v", img)
		imgName, haveBest, exactMatch := processImage(registry, best, secondBest, &img)
		if imgName == "" {
			continue
		}

		// If there is a current image, don't bother looking anymore
		if haveBest {
			log.Debugf("Have current image for %s", imgName)
			bestImg = fmt.Sprintf("%s:%s", registry, best)
			foundBest = true
			return fmt.Sprintf("%s:%s", registry, best), true, exactMatch
		}

		if exactMatch {
			ret = imgName
			foundExact = true
		} else if !foundExact {
			ret = imgName
		} else if ret == "" {
			ret = imgName
		}
	}
	if foundBest {
		ret = bestImg
	}
	return ret, false, foundExact
}
