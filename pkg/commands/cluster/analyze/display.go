// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"io"
	v1 "k8s.io/api/core/v1"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/analyze/triage"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"sort"
	"text/tabwriter"
	"time"
)

func displayEventSymptoms(writer io.Writer, infos []*triage.ResourceSymptomInfo[v1.Event]) {
	var exists bool

	// Check if any symptoms exist
	for _, info := range infos {
		exists = len(info.Symptoms) > 0 || exists
	}
	if !exists {
		return
	}

	w := new(tabwriter.Writer).Init(writer, 30, 8, 1, '\t', 0)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Events:\n")
	fmt.Fprintf(w, "-------\n")

	// Sort oldest first
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Resource.LastTimestamp.Time.Before(infos[j].Resource.LastTimestamp.Time)
	})

	// Write the events
	for _, info := range infos {
		r := info.Resource
		fmt.Fprintf(w, "%s    %s %s\n",
			r.LastTimestamp.Format(time.RFC3339),
			r.InvolvedObject.Kind,
			r.Reason)
		fmt.Fprintf(w, "Object: %s/%s\n", r.Namespace, r.InvolvedObject.Name)

		// usually there i only one message per event, but our code might augment this
		// with additional messages in the future
		for _, s := range info.Symptoms {
			const maxMsg = 200
			fmt.Fprintf(w, "Message: %s\n", s.Message)
		}
		fmt.Fprintln(w)

	}
	fmt.Fprintln(w)

	w.Flush()
}

func displayNodeSymptoms(writer io.Writer, infos []*triage.ResourceSymptomInfo[v1.Node]) error {
	var exists bool

	for _, info := range infos {
		exists = len(info.Symptoms) > 0 || exists
	}
	w := new(tabwriter.Writer).Init(writer, 30, 8, 1, '\t', 0)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Cluster Nodes:\n")
	fmt.Fprintf(w, "--------------\n")

	if !exists {
		fmt.Fprintf(writer, "Cluster nodes are normal\n")
		return nil
	}

	// Sort control planes first
	sort.Slice(infos, func(i, j int) bool {
		return k8s.IsControlPlane(infos[i].Resource)
	})

	// Write the node summary table
	for _, info := range infos {
		for _, s := range info.Symptoms {
			const maxMsg = 200
			fmt.Fprintf(w, "%s\n", s.Message)
		}
	}
	fmt.Fprintln(w)

	w.Flush()
	return nil
}

func displayPodSymptoms(writer io.Writer, infos []*triage.ResourceSymptomInfo[v1.Pod]) error {
	var exists bool

	// Check if any symptoms exist
	for _, info := range infos {
		exists = len(info.Symptoms) > 0 || exists
	}
	w := new(tabwriter.Writer).Init(writer, 30, 8, 1, '\t', 0)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Pods:\n")
	fmt.Fprintf(w, "-----\n")

	if !exists {
		fmt.Fprintf(writer, "Pods are normal\n")
		return nil
	}

	// Write the summary table
	for _, info := range infos {
		if len(info.Symptoms) == 0 {
			continue
		}
		fmt.Fprintf(w, "Pod: %s/%s on Node %s\n", info.Resource.Namespace, info.Resource.Name, info.Resource.Spec.NodeName)
		for _, s := range info.Symptoms {
			const maxMsg = 200
			fmt.Fprintf(w, "%s\n", s.Message)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	w.Flush()
	return nil
}

func displayImageProblems(writer io.Writer, data *PodmanImageData) error {
	var problem bool
	for _, msg := range data.ImageDetailErrorMap {
		problem = len(msg) > 0 || problem
	}
	w := new(tabwriter.Writer).Init(writer, 30, 8, 1, '\t', 0)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Images:\n")
	fmt.Fprintf(w, "-----\n")

	if !problem {
		fmt.Fprintf(writer, "Images are normal\n")
		return nil
	}

	// Write the node summary table
	for nodeName, msg := range data.ImageDetailErrorMap {
		if len(msg) == 0 {
			continue
		}
		fmt.Fprintf(w, "Image problem on Node %s: %s\n", nodeName, msg)
	}
	fmt.Fprintln(w)

	w.Flush()
	return nil
}
