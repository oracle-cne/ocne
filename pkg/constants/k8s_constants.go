// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	// OCNESystemNamespace is the OCNE system namespace
	OCNESystemNamespace = "ocne-system"

	// OCNECatalogLabelKey is the catalog service label key
	// that marks catalog services
	OCNECatalogLabelKey = "catalog.ocne.io/is-catalog"

	// OCNECatalogAnnotationKey is the annoation key that
	// contains the catalog friendly name
	OCNECatalogAnnotationKey = "catalog.ocne.io/name"

	// OCNECatalogURIKey is the annotation key that
	// contains any extra relative path required to
	// query the catalog
	OCNECatalogURIKey = "catalog.ocne.io/uri"

	// OCNECatalogProtoKey is the annotation key that
	// indicates the catalog protocol.  The current
	// valid values are "helm" and "artifacthub".
	OCNECatalogProtoKey = "catalog.ocne.io/protocol"

	// DefaultCatalogName is the default OCNE catalog name
	DefaultCatalogName = "Oracle Cloud Native Environment Application Catalog"

	// CatalogServiceName is the default OCNE catalog service name
	CatalogServiceName = "ocne-catalog"

	// UIServiceName is the default OCNE UI service name
	UIServiceName = "ui"

	// UISecretNameTLS is the name of the TLS secret for the UI
	UISecretNameTLS = "ui-tls"

	// CASecretNameTLS is the name of the TLS secret that stores the CA
	// certificate and key used to sign the UI certificate
	CASecretNameTLS = "certificate-authority-tls"

	CNIFlannel = "flannel"
	CNINone    = "none"

	CNIFlannelRelease   = "flannel"
	CNIFlannelNamespace = "kube-flannel"
	CNIFlannelChart     = "flannel"
	CNIFlannelVersion   = "0.22.3"
	CNIFlannelImageTag  = "v0.22.3-2"

	CatalogRelease   = "ocne-catalog"
	CatalogNamespace = "ocne-system"
	CatalogChart     = "ocne-catalog"
	CatalogVersion   = "2.0.0"
	CatalogName      = "Oracle Cloud Native Environment Application Catalog"

	CommunityCatalogName      = "ArtifactHub Community Catalog"
	CommunityCatalogURI       = "https://artifacthub.io"
	CommunityCatalogNamespace = OCNESystemNamespace

	UIRelease        = "ui"
	UINamespace      = "ocne-system"
	UIChart          = "ui"
	UIVersion        = ""
	UIInitContainer  = "ui-plugins"
	UIPluginsVersion = "v2.0.0"

	CertManagerRelease   = "cert-manager"
	CertManagerNamespace = "cert-manager"
	CertManagerChart     = "cert-manager"
	CertManagerVersion   = ""

	CoreCAPIRelease    = "core-capi"
	CoreCAPINamespace  = "capi-system"
	CoreCAPIChart      = "core-capi"
	CoreCAPIVersion    = "1.7.1"
	CoreCAPIDeployment = "core-capi-controller-manager"

	KubeadmBootstrapCAPIRelease    = "bootstrap-capi"
	KubeadmBootstrapCAPINamespace  = "capi-kubeadm-bootstrap-system"
	KubeadmBootstrapCAPIChart      = "bootstrap-capi"
	KubeadmBootstrapCAPIVersion    = "1.7.1"
	KubeadmBootstrapCAPIDeployment = "bootstrap-capi-controller-manager"

	KubeadmControlPlaneCAPIRelease    = "control-plane-capi"
	KubeadmControlPlaneCAPINamespace  = "capi-kubeadm-control-plane-system"
	KubeadmControlPlaneCAPIChart      = "control-plane-capi"
	KubeadmControlPlaneCAPIVersion    = "1.7.1"
	KubeadmControlPlaneCAPIDeployment = "control-plane-capi-controller-manager"

	OCICAPIRelease    = "capoci"
	OCICAPINamespace  = "cluster-api-provider-oci-system"
	OCICAPIChart      = "oci-capi"
	OCICAPIVersion    = ""
	OCICAPIDeployment = "capoci-controller-manager"

	DefaultPodImage = "container-registry.oracle.com/os/oraclelinux:8"
	ScriptMountPath = "/ocne-scripts"
	KubeNamespace   = "kube-system"
	KubeCMName      = "kubeadm-config"

	CoreDNSImage   = "container-registry.oracle.com/olcne/coredns"
	KubeProxyImage = "container-registry.oracle.com/olcne/kube-proxy"
	CurrentTag     = "current"
)
