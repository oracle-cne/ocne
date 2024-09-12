// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze/triage"
	"github.com/oracle-cne/ocne/pkg/constants"
)

func analyzePods(p *analyzeParams) error {
	// Read the pods.json from each namespace directory into a map
	podMap, err := readNamespacedJSONFiles[v1.PodList](p, "pods.json")
	if err != nil {
		return err
	}
	// Analyze the pods for each namespace
	var allSymptomInfos []*triage.ResourceSymptomInfo[v1.Pod]
	for _, podList := range podMap {
		// Find pods with problems
		symptomInfos, err := findPodSymptoms(&podList)
		if err != nil {
			return err
		}
		allSymptomInfos = append(allSymptomInfos, symptomInfos...)
	}

	return displayPodSymptoms(os.Stdout, allSymptomInfos)
}

// Return a list of symptomLists, one for each pod,` for the pods in a single namespace.
func findPodSymptoms(podList *v1.PodList) ([]*triage.ResourceSymptomInfo[v1.Pod], error) {
	var symptomInfos []*triage.ResourceSymptomInfo[v1.Pod]
	for i, pod := range podList.Items {
		if pod.Status.Phase == v1.PodSucceeded {
			continue
		}

		// Add a new pod symptomList to the list
		symptomInfo := triage.NewPodSymptomInfo(&podList.Items[i])
		symptomInfos = append(symptomInfos, symptomInfo)

		// Add a message for every problem
		if pod.Status.Phase == v1.PodFailed {
			symptomInfo.AddSymptom(fmt.Sprintf("  pod has failed"))
		}

		// Skip cluster dump pods, they may have not been ready when the pods.json dump was taken
		if pod.Namespace == constants.OCNESystemNamespace && strings.HasPrefix(pod.Name, "ocne-dump") {
			continue
		}

		// conditions
		for _, cond := range pod.Status.Conditions {
			switch cond.Type {
			case v1.PodReady:
				if cond.Status == "False" {
					symptomInfo.AddSymptom(fmt.Sprintf("  pod is not ready: %s", cond.Message))
				}
			}
		}
		// init containers
		for _, status := range pod.Status.InitContainerStatuses {
			if !status.Ready {
				msg := strings.Builder{}
				if status.State.Waiting != nil {
					if status.State.Waiting.Reason != "" {
						msg.WriteString(status.State.Waiting.Reason + " ")
					}
					if status.State.Waiting.Message != "" {
						msg.WriteString(status.State.Waiting.Message + " ")
					}
				}
				if status.LastTerminationState.Terminated != nil {
					msg.WriteString(status.LastTerminationState.Terminated.Message)
				}
				symptomInfo.AddSymptom(fmt.Sprintf("  init container %s is not ready: %s",
					status.Name,
					msg.String()))
			}
		}
		// containers
		for _, status := range pod.Status.ContainerStatuses {
			if !status.Ready {
				msg := strings.Builder{}
				if status.State.Waiting != nil {
					if status.State.Waiting.Reason != "" {
						msg.WriteString(status.State.Waiting.Reason + " ")
					}
					if status.State.Waiting.Message != "" {
						msg.WriteString(status.State.Waiting.Message + " ")
					}
				}
				if status.LastTerminationState.Terminated != nil {
					msg.WriteString(status.LastTerminationState.Terminated.Message)
				}
				symptomInfo.AddSymptom(fmt.Sprintf("  container %s is not ready: %s",
					status.Name,
					msg.String()))
			}
		}
	}

	return symptomInfos, nil
}
