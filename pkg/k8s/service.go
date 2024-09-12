// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetService returns the specified service.
func GetService(client kubernetes.Interface, namespace string, name string) (*v1.Service, error) {
	service, err := client.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	return service, err
}

// WaitForService - wait for service to exist
func WaitForService(client kubernetes.Interface, namespace string, name string) error {
	count := 0
	maxRetry := 12
	for {
		_, err := GetService(client, namespace, name)
		if err == nil {
			break
		}
		count++
		if count > maxRetry {
			return err
		}
		time.Sleep(time.Second * 5)
	}
	return nil
}

// FindServicesByLabelKey returns a ServiceList for services that match the specified label key
func FindServicesByLabelKey(client kubernetes.Interface, namespace string, key string) (*v1.ServiceList, error) {
	req, _ := labels.NewRequirement(key, selection.Exists, nil)
	sel := labels.NewSelector().Add(*req)
	return FindServicesByLabelSelector(client, namespace, sel)
}

// FindServicesByLabelKeyVal returns a ServiceList for services that match the specified label key/value pair
func FindServicesByLabelKeyVal(client kubernetes.Interface, namespace string, key string, val string) (*v1.ServiceList, error) {
	req, _ := labels.NewRequirement(key, selection.Equals, []string{val})
	sel := labels.NewSelector().Add(*req)
	return FindServicesByLabelSelector(client, namespace, sel)
}

// FindServicesByLabelSelector returns a ServiceList for services that match the label selector
func FindServicesByLabelSelector(client kubernetes.Interface, namespace string, selector labels.Selector) (*v1.ServiceList, error) {
	list, err := client.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	return list, err
}

func CreateService(client kubernetes.Interface, svc *v1.Service) error {
	_, err := client.CoreV1().Services(svc.ObjectMeta.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	return err
}

func DeleteService(client kubernetes.Interface, namespace string, name string) error {
	return client.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}
