// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
)

// CreateSecret creates a Secret
func CreateSecret(client kubernetes.Interface, namespace string, secret *v1.Secret) error {
	_, err := client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	return err
}

// FindSecretsByLabelKey returns a SecretList for services that match the specified label key
func FindSecretsByLabelKey(client kubernetes.Interface, namespace string, key string) (*v1.SecretList, error) {
	req, _ := labels.NewRequirement(key, selection.Exists, nil)
	sel := labels.NewSelector().Add(*req)
	return FindSecretsByLabelSelector(client, namespace, sel)
}

// FindSecretsByLabelKeyVal returns a SecretList for services that match the specified label key/value pair
func FindSecretsByLabelKeyVal(client kubernetes.Interface, namespace string, key string, val string) (*v1.SecretList, error) {
	req, _ := labels.NewRequirement(key, selection.Equals, []string{val})
	sel := labels.NewSelector().Add(*req)
	return FindSecretsByLabelSelector(client, namespace, sel)
}

// FindSecretsByLabelSelector returns a SecretList for services that match the label selector
func FindSecretsByLabelSelector(client kubernetes.Interface, namespace string, selector labels.Selector) (*v1.SecretList, error) {
	list, err := client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	return list, err
}
