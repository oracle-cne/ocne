// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze/triage"
	v1 "k8s.io/api/core/v1"
)

func analyzeNodes(p *analyzeParams) error {

	return nil
}

func analyzeClusterNodes(p *analyzeParams) error {
	nodeList, err := readClusterWideJSONOrYAMLFile[v1.NodeList](p, "nodes")
	if err != nil {
		return err
	}
	if nodeList == nil {
		return nil
	}
	symptomInfos, err := findNodeSymptoms(nodeList)
	if err != nil {
		return err
	}

	return displayNodeSymptoms(p.writer, symptomInfos)
}

func findNodeSymptoms(nodeList *v1.NodeList) ([]*triage.ResourceSymptomInfo[v1.Node], error) {
	var symptomInfos []*triage.ResourceSymptomInfo[v1.Node]
	for i, node := range nodeList.Items {
		symptomInfo := triage.NewNodeSymptomInfo(&nodeList.Items[i])
		symptomInfos = append(symptomInfos, symptomInfo)

		if node.Spec.Unschedulable {
			symptomInfo.AddSymptom(fmt.Sprintf("Pods cannot be scheduled on node %s", node.Name))
		}
		for _, cond := range node.Status.Conditions {
			switch cond.Type {
			case v1.NodeReady:
				if cond.Status == "False" {
					symptomInfo.AddSymptom(fmt.Sprintf("Node %s is not ready: %s", node.Name, cond.Message))
				}
			case v1.NodeMemoryPressure:
			case v1.NodeDiskPressure:
			case v1.NodePIDPressure:
			case v1.NodeNetworkUnavailable:
			default:
				if cond.Status == "True" {
					symptomInfo.AddSymptom(fmt.Sprintf("Node %s has a problem: %s", node.Name, cond.Message))
				}
			}
		}
	}
	return symptomInfos, nil
}
