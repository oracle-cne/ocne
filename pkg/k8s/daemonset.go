// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetDaemonSet returns the specified deployment
func GetDaemonSet(client kubernetes.Interface, namespace string, name string) (*v1.DaemonSet, error) {
	deployment, err := client.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	return deployment, err
}

// UpdateDaemonSet updates an existing deployment in a namespace
func UpdateDaemonSet(client kubernetes.Interface, dep *v1.DaemonSet, namespace string) (*v1.DaemonSet, error) {
	return client.AppsV1().DaemonSets(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
}
