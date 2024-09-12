// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/oracle-cne/ocne/pkg/k8s"
)

func displayAllInfo(ci *clusterInfo, writer io.Writer) {
	fmt.Fprintf(writer, "Cluster Summary:\n")
	fmt.Fprintf(writer, "  control plane nodes: %d\n", ci.numControlPlaneNodes)
	fmt.Fprintf(writer, "  worker nodes: %d\n", ci.numWorkerNodes)
	fmt.Fprintf(writer, "  nodes with available updates: %d\n", ci.numNodesWithUpdatesAvailable)

	displayNodes(ci.nodeInfos, writer)

}

func displayNodes(infos []*nodeInfo, writer io.Writer) {
	w := new(tabwriter.Writer).Init(writer, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "\nNodes:\n")
	fmt.Fprintf(w, "  Name\tRole\tState\tVersion\tUpdate Available\n")
	fmt.Fprintf(w, "  ----\t----\t-----\t-------\t----------------\n")

	// Write the node summary table
	for _, n := range infos {
		status := "NotReady"
		if k8s.IsNodeReady(n.node.Status) {
			status = "Ready"
		}
		if n.node.Spec.Unschedulable {
			status = status + ",SchedulingDisabled "
		}

		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%v\n", n.node.Name, n.role, status, n.node.Status.NodeInfo.KubeletVersion, n.updateAvailable)
	}
	fmt.Fprintln(w)

	// Write the node details
	for _, n := range infos {
		displayNodeDetails(n, w)
	}
	w.Flush()
}

func displayNodeDetails(ni *nodeInfo, w io.Writer) {
	if ni.nodeDump == nil {
		return
	}

	fmt.Fprintf(w, "\nNode: %s\n", ni.node.Name)
	fmt.Fprintln(w, "  Registry and tag for ostree patch images:")
	displayLines(w, ni.nodeDump.updateYAML, "    ")
	fmt.Fprintln(w, "  Ostree deployments:")
	displayLines(w, ni.nodeDump.ostreeRefs, "    ")
	fmt.Fprintln(w)
}

func loadSummaryInfo(ci *clusterInfo) {
	for _, n := range ci.nodeInfos {
		if n.controlPlane {
			ci.numControlPlaneNodes += 1
		} else {
			ci.numWorkerNodes += 1
		}
		if n.updateAvailable {
			ci.numNodesWithUpdatesAvailable += 1
		}
	}
	return
}

func displayLines(w io.Writer, s string, prefix string) error {
	scanner := bufio.NewScanner(bytes.NewBufferString(s))
	for scanner.Scan() {
		_, err := w.Write([]byte(prefix + scanner.Text() + "\n"))
		if err != nil {
			return err
		}
	}
	return nil
}
