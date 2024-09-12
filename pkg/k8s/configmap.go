// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetConfigmap(client kubernetes.Interface, namespace string, name string) (*corev1.ConfigMap, error) {
	// Retrieve the existing ConfigMap
	return client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func UpdateConfigMap(client kubernetes.Interface, configMap *corev1.ConfigMap, namespace string) (*corev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
}

func DeleteConfigmap(client kubernetes.Interface, namespace string, name string) error {
	if err := client.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, *metav1.NewDeleteOptions(0)); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func CreateConfigmap(client kubernetes.Interface, configMap *corev1.ConfigMap) error {
	_, err := client.CoreV1().ConfigMaps(configMap.ObjectMeta.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	return err
}

func CreateConfigMapWithData(client kubernetes.Interface, namespace string, name string, data map[string]string) error {
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Immutable: nil,
		Data:      data,
	}

	return CreateConfigmap(client, &cm)
}
