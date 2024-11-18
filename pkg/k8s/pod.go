// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodOptions are used to create a pod that mounts a configmap
type PodOptions struct {
	NodeName      string
	Namespace     string
	PodName       string
	ImageName     string
	ConfigMapName string
	HostPID       bool
	HostNetwork   bool
	Env           []v1.EnvVar
}

func GetPod(client kubernetes.Interface, namespace string, name string) (*v1.Pod, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	return pod, err
}

func DeletePod(client kubernetes.Interface, namespace string, name string) error {
	if err := client.CoreV1().Pods(namespace).Delete(context.Background(), name, *metav1.NewDeleteOptions(0)); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func GetPodsBySelector(client kubernetes.Interface, namespace string, selector string) (*v1.PodList, error) {
	return client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
}

// WaitUntilPodReady waits until a pod is ready
func WaitUntilPodReady(client kubernetes.Interface, namespace string, name string) error {
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: fmt.Sprintf("Waiting for pod %s/%s to be ready", namespace, name),
			WaitFunction: func(i interface{}) error {
				count := 0
				maxRetry := 48
				for {
					pod, err := GetPod(client, namespace, name)
					if err == nil {
						if len(pod.Status.ContainerStatuses) == 1 && pod.Status.ContainerStatuses[0].Ready {
							return nil
						}
					}
					count++
					if count > maxRetry {
						if err != nil {
							return err
						}
						return errors.New(fmt.Sprintf("Timed out waiting for pod %s/%s to be ready", namespace, name))
					}
					time.Sleep(time.Second * 5)
				}
			},
		},
	})
	if haveError == true {
		return fmt.Errorf("Timeout waiting for pod %s/%s to be ready", namespace, name)
	}
	return nil
}

func WaitForPod(client kubernetes.Interface, namespace string, name string) error {
	count := 0
	maxRetry := 48
	for {
		pod, err := GetPod(client, namespace, name)
		if err == nil {
			if len(pod.Status.ContainerStatuses) == 1 && pod.Status.ContainerStatuses[0].Ready {
				return nil
			}
		}
		count++
		if count > maxRetry {
			if err != nil {
				return err
			}
			return errors.New(fmt.Sprintf("timed out waiting for pod %s/%s to be ready", namespace, name))
		}
		time.Sleep(time.Second * 5)
	}
}

// BasicAdminPod returns the skeleton of a pod specification that can be
// used to create a privileged pod on a node.
func BasicAdminPod(node string, namespace string, podNamePrefix string, toolbox bool) *v1.Pod {
	var image string
	if toolbox {
		image = "olcne/ock:toolbox"
	} else {
		image = "os/oraclelinux:8"
	}
	privileged := true
	hostPathType := v1.HostPathDirectory
	volumeName := "host-root"
	podName := strings.ReplaceAll(fmt.Sprintf("%s-%s", podNamePrefix, node), ".", "-")
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"run": podName,
			},
		},
		Spec: v1.PodSpec{
			NodeName:      node,
			RestartPolicy: v1.RestartPolicyNever,
			HostNetwork:   true,
			HostPID:       true,
			Containers: []v1.Container{
				{
					Name:      podName,
					Image:     image,
					Stdin:     true,
					StdinOnce: true,
					TTY:       true,

					SecurityContext: &v1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: "/hostroot",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/",
							Type: &hostPathType,
						},
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:      "node-role.kubernetes.io/control-plane",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/not-ready",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/unschedulable",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/unreachable",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}
}

// CreateScriptsPod create a pod that mounts the config map with the same name as the pod
func CreateScriptsPod(client kubernetes.Interface, p *PodOptions) error {
	privileged := true
	scriptVolumeName := "ocne-scripts"
	hostVolumeName := "host-root"
	hostPathType := v1.HostPathDirectory
	var accessMode int32 = 0o0500

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.PodName,
			Namespace: p.Namespace,
		},
		Spec: v1.PodSpec{
			NodeName:    p.NodeName,
			HostPID:     p.HostPID,
			HostNetwork: p.HostNetwork,
			SecurityContext: &v1.PodSecurityContext{
				RunAsUser:    util.Int64Ptr(0),
				RunAsGroup:   util.Int64Ptr(0),
				RunAsNonRoot: util.BoolPtr(false),
			},
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:    "app",
					Image:   p.ImageName,
					Command: []string{"sleep", "10d"},
					Env:     p.Env,
					SecurityContext: &v1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      scriptVolumeName,
							MountPath: constants.ScriptMountPath,
						},
						{
							Name:      hostVolumeName,
							MountPath: "/hostroot",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: scriptVolumeName,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: p.ConfigMapName,
							},
							DefaultMode: &accessMode,
						},
					},
				},
				{
					Name: hostVolumeName,
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/",
							Type: &hostPathType,
						},
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:      "node-role.kubernetes.io/control-plane",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/not-ready",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/unschedulable",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
				{
					Key:      "node.kubernetes.io/unreachable",
					Effect:   v1.TaintEffectNoSchedule,
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}

	_, err := client.CoreV1().Pods(p.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	return err
}

// StartAdminPodOnNode creates a privileged pod on a node, executes a command,
// and tears down the pod.
func StartAdminPodOnNode(client kubernetes.Interface, node string, namespace string, podNamePrefix string, toolbox bool) (*v1.Pod, error) {
	pod := BasicAdminPod(node, namespace, podNamePrefix, toolbox)

	// Ensure that the cluster is up
	_, err := WaitUntilGetNodesSucceeds(client)
	if err != nil {
		return nil, err
	}

	// Ensure that the node actually exists
	_, err = GetNode(client, node)
	if err != nil {
		return nil, err
	}

	_, err = client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	err = WaitForPod(client, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if err != nil {
		return nil, err
	}
	return pod, nil
}
