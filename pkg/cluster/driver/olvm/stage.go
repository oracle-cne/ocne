// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"os"
	"strings"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/capi"
	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TemplateData - for each vmTemplateName maintain a list of OLVMMachineTemplates that
// reference it.
type TemplateData struct {
	HasUpdate        bool
	NewTemplate      string
	MachineTemplates []*capi.GraphNode
}

// vmTemplateName should be treated as a constant
var vmTemplateName = []string{"spec", "template", "spec", "ovirt", "vmTemplateName"}

// Stage looks at the resources for an OLVM CAPI cluster and generates as
// much of the material necessary to update a cluster from one version to
// another. Some instructions are printed that tell a user how to apply the
// staged update to their cluster.
func (cad *OlvmDriver) Stage(version string) (string, string, bool, error) {
	restConfig, kubeClient, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return "", "", false, err
	}

	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return "", "", false, err
		}
		cad.ClusterResources = cdi
	}

	clusterObj, err := capi.GetClusterObject(cad.ClusterResources)
	if err != nil {
		return "", "", false, err
	}

	// Update OLVMMachineDeployments to use the new VM templates.
	log.Debugf("Getting graph for Cluster %s in namespace %s", clusterObj.GetName(), clusterObj.GetNamespace())
	graph, err := capi.GetClusterGraph(restConfig, clusterObj.GetNamespace(), clusterObj.GetName())
	if err != nil {
		return "", "", false, err
	}

	// Make sure that there is a control plane defined.  Also check to see
	// if the minor version changed.
	currentKubeVersion, found, err := unstructured.NestedString(graph.ControlPlane.Object.Object, capi.ControlPlaneVersion...)
	if err != nil {
		return "", "", false, err
	} else if !found {
		return "", "", false, fmt.Errorf("%s/%s %s in %s does not have a version", graph.ControlPlane.Object.GroupVersionKind().String(), graph.ControlPlane.Object.GroupVersionKind().String(), graph.ControlPlane.Object.GetName(), graph.ControlPlane.Object.GetNamespace())
	}

	// Apply necessary control plane modifications whether there
	// is an update or not.
	err = capi.PatchControlPlane(restConfig, graph.ControlPlane.Object)
	if err != nil {
		return "", "", false, err
	}

	minorVersionCmp, err := versions.CompareKubernetesVersions(currentKubeVersion, version)
	if err != nil {
		return "", "", false, err
	}
	minorVersionChanged := minorVersionCmp != 0
	log.Debugf("kubernetes minor version changes = %t", minorVersionChanged)

	// Get the collection of vmTemplateNames in use
	vmTemplates, err := cad.graphToVMTemplates(graph)
	if err != nil {
		return "", "", false, err
	}

	// Make the new machine templates by creating a new OLVMMachineTemplate
	// for each existing one that uses an existing OLVM Template.
	updatedMts := map[*capi.GraphNode]*unstructured.Unstructured{}

	for _, img := range vmTemplates {
		log.Debugf("Creating template for %s", img.NewTemplate)
		newTemplate := img.NewTemplate

		// Template updates can be forced for testing purposes.
		// This is useful because the templates are generated only
		// if there is a reasonable update to perform.
		// This calculation is made by looking at resources, timestamps, and other
		// durable data challenging to set up within the
		// context of a test harness.
		if os.Getenv("OCNE_OLVM_STAGE_FORCE_TEMPLATES") == "" && !img.HasUpdate {
			continue
		}

		for _, mtNode := range img.MachineTemplates {
			mt := mtNode.Object.DeepCopy()
			name := util.IncrementCount(mt.GetName(), "-")
			mt.SetName(name)

			err = unstructured.SetNestedField(mt.Object, newTemplate, "spec", "template", "spec", "ovirt", "vmTemplateName")
			if err != nil {
				return "", "", false, err
			}

			err = k8s.CreateResource(restConfig, mt)
			if err != nil {
				return "", "", false, err
			}

			updatedMts[mtNode] = mt
		}
	}

	// Display some state information and instructions.  The new machine
	// templates that were generated need to get propagated into the
	// MachineDeployments and KubeadmControlPlanes in the cluster.
	var helpMessages []string
	err = graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{}) error {
		updatedMts := arg.(map[*capi.GraphNode]*unstructured.Unstructured)

		var umt *unstructured.Unstructured
		umt, ok := updatedMts[mtNode]
		if !ok {
			return nil
		}

		if parent == graph.ControlPlane {
			kubeVersions, err := versions.GetKubernetesVersions(version)
			if err != nil {
				return err
			}

			patches, err := capi.GetControlPlanePatches(parent.Object, kubeVersions.Kubernetes, umt.GetName())
			if err != nil {
				return err
			}

			helpMessages = append(helpMessages, fmt.Sprintf("To update KubeadmControlPlane %s in %s, run:\n    kubectl patch -n %s kubeadmcontrolplane %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches))
		} else {
			kubeVersions, err := versions.GetKubernetesVersions(version)
			if err != nil {
				return err
			}

			patches := (&util.JsonPatches{}).Replace(capi.MachineDeploymentVersion, kubeVersions.Kubernetes).Replace(append(capi.MachineDeploymentInfrastructureRef, "name"), umt.GetName()).String()
			helpMessages = append(helpMessages, fmt.Sprintf("To update MachineDeployment %s in %s, run:\n    kubectl patch -n %s machinedeployment %s --type=json -p='%s'\n", parent.Object.GetName(), parent.Object.GetNamespace(), parent.Object.GetNamespace(), parent.Object.GetName(), patches))
		}

		return nil
	}, updatedMts)
	if err != nil {
		return "", "", false, err
	}

	// Hand back the kubeconfig for the managed cluster.
	clusterName, _ := clusterObj.GetLabels()[ClusterNameLabel]
	kcfg, err := cad.waitForKubeconfig(kubeClient, clusterName)
	kcfgPath, err := util.InMemoryFile(fmt.Sprintf("kcfg.%s", clusterName))

	f, err := os.OpenFile(kcfgPath, os.O_RDWR, 0)
	if err != nil {
		return "", "", false, err
	}
	_, err = f.Write([]byte(kcfg))
	f.Close()
	if err != nil {
		return "", "", false, err
	}
	return kcfgPath, strings.Join(helpMessages, "\n"), true, nil
}

