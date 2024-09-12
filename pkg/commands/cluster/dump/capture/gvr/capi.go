// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	addonsGroup         = "addons.cluster.x-k8s.io"
	bootstrapGroup      = "bootstrap.cluster.x-k8s.io"
	clusterGroup        = "cluster.x-k8s.io"
	clusterctlGroup     = "clusterctl.cluster.x-k8s.io"
	controlPlaneGroup   = "controlplane.cluster.x-k8s.io"
	infrastructureGroup = "infrastructure.cluster.x-k8s.io"
	ipamGroup           = "ipam.cluster.x-k8s.io"
	runtimeGroup        = "runtime.cluster.x-k8s.io"
)

// CapiClusterResources - resources that are cluster-wide
var CapiClusterResources = []schema.GroupVersionResource{
	{Group: runtimeGroup, Version: v1alpha1, Resource: "extensionconfigs"},
}

// CapiNamespacedResources - resources that are namespaced
var CapiNamespacedResources = []schema.GroupVersionResource{
	{Group: addonsGroup, Version: v1beta1, Resource: "clusterresourcesetbindings"},
	{Group: addonsGroup, Version: v1beta1, Resource: "clusterresourcesets"},
	{Group: bootstrapGroup, Version: v1alpha1, Resource: "ocneconfigs"},
	{Group: bootstrapGroup, Version: v1alpha1, Resource: "ocneconfigtemplates"},
	{Group: clusterGroup, Version: v1beta1, Resource: "clusterclasses"},
	{Group: clusterGroup, Version: v1beta1, Resource: "clusters"},
	{Group: clusterGroup, Version: v1beta1, Resource: "machinedeployments"},
	{Group: clusterGroup, Version: v1beta1, Resource: "machinehealthchecks"},
	{Group: clusterGroup, Version: v1beta1, Resource: "machinepools"},
	{Group: clusterGroup, Version: v1beta1, Resource: "machines"},
	{Group: clusterGroup, Version: v1beta1, Resource: "machinesets"},
	{Group: clusterctlGroup, Version: v1alpha3, Resource: "providers"},
	{Group: controlPlaneGroup, Version: v1alpha1, Resource: "ocnecontrolplanes"},
	{Group: controlPlaneGroup, Version: v1alpha1, Resource: "ocnecontrolplanetemplates"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ociclusteridentities"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ociclusters"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ociclustertemplates"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimachinepools"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimachines"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimachinetemplates"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedclusters"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedclustertemplates"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedcontrolplanes"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedcontrolplanetemplates"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedmachinepools"},
	{Group: infrastructureGroup, Version: v1beta2, Resource: "ocimanagedmachinepooltemplates"},
	{Group: ipamGroup, Version: v1alpha1, Resource: "ipaddressclaims"},
	{Group: ipamGroup, Version: v1alpha1, Resource: "ipaddresses"},
}
