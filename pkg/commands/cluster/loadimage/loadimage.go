// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loadimage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// Load streams a local image archive into host Podman on each selected node.
func Load(kubeconfig, nodeName, archivePath string, out, errOut io.Writer) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("image archive must be a regular file")
	}

	restConfig, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	nodes, err := k8s.GetNodeList(kubeClient)
	if err != nil {
		return err
	}
	if nodeName != "" {
		matched := false
		for _, node := range nodes.Items {
			if node.Name == nodeName {
				nodes.Items = []v1.Node{node}
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("node %q not found", nodeName)
		}
	}
	if len(nodes.Items) == 0 {
		return fmt.Errorf("no cluster nodes found")
	}
	if err := k8s.CreateNamespaceIfNotExists(kubeClient, constants.OCNESystemNamespace); err != nil {
		return err
	}

	for _, node := range nodes.Items {
		// An independent name avoids interfering with consoles or concurrent loads.
		prefix := "load-image"
		pod := k8s.BasicAdminPod(node.Name, constants.OCNESystemNamespace, prefix, false)
		pod.Name = "load-image-" + string(uuid.NewUUID())
		pod.Labels["run"] = pod.Name
		pod.Spec.Containers[0].Name = "load-image"
		pod.Spec.Containers[0].Command = []string{"sleep", "infinity"}
		pod.Spec.Containers[0].Stdin = false
		pod.Spec.Containers[0].TTY = false
		if _, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create image loader on node %s: %w", node.Name, err)
		}
		err = func() (loadErr error) {
			defer func() {
				if err := k8s.DeletePod(kubeClient, pod.Namespace, pod.Name); err != nil {
					loadErr = errors.Join(loadErr, fmt.Errorf("remove image loader: %w", err))
				}
			}()
			if err := k8s.WaitForPod(kubeClient, pod.Namespace, pod.Name); err != nil {
				return err
			}
			if _, err := archive.Seek(0, io.SeekStart); err != nil {
				return err
			}
			request := kubeClient.CoreV1().RESTClient().Post().
				Resource("pods").Namespace(pod.Namespace).Name(pod.Name).
				SubResource("exec").VersionedParams(&v1.PodExecOptions{
				Container: pod.Spec.Containers[0].Name,
				Command:   []string{"chroot", "/hostroot", "podman", "load"},
				Stdin:     true, Stdout: true, Stderr: true,
			}, scheme.ParameterCodec)
			executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
			if err != nil {
				return err
			}
			return executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
				Stdin: archive, Stdout: out, Stderr: errOut,
			})
		}()
		if err != nil {
			return fmt.Errorf("load image archive on node %s: %w", node.Name, err)
		}
	}
	_, err = fmt.Fprintf(out, "Loaded image archive on %d node(s).\n", len(nodes.Items))
	return err
}
