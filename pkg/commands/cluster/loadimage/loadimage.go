// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loadimage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/copy"
	dockerarchive "github.com/containers/image/v5/docker/archive"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/oracle-cne/ocne/pkg/image"
	log "github.com/sirupsen/logrus"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// Load copies the image selected for each node architecture into host Podman.
func Load(kubeconfig, nodeName, selector, source string, out, errOut io.Writer) error {
	if nodeName != "" && selector != "" {
		return fmt.Errorf("node and selector are mutually exclusive")
	}
	if _, err := labels.Parse(selector); err != nil {
		return fmt.Errorf("invalid node selector: %w", err)
	}
	// Bare paths retain archive auto-detection; explicit references use the same
	// transports as the other image commands.
	if alltransports.TransportFromImageName(source) == nil {
		info, err := os.Stat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("image archive must be a regular file; use oci:PATH for a layout")
		}
		reader, err := dockerarchive.NewReader(image.GetSystemContext(""), source)
		if err == nil {
			reader.Close()
			source = "docker-archive:" + source
		} else {
			source = "oci-archive:" + source
		}
	}
	if _, err := alltransports.ParseImageName(source); err != nil {
		return err
	}

	restConfig, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	nodes, err := kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: selector})
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
		return fmt.Errorf("no cluster nodes match the requested selection")
	}
	// Prepare every required architecture before modifying the cluster.
	dir, err := os.MkdirTemp("", "ocne-load-image-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	archives := map[string]string{}
	for _, node := range nodes.Items {
		arch := node.Status.NodeInfo.Architecture
		if arch == "" {
			return fmt.Errorf("node %s has no reported architecture", node.Name)
		}
		if _, ok := archives[arch]; ok {
			continue
		}
		archivePath := filepath.Join(dir, fmt.Sprintf("%d.tar", len(archives)))
		if err := stageImage(source, arch, archivePath); err != nil {
			return fmt.Errorf("prepare image for node %s (%s): %w", node.Name, arch, err)
		}
		archives[arch] = archivePath
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
			archive, err := os.Open(archives[node.Status.NodeInfo.Architecture])
			if err != nil {
				return err
			}
			defer archive.Close()
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
	log.Infof("Loaded image on %d node(s).", len(nodes.Items))
	return nil
}

// stageImage resolves manifest lists using the existing architecture mapping.
// Check single-image configs too: copying an archive alone can accept a foreign
// architecture when the source is not a manifest list.
func stageImage(source, arch, archivePath string) error {
	systemCtx := image.GetSystemContext(arch)
	if systemCtx.ArchitectureChoice == "" {
		return fmt.Errorf("unsupported node architecture %q", arch)
	}
	ref, err := alltransports.ParseImageName(source)
	if err != nil {
		return err
	}
	img, err := ref.NewImage(context.Background(), systemCtx)
	if err != nil {
		return err
	}
	defer img.Close()
	config, err := img.OCIConfig(context.Background())
	if err != nil {
		return err
	}
	if config.OS != systemCtx.OSChoice || config.Architecture != systemCtx.ArchitectureChoice {
		return fmt.Errorf("image platform %s/%s does not match node platform %s/%s",
			config.OS, config.Architecture, systemCtx.OSChoice, systemCtx.ArchitectureChoice)
	}

	// Preserve a selected name when the source transport exposes one. Unnamed
	// OCI sources are imported by digest, just as with podman load.
	name := ""
	if named := ref.DockerReference(); named != nil {
		name = named.String()
	} else if ref.Transport().Name() == "docker-archive" {
		reader, readerRef, err := dockerarchive.NewReaderForReference(systemCtx, ref)
		if err != nil {
			return err
		}
		defer reader.Close()
		tags, err := reader.ManifestTagsForReference(readerRef)
		if err != nil {
			return err
		}
		if len(tags) > 0 {
			name = tags[0]
		}
	} else if ref.Transport().Name() == "oci" || ref.Transport().Name() == "oci-archive" {
		_, name, _ = strings.Cut(ref.StringWithinTransport(), ":")
	}
	dest := "oci-archive:" + archivePath
	if name != "" {
		dest += ":" + name
	}
	return image.Copy(source, dest, arch, copy.CopySystemImage)
}
