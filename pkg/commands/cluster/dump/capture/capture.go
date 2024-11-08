// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capture

import (
	"bufio"
	"io"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"path/filepath"
	"sync"
)

const (
	createFileError = "Failed to create file %s: %v"
	namespacesDir   = "namespaces"
	podLogDir       = "podlogs"
	clusterWideDir  = "cluster-wide"

	// Throttle the go routines so Kuberenetes API server doesn't get overloaded
	maxGoRoutines = 50
)

var containerStartLog = "==== START logs for container %s of pod %s/%s ====\n"
var containerEndLog = "==== END logs for container %s of pod %s/%s ====\n"

type CaptureParams struct {
	KubeClient        kubernetes.Interface
	DynamicClient     dynamic.Interface
	Namespaces        []string
	RootDumpDir       string
	ClusterDumpDir    string
	IncludeConfigMaps bool
	SkipPodLogs       bool
	Redact            bool
	Managed           bool
	ToJSON            bool
}

type PodLogs struct {
	IsPodLog bool
	Duration int64
}

type captureSync struct {
	wg      sync.WaitGroup
	channel chan int
}

// CaptureCuratedResources captures a set of curated or well-known resources
func CaptureCuratedResources(cp CaptureParams) error {
	cs := captureSync{
		wg:      sync.WaitGroup{},
		channel: make(chan int, maxGoRoutines),
	}

	for _, ns := range cp.Namespaces {
		// background goroutine will log any errors, but they are not fatal
		captureByNamespace(&cs, cp.DynamicClient, filepath.Join(cp.ClusterDumpDir, namespacesDir, ns), ns, cp.Redact, cp.Managed, cp.ToJSON)
	}
	captureClusterWide(&cs, cp.DynamicClient, filepath.Join(cp.ClusterDumpDir, clusterWideDir), cp.Redact, cp.Managed, cp.ToJSON)

	// capture podLogs last
	if !cp.SkipPodLogs {
		for _, ns := range cp.Namespaces {
			// background goroutine will log any errors, but they are not fatal
			goCapturePodLogs(&cs, cp.KubeClient, filepath.Join(cp.ClusterDumpDir, namespacesDir, ns, podLogDir), ns)
		}
	}
	cs.wg.Wait()

	return nil
}

// CaptureAllResources captures all resources except Secrets and ConfigMaps
func CaptureAllResources(cp CaptureParams) error {
	cs := captureSync{
		wg:      sync.WaitGroup{},
		channel: make(chan int, maxGoRoutines),
	}

	// discover all the cluster-wide and namespaced GVRs (equiv to kubectl api-resources)
	clusterGVRs, namespacedGVRS, err := discoverGVRs(cp.KubeClient, cp.IncludeConfigMaps)
	if err != nil {
		return err
	}

	for _, ns := range cp.Namespaces {
		// background goroutine will log any errors, but they are not fatal
		goCaptureDynamicRes(&cs, namespacedGVRS, cp.DynamicClient, filepath.Join(cp.ClusterDumpDir, namespacesDir, ns), ns, cp.Redact, cp.Managed, cp.ToJSON)
	}
	goCaptureDynamicRes(&cs, clusterGVRs, cp.DynamicClient, filepath.Join(cp.ClusterDumpDir, clusterWideDir), "", cp.Redact, cp.Managed, cp.ToJSON)

	// capture podLogs last
	if !cp.SkipPodLogs {
		for _, ns := range cp.Namespaces {
			// background goroutine will log any errors, but they are not fatal
			goCapturePodLogs(&cs, cp.KubeClient, filepath.Join(cp.ClusterDumpDir, namespacesDir, ns, podLogDir), ns)
		}
	}
	cs.wg.Wait()

	return nil
}

// read each line, do not sanitize it and write to writer
func writeWithoutSanitize(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	bufWriter := bufio.NewWriter(writer)
	for scanner.Scan() {
		_, err := bufWriter.WriteString(scanner.Text() + "\n")
		if err != nil {
			return err
		}
	}
	bufWriter.Flush()
	return nil
}
