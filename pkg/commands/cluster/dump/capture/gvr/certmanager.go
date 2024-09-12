// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	cm     = "cert-manager.io"
	cmAcme = "acme.cert-manager.io"
)

// CertmanagerClusterResources - resources that are cluster-wide
var CertmanagerClusterResources = []schema.GroupVersionResource{
	{Group: cm, Version: v1, Resource: "certificatesigningrequests"},
	{Group: cm, Version: v1, Resource: "clusterissuers"},
}

// CertmanagerClusterResources -  resources that are namespaced
var CertmanagerNamespacedResources = []schema.GroupVersionResource{
	{Group: cm, Version: v1, Resource: "certificaterequests"},
	{Group: cm, Version: v1, Resource: "certificates"},
	{Group: cm, Version: v1, Resource: "issuers"},

	{Group: cmAcme, Version: v1, Resource: "challenges"},
	{Group: cmAcme, Version: v1, Resource: "certificaterequests"},
}