// templateNameFromMachineTemplate gets a vmTemplateName from an OLVMMachineTemplate
func templateNameFromMachineTemplate(mt *unstructured.Unstructured) (string, error) {
	templateName, found, err := unstructured.NestedString(mt.Object, vmTemplateName...)
	if !found {
		err = fmt.Errorf("OLVMMachineTemplate %s in %s has no vmTemplateName", mt.GetName(), mt.GetNamespace())
	}
	if err != nil {
		return "", err
	}
	log.Debugf("OLVMMachineTemplate %s in %s has vmTemplateName %s", mt.GetName(), mt.GetNamespace(), templateName)

	return templateName, nil
}

// graphToVMTemplates scrapes the graph of CAPI resources and extracts the
// template names from the OLVMMachineTemplates.
func (cad *OlvmDriver) graphToVMTemplates(graph *capi.ClusterGraph) (map[string]*TemplateData, error) {
	ret := map[string]*TemplateData{}

	err := graph.WalkMachineTemplates(func(parent *capi.GraphNode, mtNode *capi.GraphNode, arg interface{}) error {
		mt := mtNode.Object
		retVal := arg.(map[string]*TemplateData)
		template, err := templateNameFromMachineTemplate(mt)
		if err != nil {
			return err
		}

		// Determine if the vmTemplateName has changed
		update, newTemplate, cpNode := hasUpdate(mt, template, cad.ClusterConfig.Providers.Olvm)

		// Create separate map entries for control-plane and machine nodes,
		// they can be configured to use different template names.
		key := fmt.Sprintf("%s-%t", template, cpNode)
		imgData, ok := retVal[key]
		if !ok {
			retVal[key] = &TemplateData{
				HasUpdate:        update,
				NewTemplate:      newTemplate,
				MachineTemplates: []*capi.GraphNode{mtNode},
			}
		} else {
			imgData.MachineTemplates = append(imgData.MachineTemplates, mtNode)
		}
		return nil
	}, ret)
	return ret, err
}

func hasUpdate(node *unstructured.Unstructured, templateName string, provider types.OlvmProvider) (bool, string, bool) {
	newTemplateName := ""
	controlPlaneNode := false

	// Determine the new template name based on if the current node is
	// for a control-plane or worker node.
	if strings.Contains(node.GetName(), "control-plane") {
		controlPlaneNode = true
		if len(provider.ControlPlaneMachine.VMTemplateName) > 0 {
			newTemplateName = provider.ControlPlaneMachine.VMTemplateName
		}
	} else if len(provider.WorkerMachine.VMTemplateName) > 0 {
		newTemplateName = provider.WorkerMachine.VMTemplateName
	}

	if len(newTemplateName) > 0 {
		return newTemplateName != templateName, newTemplateName, controlPlaneNode
	}
	return false, "", controlPlaneNode
}
