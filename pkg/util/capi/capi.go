// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"github.com/oracle-cne/ocne/pkg/k8s"

)

// These should be treated as constants
var ClusterInfrastructureRef []string = []string{"spec", "infrastructureRef"}
var ClusterControlPlaneRef []string = []string{"spec", "controlPlaneRef"}
var ControlPlaneMachineTemplateInfrastructureRef []string = []string{"spec", "machineTemplate", "infrastructureRef"}
var MachineDeploymentInfrastructureRef []string = []string{"spec", "template", "spec", "infrastructureRef"}

type GraphNode struct {
	Object *unstructured.Unstructured
	Children map[string]map[string]*GraphNode
}

type ClusterGraph struct {
	Cluster *GraphNode
	InfrastructureCluster *GraphNode
	ControlPlane *GraphNode
	MachineTemplates map[string]map[string]*GraphNode
	MachineDeployments map[string]*GraphNode
	MachineSets map[string]*GraphNode
	Machines map[string]*GraphNode
	All map[string]map[string]*GraphNode
}

type Named interface {
	GetName() string
}

func newGraphNode() *GraphNode {
	return &GraphNode{
		Children: map[string]map[string]*GraphNode{},
	}
}

func getFromNestedMap[V any](m map[string]map[string]*V, gvk string, name string) *V {
	kindMap, ok := m[gvk]
	if !ok {
		return nil
	}

	return kindMap[name]
}

func addToNestedMap[V Named](m map[string]map[string]V, firstKey string, v V) {
	kindMap, ok := m[firstKey]
	if !ok {
		m[firstKey] = map[string]V{}
		kindMap = m[firstKey]
	}

	kindMap[v.GetName()] = v
}

func (gn *GraphNode) AddChild(c *GraphNode) {
	addToNestedMap(gn.Children, c.Object.GroupVersionKind().String(), c)
}

func (gn *GraphNode) GetName() string {
	return gn.Object.GetName()
}

func newClusterGraph() *ClusterGraph {
	return &ClusterGraph{
		Cluster: newGraphNode(),
		InfrastructureCluster: newGraphNode(),
		ControlPlane: newGraphNode(),
		MachineTemplates: map[string]map[string]*GraphNode{},
		MachineDeployments: map[string]*GraphNode{},
		MachineSets: map[string]*GraphNode{},
		Machines: map[string]*GraphNode{},
		All: map[string]map[string]*GraphNode{},
	}
}

func makeUnst(apiVersion string, kind string, namespace string, name string) *GraphNode {
	ret := newGraphNode()
	ret.Object = &unstructured.Unstructured{}
	ret.Object.SetNamespace(namespace)
	ret.Object.SetName(name)
	ret.Object.SetGroupVersionKind(schema.FromAPIVersionAndKind(apiVersion, kind))
	return ret
}

func makeUnstFromRef(ref map[string]string, namespace string) (*GraphNode, error) {
	apiVersion, ok := ref["apiVersion"]
	if !ok {
		return nil, fmt.Errorf("Reference does not contain an apiVersion")
	}
	kind, ok := ref["kind"]
	if !ok {
		return nil, fmt.Errorf("Reference does not contain a kind")
	}
	name, ok := ref["name"]
	if !ok {
		return nil, fmt.Errorf("Reference does not contain a name")
	}

	ret := makeUnst(apiVersion, kind, namespace, name)
	return ret, nil
}

func stringMap(u *unstructured.Unstructured, path ...string) (map[string]string, error) {
	ret, _, err := unstructured.NestedStringMap(u.Object, path...)
	return ret, err
}

