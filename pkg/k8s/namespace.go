// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetNamespaceList returns the list of namespaces
func GetNamespaceList(client kubernetes.Interface) (*v1.NamespaceList, error) {
	return client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
}

// GetNamespaces returns the namespace names
func GetNamespaces(cli kubernetes.Interface) ([]string, error) {
	names := []string{}
	list, err := GetNamespaceList(cli)
	if err != nil {
		return nil, err
	}

	for i, _ := range list.Items {
		names = append(names, list.Items[i].Name)
	}
	return names, nil
}

// CreateNamespaceIfNotExists creates a namespace if it does not already exist
func CreateNamespaceIfNotExists(client kubernetes.Interface, name string) error {
	_, err := getNamespace(client, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return createNamespace(client, name)
		}
		return err
	}
	return nil
}

// getNamespace returns a single namespace
func getNamespace(client kubernetes.Interface, name string) (*v1.Namespace, error) {
	return client.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// createNamespace creates a single namespace
func createNamespace(client kubernetes.Interface, name string) error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// kube-controller-manager makes the default service account
	// after seeing the new namespace being created.  Wait for that
	// service account to exist so that pods can be created in the
	// namespace that don't need privileges.
	return WaitForServiceAccount(client, "default", name)
}
