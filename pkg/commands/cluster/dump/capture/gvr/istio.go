// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	istioExtension  = "extensions.istio.io"
	istioInstall    = "install.istio.io"
	istioNetworking = "networking.istio.io"
	istioSecurity   = "security.istio.io"
	istioTelemetry  = "telemetry.istio.io"
)

// IstioClusterResources - resources that are cluster-wide
var IstioClusterResources = []schema.GroupVersionResource{}

// IstioNamespacedResources - resources that are namespaced
var IstioNamespacedResources = []schema.GroupVersionResource{
	{Group: istioExtension, Version: v1alpha1, Resource: "wasmplugins"},
	{Group: istioInstall, Version: v1alpha1, Resource: "istiooperators"},

	{Group: istioNetworking, Version: v1beta1, Resource: "destinationrules"},
	{Group: istioNetworking, Version: v1alpha3, Resource: "envoyfilters"},
	{Group: istioNetworking, Version: v1beta1, Resource: "gateways"},
	{Group: istioNetworking, Version: v1beta1, Resource: "proxyconfigs"},
	{Group: istioNetworking, Version: v1beta1, Resource: "serviceentries"},
	{Group: istioNetworking, Version: v1beta1, Resource: "sidecars"},
	{Group: istioNetworking, Version: v1beta1, Resource: "workloadentries"},
	{Group: istioNetworking, Version: v1beta1, Resource: "workloadgroups"},
	{Group: istioNetworking, Version: v1beta1, Resource: "destinationrules"},

	{Group: istioSecurity, Version: v1, Resource: "authorizationpolicies"},
	{Group: istioSecurity, Version: v1beta1, Resource: "peerauthentications"},
	{Group: istioSecurity, Version: v1, Resource: "requestauthentications"},

	{Group: istioTelemetry, Version: v1alpha1, Resource: "telemetries"},
}
