// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	k8sApps      = "apps"
	k8sAdmission = "admissionregistration.k8s.io"
	k8sNetwork   = "networking.k8s.io"
	k8sRbac      = "rbac.authorization.k8s.io"
)

// K8sClusterResources - resources that are cluster-wide
var K8sClusterResources = []schema.GroupVersionResource{
	{Group: "", Version: v1, Resource: "namespaces"},
	{Group: "", Version: v1, Resource: "nodes"},
	{Group: "", Version: v1, Resource: "persistentvolumes"},

	{Group: k8sAdmission, Version: v1, Resource: "mutatingwebhookconfigurations"},
	{Group: k8sAdmission, Version: v1, Resource: "validatingwebhookconfigurations"},

	{Group: k8sRbac, Version: v1, Resource: "clusterroles"},
	{Group: k8sRbac, Version: v1, Resource: "clusterrolebindings"},
}

// K8sNamespacedResources - resources that are namespaced
var K8sNamespacedResources = []schema.GroupVersionResource{
	{Group: "", Version: v1, Resource: "endpoints"},
	{Group: "", Version: v1, Resource: "events"},
	{Group: "", Version: v1, Resource: "persistentvolumeclaims"},
	{Group: "", Version: v1, Resource: "pods"},
	{Group: "", Version: v1, Resource: "serviceaccounts"},
	{Group: "", Version: v1, Resource: "services"},

	{Group: k8sApps, Version: v1, Resource: "deployments"},
	{Group: k8sApps, Version: v1, Resource: "daemonsets"},
	{Group: k8sApps, Version: v1, Resource: "replicasets"},
	{Group: k8sApps, Version: v1, Resource: "statefulsets"},

	{Group: k8sNetwork, Version: v1, Resource: "ingresses"},
	{Group: k8sNetwork, Version: v1, Resource: "networkpolicies"},

	{Group: k8sRbac, Version: v1, Resource: "roles"},
	{Group: k8sRbac, Version: v1, Resource: "rolebindings"},
	{Group: k8sRbac, Version: v1, Resource: "serviceaccounts"},
}
