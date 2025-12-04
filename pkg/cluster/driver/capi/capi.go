// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"slices"
)

const (
	ClusterNameLabel = "cluster.x-k8s.io/cluster-name"
)

// These should be treated as constants
var ClusterInfrastructureRef []string = []string{"spec", "infrastructureRef"}
var ClusterControlPlaneRef []string = []string{"spec", "controlPlaneRef"}
var ControlPlaneVersion []string = []string{"spec", "version"}
var ControlPlaneMachineTemplateInfrastructureRef []string = []string{"spec", "machineTemplate", "infrastructureRef"}
var ControlPlaneJoinPatches []string = []string{"spec", "kubeadmConfigSpec", "joinConfiguration", "patches"}
var ControlPlaneJoinSkipPhases []string = []string{"spec", "kubeadmConfigSpec", "joinConfiguration", "skipPhases"}
var MachineDeploymentInfrastructureRef []string = []string{"spec", "template", "spec", "infrastructureRef"}
var MachineDeploymentVersion []string = []string{"spec", "template", "spec", "version"}

var SkipKubeProxyAnnotation = "controlplane.cluster.x-k8s.io/skip-kube-proxy"
var SkipCoreDNSAnnotation = "controlplane.cluster.x-k8s.io/skip-coredns"
var ControlPlaneAPI = "controlplane.cluster.x-k8s.io/v1beta1"
var KubeadmControlPlane = "KubeadmControlPlane"

var PatchesDirectory = "directory"
var PhasePreflight = "preflight"

type GraphNode struct {
	Object   *unstructured.Unstructured
	Children map[string]map[string]*GraphNode
}

type ClusterGraph struct {
	Cluster               *GraphNode
	InfrastructureCluster *GraphNode
	ControlPlane          *GraphNode
	MachineTemplates      map[string]map[string]*GraphNode
	MachineDeployments    map[string]*GraphNode
	MachineSets           map[string]*GraphNode
	Machines              map[string]*GraphNode
	All                   map[string]map[string]*GraphNode
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
		Cluster:               newGraphNode(),
		InfrastructureCluster: newGraphNode(),
		ControlPlane:          newGraphNode(),
		MachineTemplates:      map[string]map[string]*GraphNode{},
		MachineDeployments:    map[string]*GraphNode{},
		MachineSets:           map[string]*GraphNode{},
		Machines:              map[string]*GraphNode{},
		All:                   map[string]map[string]*GraphNode{},
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

// GetClusterObject returns the CAPI cluster object
func GetClusterObject(clusterResources string) (unstructured.Unstructured, error) {
	clusterObj, err := k8s.FindIn(clusterResources, func(u unstructured.Unstructured) bool {
		if u.GetKind() != "Cluster" {
			return false
		}
		if u.GetAPIVersion() != "cluster.x-k8s.io/v1beta1" {
			return false
		}
		_, ok := u.GetLabels()[ClusterNameLabel]
		return ok
	})
	if err != nil {
		if k8s.IsNotExist(err) {
			return unstructured.Unstructured{}, fmt.Errorf("Cluster resources do not include a valid cluster.x-k8s.io/v1beta1/Cluster")
		} else {
			return unstructured.Unstructured{}, err
		}
	}
	return clusterObj, err
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

type WalkResourceCb func(*GraphNode, *GraphNode, interface{}) error

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

// PatchControlPlane adds any changes required to a KubeadmControlPlane
func PatchControlPlane(restConfig *rest.Config, kcp *unstructured.Unstructured) error {
	// Ensure this is a KubeadmControlPlane
	if kcp.GetAPIVersion() != ControlPlaneAPI || kcp.GetKind() != KubeadmControlPlane {
		return fmt.Errorf("Control plane object %s in namespace %s is not a %s/%s", kcp.GetName(), kcp.GetNamespace(), ControlPlaneAPI, KubeadmControlPlane)
	}

	didUpdate := false
	annots := kcp.GetAnnotations()
	if annots == nil {
		annots = map[string]string{}
	}
	_, ok := annots[SkipKubeProxyAnnotation]
	if !ok {
		annots[SkipKubeProxyAnnotation] = "true"
		didUpdate = true
	}

	_, ok = annots[SkipCoreDNSAnnotation]
	if !ok {
		annots[SkipCoreDNSAnnotation] = "true"
		didUpdate = true
	}

	if !didUpdate {
		return nil
	}

	kcp.SetAnnotations(annots)
	err := k8s.UpdateResource(restConfig, kcp)
	if err != nil {
		return err
	}

	return nil
}

// GetControlPlanePatches calculates the set of patches that need to be applied to the KubeadmControlPlane after staging is complete to
// apply the new configuration
func GetControlPlanePatches(kcp *unstructured.Unstructured, version string, mtName string) (*util.JsonPatches, error) {
	ret := &util.JsonPatches{}

	// These are mandatory changes to update control plane nodes
	ret.Replace(ControlPlaneVersion, version)
	ret.Replace(append(ControlPlaneMachineTemplateInfrastructureRef, "name"), mtName)

	//  The joinConfiguration needs to apply the OCK patches
	patches, found, err := unstructured.NestedStringMap(kcp.Object, ControlPlaneJoinPatches...)
	if err != nil {
		return nil, err
	}

	if found {
		return ret, nil
	}

	patchDir, ok := patches[PatchesDirectory]
	if ok {
		if patchDir != ignition.OckPatchDirectory {
			ret.Replace(append(ControlPlaneJoinPatches, PatchesDirectory), ignition.OckPatchDirectory)
		}
	} else {
		ret.Add(ControlPlaneJoinPatches, map[string]string{PatchesDirectory: ignition.OckPatchDirectory})
	}

	joinSkips, found, err := unstructured.NestedStringSlice(kcp.Object, ControlPlaneJoinSkipPhases...)
	if err != nil {
		return nil, err
	}
	if !found {
		joinSkips = []string{}
	}
	if !slices.Contains(joinSkips, PhasePreflight) {
		joinSkips = append(joinSkips, PhasePreflight)
		err = unstructured.SetNestedStringSlice(kcp.Object, joinSkips, ControlPlaneJoinSkipPhases...)
		if err != nil {
			return nil, err
		}

		// If the field was already there, replace it.  Otherwise, add it.
		if found {
			ret.Replace(ControlPlaneJoinSkipPhases, joinSkips)
		} else {
			ret.Add(ControlPlaneJoinSkipPhases, joinSkips)
		}
	}

	return ret, nil
}
