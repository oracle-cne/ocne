// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
)

func Backup(kubeconfig string, out string) error {
	// get a kubernetes client
	restConfig, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}

	// sanity check to make sure we can access cluster
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// get config needed to use kubectl
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, kubeconfig, metav1.NamespaceSystem, []string{"\"level\":\"info\""}, false)
	if err != nil {
		return err
	}

	// get etcd pod
	podName, err := getEtcdPod(kubeClient)
	if err != nil {
		return err
	}
	if err := k8s.WaitUntilPodReady(kubeClient, metav1.NamespaceSystem, podName); err != nil {
		return err
	}

	// Exec in etcd pod to backup etcd
	log.Infof("Running etcd backup on pod %s", podName)
	if err := kubectl.RunCommand(kcConfig, podName, "sh", "-c", "ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key snapshot save /tmp/snapshot.db"); err != nil {
		return err
	}

	// copy the snapshot from etcd pod to the local directory
	cc := &kubectl.CopyConfig{
		KubectlConfig: kcConfig,
		FilePaths:     []kubectl.FilePath{{LocalPath: out, RemotePath: "/tmp/snapshot.db"}},
		PodName:       podName,
	}
	if err := kubectl.CopyFilesFromPod(cc, "Copying data from etcd pod to the local system"); err != nil {
		return err
	}

	log.Info("Etcd successfully backed up")

	return nil
}

// get the name of the first etcd pod
func getEtcdPod(client kubernetes.Interface) (string, error) {
	requirements := make([]labels.Requirement, 0)
	req1, _ := labels.NewRequirement("component", selection.Equals, []string{"etcd"})
	requirements = append(requirements, *req1)
	req2, _ := labels.NewRequirement("tier", selection.Equals, []string{"control-plane"})
	requirements = append(requirements, *req2)
	sel := labels.NewSelector().Add(requirements...)
	list, err := client.CoreV1().Pods(metav1.NamespaceSystem).List(context.TODO(), metav1.ListOptions{LabelSelector: sel.String()})

	if err != nil {
		return "", err
	}
	if len(list.Items) < 1 {
		return "", fmt.Errorf("Can not find etcd pod")
	}
	return list.Items[0].Name, nil
}
