// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetDaemonSets returns a list of daemon sets subject to a selector
func GetDaemonSets(client kubernetes.Interface, namespace string, selector string) ([]v1.DaemonSet, error) {
	daemonSets, err := client.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	return daemonSets.Items, nil
}

// GetDamonSetsWithAnnotations returns a list of daemon sets with annotations
func GetDaemonSetsWithAnnotations(client kubernetes.Interface, namespace string, annots map[string]string) ([]*v1.DaemonSet, error) {
	daemonSets, err := client.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := []*v1.DaemonSet{}
	for _, ds := range daemonSets.Items {
		if stringMapSubset(ds.Annotations, annots) {
			ret = append(ret, &ds)
		}
	}
	return ret, nil
}

// GetDaemonSet returns the specified daemon set
func GetDaemonSet(client kubernetes.Interface, namespace string, name string) (*v1.DaemonSet, error) {
	daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	return daemonSet, err
}

// UpdateDaemonSet updates an existing daemon set in a namespace
func UpdateDaemonSet(client kubernetes.Interface, ds *v1.DaemonSet, namespace string) (*v1.DaemonSet, error) {
	return client.AppsV1().DaemonSets(namespace).Update(context.TODO(), ds, metav1.UpdateOptions{})
}


// GetDaemonSetPods returns a list of pods controlled by a daemon set
func GetDaemonSetPods(client kubernetes.Interface, ds *v1.DaemonSet) ([]*corev1.Pod, error) {
	return GetPodsByOwner(client, ds.Namespace, string(ds.UID))
}
