// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getDeployment returns the specified deployment
func getDeployment(client kubernetes.Interface, namespace string, name string) (*v1.Deployment, error) {
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	return deployment, err
}

// WaitForDeployment - wait for deployment to be ready
func WaitForDeployment(client kubernetes.Interface, namespace string, name string, expectedReplicas int32) error {
	count := 0
	maxRetry := 48
	for {
		deployment, err := getDeployment(client, namespace, name)
		if err == nil {
			if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas >= expectedReplicas && deployment.Status.AvailableReplicas >= expectedReplicas {
				return nil
			}
		}
		count++
		if count > maxRetry {
			if err != nil {
				return err
			}
			return errors.New(fmt.Sprintf("timed out waiting for deployment %s/%s to be ready", namespace, name))
		}
		time.Sleep(time.Second * 10)
	}
}