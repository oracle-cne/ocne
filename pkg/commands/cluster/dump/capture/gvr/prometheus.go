// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	promMonitoring = "monitoring.coreos.com"
)

// PromClusterResources - resources that are cluster-wide
var PromClusterResources = []schema.GroupVersionResource{}

// PromNamespacedResources - resources that are namespaced
var PromNamespacedResources = []schema.GroupVersionResource{
	{Group: promMonitoring, Version: v1, Resource: "podmonitors"},
	{Group: promMonitoring, Version: v1, Resource: "servicemonitors"},
}
