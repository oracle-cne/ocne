// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capture

import (
	"bufio"
	"context"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/dump/capture/sanitize"
	"github.com/oracle-cne/ocne/pkg/k8s"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
)

// goCapturePodLogs will get the list of pods in the namespace synchronously then dump each pod in a goroutine
func goCapturePodLogs(cs *captureSync, kubeClient kubernetes.Interface, outDir string, namespace string) {
	podList, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf(fmt.Sprintf("An error occurred while listing Pods in namespace %s: %s\n", namespace, err.Error()))
		return
	}
	for i, pod := range podList.Items {
		// ignore the pod which is doing the dump of node data
		if strings.HasPrefix(pod.Name, "ocne-dump-node") {
			continue
		}

		cs.wg.Add(1)

		// block until a slot is free in the channel buffer.
		cs.channel <- 0
		go func() {
			err := CapturePodLog(kubeClient, outDir, namespace, podList.Items[i].Name, 0)
			if err != nil {
				log.Errorf(err.Error())
			}
			<-cs.channel
			cs.wg.Done()
		}()
	}
}

// CapturePodLog captures the log from the pod in the outDir
func CapturePodLog(kubeClient kubernetes.Interface, outDir, namespace string, podName string, duration int64) error {
	pod, err := k8s.GetPod(kubeClient, namespace, podName)
	if err != nil {
		return err
	}

	// Create directory for the namespace and the pod, under the root level directory containing the bug report
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return fmt.Errorf("Error creating the directory %s: %s", outDir, err.Error())
	}

	// Create logs.txt under the directory for the namespace
	var logPath = filepath.Join(outDir, podName+".txt")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Error creating file %s: %v", logPath, err.Error())
	}
	defer f.Close()

	// Capture logs for both init containers and containers
	var cs []corev1.Container
	var podLogOptions corev1.PodLogOptions
	if duration != 0 {
		podLogOptions.SinceSeconds = &duration
	}
	cs = append(cs, pod.Spec.InitContainers...)
	cs = append(cs, pod.Spec.Containers...)
	// Write the log from all the containers to a single file, with lines differentiating the logs from each of the containers
	for _, c := range cs {
		writeToFile := func(contName string) error {
			podLogOptions.Container = contName
			podLogOptions.InsecureSkipTLSVerifyBackend = true
			podLog, err := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions).Stream(context.TODO())
			if err != nil {
				log.Errorf("Error reading the logs from pod %s: %s\n", podName, err.Error())
				// this is not a fatal error
				return nil
			}
			defer podLog.Close()

			reader := bufio.NewScanner(podLog)
			f.WriteString(fmt.Sprintf(containerStartLog, contName, namespace, podName))
			for reader.Scan() {
				f.WriteString(sanitize.SanitizeString(reader.Text()+"\n", nil))
			}
			f.WriteString(fmt.Sprintf(containerEndLog, contName, namespace, podName))
			return nil
		}
		writeToFile(c.Name)
	}
	return nil
}
