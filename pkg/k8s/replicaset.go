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

// GetReplicaSets returns a list of replica sets for a given owner
func GetReplicaSets(client kubernetes.Interface, namespace string, uid string) ([]*v1.ReplicaSet, error) {
	replicaSets, err := client.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := []*v1.ReplicaSet{}
	for _, rs := range replicaSets.Items {
		for _, or := range rs.OwnerReferences {
			if string(or.UID) == uid {
				ret = append(ret, &rs)
				break
			}
		}
	}

	return ret, nil
}

// GetReplicaSetPods returns a list of pods controlled by a replica set
func GetReplicaSetPods(client kubernetes.Interface, replicaSet *v1.ReplicaSet) ([]*corev1.Pod, error) {
	return GetPodsByOwner(client, replicaSet.Namespace, string(replicaSet.UID))
}
