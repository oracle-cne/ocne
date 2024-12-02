// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze/triage"
	v1 "k8s.io/api/core/v1"
	"os"
	"regexp"
)

var excludeRegexpr = []string{"Search Line limits were exceeded"}

func analyzeEvents(p *analyzeParams) error {
	// Read the events.json/yaml from each namespace directory into a map
	evMap, err := readNamespacedJSONOrYAMLFiles[v1.EventList](p, "events")
	if err != nil {
		return err
	}

	// Analyze the events for each namespace
	var allSymptoms []*triage.ResourceSymptomInfo[v1.Event]
	for _, evList := range evMap {
		// Filter out normal and exclusion warnings
		symptoms, err := getEventSymptoms(&evList)
		if err != nil {
			return err
		}
		allSymptoms = append(allSymptoms, symptoms...)

		// analyzeEventList(ns.(string), events)
	}
	if p.verbose {
		displayEventSymptoms(os.Stdout, allSymptoms)
	}

	return nil
}

func getEventSymptoms(eventList *v1.EventList) ([]*triage.ResourceSymptomInfo[v1.Event], error) {
	var symptomInfos []*triage.ResourceSymptomInfo[v1.Event]

	// Compile the regex for exclusion strings.
	regExs := []*regexp.Regexp{}
	for i, _ := range excludeRegexpr {
		r, err := regexp.Compile(excludeRegexpr[i])
		if err != nil {
			return nil, fmt.Errorf("Error compiling regex: %v", err)
		}
		regExs = append(regExs, r)
	}

	// Filter out normal and exclusion warnings
	events, err := filterOut(eventList, func(ev *v1.Event) bool {
		// exclude normal
		if ev.Type == v1.EventTypeNormal {
			return true
		}
		// exclude any exclusion match
		for _, r := range regExs {
			if r.FindString(ev.Message) != "" {
				return true
			}
		}
		return false
	})

	for i, _ := range events {
		// Add a new pod symptomList to the list
		symptomInfo := triage.NewEventSymptomInfo(events[i])
		symptomInfos = append(symptomInfos, symptomInfo)
		symptomInfo.AddSymptom(fmt.Sprintf(events[i].Message))

	}

	return symptomInfos, err
}

// filterOut filters out (removes) specific events
func filterOut(events *v1.EventList, fExclude func(*v1.Event) bool) ([]*v1.Event, error) {
	outList := []*v1.Event{}
	for i, ev := range events.Items {
		if !fExclude(&ev) {
			outList = append(outList, &events.Items[i])
		}
	}
	return outList, nil
}

func analyzeEventList(namespace string, events []*v1.Event) {
	for _, ev := range events {
		fmt.Println("------------------------")
		fmt.Printf("Kind: %s  Object: %s/%s:  Reason: %s  Message: %s\n", namespace,
			ev.InvolvedObject.Name, ev.InvolvedObject.Kind, ev.Reason, ev.Message)
	}

}