func getByRef(restConf *rest.Config, u *unstructured.Unstructured, path ...string) (*GraphNode, error) {
	ref, err := stringMap(u, path...)
	if err != nil {
		return nil, err
	}

	ret, err := makeUnstFromRef(ref, u.GetNamespace())
	if err != nil {
		return nil, err
	}

	err = k8s.GetResource(restConf, ret.Object)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func getByOwner(restConf *rest.Config, owner *unstructured.Unstructured, apiVersion string, kind string) ([]*GraphNode, error) {
	ul, err := k8s.GetResources(restConf, owner.GetNamespace(), apiVersion, kind)
	if err != nil {
		return nil, err
	}

	ret := []*GraphNode{}
	for _, u := range ul.Items {
		if u.GetNamespace() != owner.GetNamespace() {
			continue
		}

		for _, o := range u.GetOwnerReferences() {
			if o.APIVersion == owner.GetAPIVersion() && o.Kind == owner.GetKind() && o.Name == owner.GetName() {
				gn := newGraphNode()
				gn.Object = &u
				ret = append(ret, gn)
				break
			}
		}
	}

	return ret, nil
}

func populateControlPlane(restConf *rest.Config, graph *ClusterGraph, controlPlane *GraphNode) error {
	if controlPlane.Object.GetKind() != "KubeadmControlPlane" {
		return fmt.Errorf("Only KubeadmControlPlanes are supported")
	}

	machineTemplate, err := getByRef(restConf, controlPlane.Object, ControlPlaneMachineTemplateInfrastructureRef...)
	if err != nil {
		return err
	}

	machineTemplate = graph.AddToAll(machineTemplate)
	controlPlane.AddChild(machineTemplate)
	addToNestedMap(graph.MachineTemplates, machineTemplate.Object.GroupVersionKind().String(), machineTemplate)

	return nil
}

func populateMachineDeployments(restConf *rest.Config, graph *ClusterGraph, cluster *GraphNode) error {
	mds, err := getByOwner(restConf, cluster.Object, "cluster.x-k8s.io/v1beta1", "MachineDeployment")
	if err != nil {
		return err
	}

	for _, md := range mds {
		md = graph.AddToAll(md)
		cluster.AddChild(md)
		graph.MachineDeployments[md.Object.GetName()] = md

		machineTemplate, err := getByRef(restConf, md.Object, MachineDeploymentInfrastructureRef...)
		if err != nil {
			return err
		}

		machineTemplate = graph.AddToAll(machineTemplate)
		md.AddChild(machineTemplate)
		addToNestedMap(graph.MachineTemplates, machineTemplate.Object.GroupVersionKind().String(), machineTemplate)
	}

	return nil
}

func GetClusterGraph(restConf *rest.Config, namespace string, name string) (*ClusterGraph, error) {
	ret := newClusterGraph()

	// Get the Cluster
	cluster := makeUnst("cluster.x-k8s.io/v1beta1", "Cluster", namespace, name)
	err := k8s.GetResource(restConf, cluster.Object)
	if err != nil {
		return nil, err
	}

	ret.Cluster = cluster
	ret.AddToAll(cluster)

	// The cluster has two references, an infrastructureRef and a controlPlaneRef
	infraCluster, err := getByRef(restConf, cluster.Object, ClusterInfrastructureRef...)
	if err != nil {
		return nil, err
	}

	cluster.AddChild(infraCluster)
	ret.InfrastructureCluster = ret.AddToAll(infraCluster)

	controlPlane, err := getByRef(restConf, cluster.Object, ClusterControlPlaneRef...)
	if err != nil {
		return nil, err
	}

	cluster.AddChild(controlPlane)
	ret.ControlPlane = ret.AddToAll(controlPlane)

	err = populateControlPlane(restConf, ret, controlPlane)
	if err != nil {
		return nil, err
	}

	err = populateMachineDeployments(restConf, ret, cluster)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (cg *ClusterGraph) AddToAll(gn *GraphNode) *GraphNode {
	egn := getFromNestedMap(cg.All, gn.Object.GroupVersionKind().String(), gn.GetName())
	if egn != nil {
		return egn
	}
	addToNestedMap(cg.All, gn.Object.GroupVersionKind().String(), gn)
	return gn
}

type WalkResourceCb func(*GraphNode, *GraphNode, interface{})error

func (cg *ClusterGraph) walkGraphNodeForMachineTemplates(gn *GraphNode, cb WalkResourceCb, arg interface{}) error {
	for gvk, children := range gn.Children {
		// This kind does not represent a machine template
		mts, ok := cg.MachineTemplates[gvk]
		if !ok {
			return nil
		}

		for n, _ := range children {
			// This child is not in the MachineTemplates, which is odd.
			mtNode, ok := mts[n]
			if !ok {
				continue
			}

			err := cb(gn, mtNode, arg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cg *ClusterGraph) WalkMachineTemplates(cb WalkResourceCb, arg interface{}) error {
	err := cg.walkGraphNodeForMachineTemplates(cg.ControlPlane, cb, arg)
	if err != nil {
		return err
	}
	for _, md := range cg.MachineDeployments {
		err = cg.walkGraphNodeForMachineTemplates(md, cb, arg)
		if err != nil {
			return err
		}
	}
	return nil
}
