// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import "github.com/oracle-cne/ocne/pkg/constants"

type LibvirtProvider struct {
	SessionURI                   *string `yaml:"uri,omitempty"`
	SshKey                       *string `yaml:"sshKey,omitempty"`
	StoragePool                  *string `yaml:"storagePool,omitempty"`
	Network                      *string `yaml:"network,omitempty"`
	ControlPlaneNode             Node    `yaml:"controlPlaneNode"`
	WorkerNode                   Node    `yaml:"workerNode"`
	BootVolumeName               *string `yaml:"bootVolumeName,omitempty"`
	BootVolumeContainerImagePath *string `yaml:"bootVolumeContainerImagePath,omitempty"`
}

type OciInstanceShape struct {
	Shape *string `yaml:"shape,omitempty"`
	Ocpus *int    `yaml:"ocpus,omitempty"`
}

type LoadBalancer struct {
	Subnet1 *string `yaml:"subnet1,omitempty"`
	Subnet2 *string `yaml:"subnet2,omitempty"`
}

type OciImageSet struct {
	Amd64 *string `yaml:"amd64,omitempty"`
	Arm64 *string `yaml:"arm64,omitempty"`
}

type OciProvider struct {
	KubeConfigPath    *string          `yaml:"kubeconfig,omitempty"`
	Compartment       *string          `yaml:"compartment,omitempty"`
	Namespace         *string          `yaml:"namespace,omitempty"`
	ControlPlaneShape OciInstanceShape `yaml:"controlPlaneShape"`
	Images            OciImageSet      `yaml:"images"`
	WorkerShape       OciInstanceShape `yaml:"workerShape"`
	SelfManaged       *bool            `yaml:"selfManaged,omitempty"`
	LoadBalancer      LoadBalancer     `yaml:"loadBalancer"`
	Vcn               *string          `yaml:"vcn,omitempty"`
	ImageBucket       *string          `yaml:"imageBucket,omitempty"`
	Proxy             Proxy            `yaml:"proxy"`
}

type OlvmProvider struct {
	Namespace           *string              `yaml:"namespace,omitempty"`
	SelfManaged         *bool                `yaml:"selfManaged,omitempty"`
	Proxy               Proxy                `yaml:"proxy"`
	NetworkInterface    *string              `yaml:"networkInterface,omitempty"`
	OlvmCluster         OlvmCluster          `yaml:"olvmCluster"`
	ControlPlaneMachine OlvmMachine          `yaml:"controlPlaneMachine"`
	WorkerMachine       OlvmMachine          `yaml:"workerMachine"`
	LocalAPIEndpoint    OlvmLocalAPIEndpoint `yaml:"localAPIEndpoint"`
}

type OlvmCluster struct {
	ControlPlaneEndpoint OlvmControlPlaneEndpoint `yaml:"controlPlaneEndpoint"`
	DatacenterName       *string                  `yaml:"ovirtDatacenterName,omitempty"`
	OVirtAPI             OlvmOvirtAPI             `yaml:"ovirtAPI"`
	OVirtOck             OlvmOvirtOck             `yaml:"ovirtOCK"`
	OlvmVmIpProfile      OlvmVmIpProfile          `yaml:"olvmVmIpProfile"`
}

type OlvmOvirtOck struct {
	DiskName          *string `yaml:"diskName,omitempty"`
	DiskSize          *string `yaml:"diskSize,omitempty"`
	StorageDomainName *string `yaml:"storageDomainName,omitempty"`
}

type OlvmOvirtAPI struct {
	ServerURL    *string `yaml:"serverURL,omitempty"`
	ServerCA     *string `yaml:"serverCA,omitempty"`
	ServerCAPath *string `yaml:"serverCAPath,omitempty"`
}

type OlvmControlPlaneEndpoint struct {
	Host *string `yaml:"host,omitempty"`
	Port *string `yaml:"port,omitempty"`
}

type OlvmVmIpProfile struct {
	Name              *string `yaml:"name,omitempty"`
	StartingIpAddress *string `yaml:"startingIpAddress,omitempty"`
	Device            *string `yaml:"device,omitempty"`
	Gateway           *string `yaml:"gateway,omitempty"`
	Netmask           *string `yaml:"netmask,omitempty"`
}

type OlvmMachine struct {
	Memory              *string            `yaml:"memory,omitempty"`
	Network             OlvmMachineNetwork `yaml:"network"`
	Cpu                 OlvmMachineCpu     `yaml:"cpu"`
	OVirtClusterName    *string            `yaml:"ovirtClusterName,omitempty"`
	OlvmVmIpProfileName *string            `yaml:"olvmVmIpProfileName,omitempty"`
	VMTemplateName      *string            `yaml:"vmTemplateName,omitempty"`
}

type OlvmMachineCpu struct {
	Architecture *string               `yaml:"architecture,omitempty"`
	Topology     OlvmMachineCpuToplogy `yaml:"topology"`
}

type OlvmMachineCpuToplogy struct {
	Cores   *int `yaml:"cores,omitempty"`
	Sockets *int `yaml:"sockets,omitempty"`
	Threads *int `yaml:"threads,omitempty"`
}

type OlvmMachineNetwork struct {
	NetworkName     *string `yaml:"networkName,omitempty"`
	InterfaceType   *string `yaml:"interfaceType,omitempty"`
	VnicName        *string `yaml:"vnicName,omitempty"`
	VnicProfileName *string `yaml:"vnicProfileName,omitempty"`
}

type OlvmLocalAPIEndpoint struct {
	BindPort         *int    `yaml:"bindPort,omitempty"`
	AdvertiseAddress *string `yaml:"advertiseAddress,omitempty"`
}

type ByoProvider struct {
	AutomaticTokenCreation *bool   `yaml:"automaticTokenCreation,omitempty"`
	NetworkInterface       *string `yaml:"networkInterface,omitempty"`
}

type Node struct {
	Memory  *string `yaml:"memory,omitempty"`
	CPUs    *int    `yaml:"cpu,omitempty"`
	Storage *string `yaml:"storage,omitempty"`
}

type CertificateInformation struct {
	Country *string `yaml:"country,omitempty"`
	Org     *string `yaml:"org,omitempty"`
	OrgUnit *string `yaml:"orgUnit,omitempty"`
	State   *string `yaml:"state,omitempty"`
}

type Providers struct {
	Libvirt LibvirtProvider `yaml:"libvirt"`
	Oci     OciProvider     `yaml:"oci"`
	Byo     ByoProvider     `yaml:"byo"`
	Olvm    OlvmProvider    `yaml:"olvm"`
}

type Proxy struct {
	HttpsProxy *string `yaml:"httpsProxy,omitempty"`
	HttpProxy  *string `yaml:"httpProxy,omitempty"`
	NoProxy    *string `yaml:"noProxy,omitempty"`
}

type Catalog struct {
	Protocol  *string `yaml:"protocol,omitempty"`
	URI       *string `yaml:"uri,omitempty"`
	Name      *string `yaml:"name,omitempty"`
	Namespace *string `yaml:"namespace,omitempty"`
}

type Application struct {
	Name       *string     `yaml:"name,omitempty"`
	Release    *string     `yaml:"release,omitempty"`
	Version    *string     `yaml:"version,omitempty"`
	Catalog    *string     `yaml:"catalog,omitempty"`
	Namespace  *string     `yaml:"namespace,omitempty"`
	Config     interface{} `yaml:"config"`
	ConfigFrom *string     `yaml:"configFrom,omitempty"`
}

type EphemeralClusterConfig struct {
	Name     *string `yaml:"name,omitempty"`
	Preserve *bool   `yaml:"preserve,omitempty"`
	Node     Node    `yaml:"node"`
}

type Config struct {
	Providers                Providers              `yaml:"providers,omitempty"`
	KubeConfig               *string                `yaml:"kubeconfig,omitempty"`
	AutoStartUI              *string                `yaml:"autoStartUI,omitempty"`
	Proxy                    Proxy                  `yaml:"proxy,omitempty"`
	KubeAPIServerBindPort    *uint16                `yaml:"kubeApiServerBindPort,omitempty"`
	KubeAPIServerBindPortAlt *uint16                `yaml:"kubeApiServerBindPortAlt,omitempty"`
	PodSubnet                *string                `yaml:"podSubnet,omitempty"`
	ServiceSubnet            *string                `yaml:"serviceSubnet,omitempty"`
	Registry                 *string                `yaml:"registry,omitempty"`
	CertificateInformation   CertificateInformation `yaml:"certificateInformation,omitempty"`
	OsTag                    *string                `yaml:"osTag,omitempty"`
	OsRegistry               *string                `yaml:"osRegistry,omitempty"`
	KubeProxyMode            *string                `yaml:"kubeProxyMode,omitempty"`
	BootVolumeContainerImage *string                `yaml:"bootVolumeContainerImage,omitempty"`
	CNI                      *string                `yaml:"cni,omitempty"`
	Headless                 *bool                  `yaml:"headless,omitempty"`
	Catalog                  *bool                  `yaml:"catalog,omitempty"`
	CommunityCatalog         *bool                  `yaml:"communityCatalog,omitempty"`
	EphemeralConfig          EphemeralClusterConfig `yaml:"ephemeralCluster"`
	Quiet                    *bool                  `yaml:"quiet,omitempty"`
	KubeVersion              *string                `yaml:"kubernetesVersion,omitempty"`
	SshPublicKeyPath         *string                `yaml:"sshPublicKeyPath,omitempty"`
	SshPublicKey             *string                `yaml:"sshPublicKey,omitempty"`
	Password                 *string                `yaml:"password,omitempty"`
	ExtraIgnitionInline      *string                `yaml:"extraIgnitionInline,omitempty"`
	ExtraIgnition            *string                `yaml:"extraIgnition,omitempty"`
}

type ClusterConfig struct {
	WorkingDirectory         *string                `yaml:"directory,omitempty"`
	Name                     *string                `yaml:"name,omitempty"`
	Provider                 *string                `yaml:"provider,omitempty"`
	Providers                Providers              `yaml:"providers"`
	Proxy                    Proxy                  `yaml:"proxy"`
	Registry                 *string                `yaml:"registry,omitempty"`
	WorkerNodes              *uint16                `yaml:"workerNodes,omitempty"`
	ControlPlaneNodes        *uint16                `yaml:"controlPlaneNodes,omitempty"`
	KubeAPIServerBindPort    *uint16                `yaml:"kubeApiServerBindPort,omitempty"`
	KubeAPIServerBindPortAlt *uint16                `yaml:"kubeApiServerBindPortAlt,omitempty"`
	VirtualIp                *string                `yaml:"virtualIp,omitempty"`
	LoadBalancer             *string                `yaml:"loadBalancer,omitempty"`
	PodSubnet                *string                `yaml:"podSubnet,omitempty"`
	ServiceSubnet            *string                `yaml:"serviceSubnet,omitempty"`
	CertificateInformation   CertificateInformation `yaml:"certificateInformation"`
	OsTag                    *string                `yaml:"osTag,omitempty"`
	OsRegistry               *string                `yaml:"osRegistry,omitempty"`
	KubeProxyMode            *string                `yaml:"kubeProxyMode,omitempty"`
	BootVolumeContainerImage *string                `yaml:"bootVolumeContainerImage,omitempty"`
	CNI                      *string                `yaml:"cni,omitempty"`
	Headless                 *bool                  `yaml:"headless,omitempty"`
	Catalog                  *bool                  `yaml:"catalog,omitempty"`
	CommunityCatalog         *bool                  `yaml:"communityCatalog,omitempty"`
	Catalogs                 []Catalog              `yaml:"catalogs"`
	Applications             []Application          `yaml:"applications"`
	EphemeralConfig          EphemeralClusterConfig `yaml:"ephemeralCluster"`
	KubeVersion              *string                `yaml:"kubernetesVersion,omitempty"`
	SshPublicKeyPath         *string                `yaml:"sshPublicKeyPath,omitempty"`
	SshPublicKey             *string                `yaml:"sshPublicKey,omitempty"`
	Password                 *string                `yaml:"password,omitempty"`
	CipherSuites             *string                `yaml:"cipherSuites,omitempty"`
	ClusterDefinitionInline  *string                `yaml:"clusterDefinitionInline,omitempty"`
	ClusterDefinition        *string                `yaml:"clusterDefinition,omitempty"`
	ExtraIgnitionInline      *string                `yaml:"extraIgnitionInline,omitempty"`
	ExtraIgnition            *string                `yaml:"extraIgnition,omitempty"`
	Quiet                    *bool                  `yaml:"quiet,omitempty"`
	KubeConfig               *string                `yaml:"kubeconfig,omitempty"`
	AutoStartUI              *string                `yaml:"autoStartUI,omitempty"`
}

type ImageInfo struct {
	BaseImage string
	Tag       string
	Digest    string
}

// ieup is short for "If Else Uint pointer". If the second argument is
// non-zero, it is returned. Otherwise, the first argument
// is returned. If both arguments are nil, a pointer to the value of 0 is returned.
func ieup(i *uint16, e *uint16) *uint16 {
	returnVal := uint16(0)
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// ieupn is short for "If Else Uint pointer nil". If one
// // the second value is not nil, a pointer to a copy of its value
// // is returned.  If the first argument is not nil but the second
// // one is, then a pointer to a copy of the first argument is
// // returned.  If both are nil, nil is returned.
func ieupn(i *uint16, e *uint16) *uint16 {
	returnVal := uint16(0)
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// ieip is short for "If Else Int Pointer". If one of the values
// // is non-nil but the other is nil, the non-nil one is returned.
// is returned. Otherwise, the value of the second argument is returned.
// If both are nil, zero is returned
func ieip(i *int, e *int) *int {
	returnVal := int(0)
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// ieipn is short for "If Else Int Pointer nil". If one
// // the second value is not nil, a pointer to a copy of its value
// // is returned.  If the first argument is not nil but the second
// // one is, then a pointer to a copy of the first argument is
// // returned.  If both are nil, nil is returned.
func ieipn(i *int, e *int) *int {
	returnVal := int(0)
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// iebp is short for "If Else Bool Pointer". If one of the values
// is non-nil but the other is nil, the non-nil one is returned.
// Otherwise, the value of the default argument is returned.
func iebp(i *bool, e *bool, def bool) *bool {
	returnVal := def
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// iebpn is short for "If Else Bool Pointer". If one
// // the second value is not nil, a pointer to a copy of its value
// // is returned.  If the first argument is not nil but the second
// // one is, then a pointer to a copy of the first argument is
// // returned.  If both are nil, nil is returned.
func iebpn(i *bool, e *bool) *bool {
	if e != nil {
		ret := *e
		return &ret
	} else if i != nil {
		ret := *i
		return &ret
	}
	return nil
}

// iesp is short for "If Else String Pointer".  If one of the values
// is non-nil but the other is nil, a separate pointer pointing to the same value is returned.
// Otherwise, a separate pointer that points to the value of the second argument is returned.
func iesp(i *string, e *string) *string {
	returnVal := ""
	if e != nil {
		returnVal = *e
	} else if i != nil {
		returnVal = *i
	}
	return &returnVal
}

// iespn is short for "If Else String Pointer Nil".  If one
// // the second value is not nil, a pointer to a copy of its value
// // is returned.  If the first argument is not nil but the second
// // one is, then a pointer to a copy of the first argument is
// // returned.  If both are nil, nil is returned.
func iespn(i *string, e *string) *string {
	if e != nil {
		ret := *e
		return &ret
	} else if i != nil {
		ret := *i
		return &ret
	}
	return nil
}

// iebpp is short for "If Else Bool Pointer Pointer".  If one
// the second value is not nil, a pointer to a copy of its value
// is returned.  If the first argument is not nil but the second
// one is, then a pointer to a copy of the first argument is
// returned.  If both are nil, nil is returned.
func iebpp(i *bool, e *bool) *bool {
	if e != nil {
		ret := *e
		return &ret
	} else if i != nil {
		ret := *i
		return &ret
	}
	return nil
}

// MergeProxy takes two Proxies and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeProxy(def *Proxy, ovr *Proxy) Proxy {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return Proxy{
		HttpsProxy: iesp(def.HttpsProxy, ovr.HttpsProxy),
		HttpProxy:  iesp(def.HttpProxy, ovr.HttpProxy),
		NoProxy:    iesp(def.NoProxy, ovr.NoProxy),
	}
}

// MergeProxyPointer takes two Proxies and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeProxyPointer(def *Proxy, ovr *Proxy) Proxy {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return Proxy{
		HttpsProxy: iespn(def.HttpsProxy, ovr.HttpsProxy),
		HttpProxy:  iespn(def.HttpProxy, ovr.HttpProxy),
		NoProxy:    iespn(def.NoProxy, ovr.NoProxy),
	}
}

// MergeCatalogs takes two Catalogs and merges them into a third.
// The two lists are appended to one another, and a unique slice
// is returned
// // For each catalog in the slice, its  *string values are set to the empty string, if that value does not exist
func MergeCatalogs(def []Catalog, ovr []Catalog) []Catalog {
	catalogToReturn := append(append([]Catalog{}, def...), ovr...)
	for i, catalog := range catalogToReturn {
		if catalog.Name == nil {
			catalog.Name = iesp(catalog.Name, nil)
		}
		if catalog.URI == nil {
			catalog.URI = iesp(catalog.URI, nil)
		}
		if catalog.Namespace == nil {
			catalog.Namespace = iesp(catalog.Namespace, nil)
		}
		if catalog.Protocol == nil {
			catalog.Protocol = iesp(catalog.Protocol, nil)
		}
		catalogToReturn[i] = catalog
	}
	return catalogToReturn
}

// MergeCatalogsIncludeNil takes two Catalogs and merges them into
// a third.  The two lists are appended to one another, and a unique
// slice is returned
func MergeCatalogsIncludeNil(def []Catalog, ovr []Catalog) []Catalog {
	return append(append([]Catalog{}, def...), ovr...)
}

// MergeApplications takes two Applications and merges them into
// a third.  The two lists are appended to one another, and a unique
// slice is returned
// For each application in the slice, its  *string values are set to the empty string, if that value does not exist
func MergeApplications(def []Application, ovr []Application) []Application {
	applicationsToReturn := append(append([]Application{}, def...), ovr...)
	for i, app := range applicationsToReturn {
		if app.ConfigFrom == nil {
			app.ConfigFrom = iesp(app.ConfigFrom, nil)
		}
		if app.Name == nil {
			app.Name = iesp(app.Name, nil)
		}
		if app.Namespace == nil {
			app.Namespace = iesp(app.Namespace, nil)
		}
		if app.Catalog == nil {
			app.Catalog = iesp(app.Catalog, nil)
		}
		if app.Version == nil {
			app.Version = iesp(app.Version, nil)
		}
		if app.Release == nil {
			app.Release = iesp(app.Release, nil)
		}
		applicationsToReturn[i] = app
	}
	return applicationsToReturn
}

// MergeApplicationsIncludeNil takes two Applications and merges them into
// a third.  The two lists are appended to one another, and a unique
// slice is returned
func MergeApplicationsIncludeNil(def []Application, ovr []Application) []Application {
	return append(append([]Application{}, def...), ovr...)
}

// MergeCertificateInformation takes two CertificateInformations and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeCertificateInformation(def *CertificateInformation, ovr *CertificateInformation) CertificateInformation {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return CertificateInformation{
		Country: iesp(def.Country, ovr.Country),
		Org:     iesp(def.Org, ovr.Org),
		OrgUnit: iesp(def.OrgUnit, ovr.OrgUnit),
		State:   iesp(def.State, ovr.State),
	}
}

// MergeCertificateInformationPointer takes two CertificateInformations and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeCertificateInformationPointer(def *CertificateInformation, ovr *CertificateInformation) CertificateInformation {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return CertificateInformation{
		Country: iespn(def.Country, ovr.Country),
		Org:     iespn(def.Org, ovr.Org),
		OrgUnit: iespn(def.OrgUnit, ovr.OrgUnit),
		State:   iespn(def.State, ovr.State),
	}
}

// MergeNode takes two Nodes and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeNode(def *Node, ovr *Node) Node {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return Node{
		Memory:  iesp(def.Memory, ovr.Memory),
		Storage: iesp(def.Storage, ovr.Storage),
		CPUs:    ieip(def.CPUs, ovr.CPUs),
	}
}

// MergeNodePointer takes two Nodes and merges them into a third.
// The default values for the result come from the first argument.
// If a value is set in the second argument, that value takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeNodePointer(def *Node, ovr *Node) Node {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return Node{
		Memory:  iespn(def.Memory, ovr.Memory),
		Storage: iespn(def.Storage, ovr.Storage),
		CPUs:    ieipn(def.CPUs, ovr.CPUs),
	}
}

// MergeLibvirtProvider takes two LibvirtProviders and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeLibvirtProvider(def *LibvirtProvider, ovr *LibvirtProvider) LibvirtProvider {
	// It is currently safe to shallow copy this because all the
	// values are scalars.  If that changes, a deep copy will have
	// to be performed instead.
	if ovr == nil {
		return *def
	}
	return LibvirtProvider{
		SessionURI:                   iesp(def.SessionURI, ovr.SessionURI),
		SshKey:                       iesp(def.SshKey, ovr.SshKey),
		StoragePool:                  iesp(def.StoragePool, ovr.StoragePool),
		Network:                      iesp(def.Network, ovr.Network),
		ControlPlaneNode:             MergeNode(&def.ControlPlaneNode, &ovr.ControlPlaneNode),
		WorkerNode:                   MergeNode(&def.WorkerNode, &ovr.WorkerNode),
		BootVolumeName:               iesp(def.BootVolumeName, ovr.BootVolumeName),
		BootVolumeContainerImagePath: iesp(def.BootVolumeContainerImagePath, ovr.BootVolumeContainerImagePath),
	}
}

// MergeLibvirtProviderPointer takes two LibvirtProviders and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeLibvirtProviderPointer(def *LibvirtProvider, ovr *LibvirtProvider) LibvirtProvider {
	// It is currently safe to shallow copy this because all the
	// values are scalars.  If that changes, a deep copy will have
	// to be performed instead.
	if ovr == nil {
		return *def
	}
	return LibvirtProvider{
		SessionURI:                   iespn(def.SessionURI, ovr.SessionURI),
		SshKey:                       iespn(def.SshKey, ovr.SshKey),
		StoragePool:                  iespn(def.StoragePool, ovr.StoragePool),
		Network:                      iespn(def.Network, ovr.Network),
		ControlPlaneNode:             MergeNodePointer(&def.ControlPlaneNode, &ovr.ControlPlaneNode),
		WorkerNode:                   MergeNodePointer(&def.WorkerNode, &ovr.WorkerNode),
		BootVolumeName:               iespn(def.BootVolumeName, ovr.BootVolumeName),
		BootVolumeContainerImagePath: iespn(def.BootVolumeContainerImagePath, ovr.BootVolumeContainerImagePath),
	}
}

// MergeOciInstanceShape takes two OciInstanceShapes and merges them into
// a third.  The default values from the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precendence.
func MergeOciInstanceShape(def *OciInstanceShape, ovr *OciInstanceShape) OciInstanceShape {
	return OciInstanceShape{
		Shape: iesp(def.Shape, ovr.Shape),
		Ocpus: ieip(def.Ocpus, ovr.Ocpus),
	}
}

// MergeOciInstanceShapePointer takes two OciInstanceShapes and merges them into
// a third.  The default values from the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precendence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOciInstanceShapePointer(def *OciInstanceShape, ovr *OciInstanceShape) OciInstanceShape {
	return OciInstanceShape{
		Shape: iespn(def.Shape, ovr.Shape),
		Ocpus: ieipn(def.Ocpus, ovr.Ocpus),
	}
}

// MergeLoadBalancer takes two LoadBalancers and merges them into
// a third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeLoadBalancer(def *LoadBalancer, ovr *LoadBalancer) LoadBalancer {
	return LoadBalancer{
		Subnet1: iesp(def.Subnet1, ovr.Subnet1),
		Subnet2: iesp(def.Subnet2, ovr.Subnet2),
	}
}

// MergeLoadBalancerPointer takes two LoadBalancers and merges them into
// a third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeLoadBalancerPointer(def *LoadBalancer, ovr *LoadBalancer) LoadBalancer {
	return LoadBalancer{
		Subnet1: iespn(def.Subnet1, ovr.Subnet1),
		Subnet2: iespn(def.Subnet2, ovr.Subnet2),
	}
}

// MergeOciImageSet takes two OciImageSets and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOciImageSet(def *OciImageSet, ovr *OciImageSet) OciImageSet {
	return OciImageSet{
		Amd64: iesp(def.Amd64, ovr.Amd64),
		Arm64: iesp(def.Arm64, ovr.Arm64),
	}
}

// MergeOciImageSetPointer takes two OciImageSets and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOciImageSetPointer(def *OciImageSet, ovr *OciImageSet) OciImageSet {
	return OciImageSet{
		Amd64: iespn(def.Amd64, ovr.Amd64),
		Arm64: iespn(def.Arm64, ovr.Arm64),
	}
}

// MergeOciProvider takes two OciProviders and merges
// them into a third.  The default values for the result come from
// the first argument.  If a value is set in the second argument, that
// values takes precedence.
func MergeOciProvider(def *OciProvider, ovr *OciProvider) OciProvider {
	return OciProvider{
		KubeConfigPath:    iesp(def.KubeConfigPath, ovr.KubeConfigPath),
		Compartment:       iesp(def.Compartment, ovr.Compartment),
		Namespace:         iesp(def.Namespace, ovr.Namespace),
		Images:            MergeOciImageSet(&def.Images, &ovr.Images),
		ImageBucket:       iesp(def.ImageBucket, ovr.ImageBucket),
		ControlPlaneShape: MergeOciInstanceShape(&def.ControlPlaneShape, &ovr.ControlPlaneShape),
		WorkerShape:       MergeOciInstanceShape(&def.WorkerShape, &ovr.WorkerShape),
		LoadBalancer:      MergeLoadBalancer(&def.LoadBalancer, &ovr.LoadBalancer),
		SelfManaged:       iebp(def.SelfManaged, ovr.SelfManaged, false),
		Vcn:               iesp(def.Vcn, ovr.Vcn),
		Proxy:             MergeProxy(&def.Proxy, &ovr.Proxy),
	}
}

// MergeOciProviderPointer takes two OciProviders and merges
// them into a third.  The default values for the result come from
// the first argument.  If a value is set in the second argument, that
// values takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOciProviderPointer(def *OciProvider, ovr *OciProvider) OciProvider {
	return OciProvider{
		KubeConfigPath:    iespn(def.KubeConfigPath, ovr.KubeConfigPath),
		Compartment:       iespn(def.Compartment, ovr.Compartment),
		Namespace:         iespn(def.Namespace, ovr.Namespace),
		Images:            MergeOciImageSetPointer(&def.Images, &ovr.Images),
		ImageBucket:       iespn(def.ImageBucket, ovr.ImageBucket),
		ControlPlaneShape: MergeOciInstanceShapePointer(&def.ControlPlaneShape, &ovr.ControlPlaneShape),
		WorkerShape:       MergeOciInstanceShapePointer(&def.WorkerShape, &ovr.WorkerShape),
		LoadBalancer:      MergeLoadBalancerPointer(&def.LoadBalancer, &ovr.LoadBalancer),
		SelfManaged:       iebpn(def.SelfManaged, ovr.SelfManaged),
		Vcn:               iespn(def.Vcn, ovr.Vcn),
		Proxy:             MergeProxyPointer(&def.Proxy, &ovr.Proxy),
	}
}

// MergeOlvmProvider takes two OlvmProviders and merges
// them into a third.  The default values for the result come from
// the first argument.  If a value is set in the second argument, that
// values takes precedence.
func MergeOlvmProvider(def *OlvmProvider, ovr *OlvmProvider) OlvmProvider {
	return OlvmProvider{
		Namespace:           iesp(def.Namespace, ovr.Namespace),
		SelfManaged:         iebp(def.SelfManaged, ovr.SelfManaged, false),
		Proxy:               MergeProxy(&def.Proxy, &ovr.Proxy),
		NetworkInterface:    iesp(def.NetworkInterface, ovr.NetworkInterface),
		OlvmCluster:         MergeOlvmCluster(&def.OlvmCluster, &ovr.OlvmCluster),
		ControlPlaneMachine: MergeOlvmMachine(&def.ControlPlaneMachine, &ovr.ControlPlaneMachine),
		WorkerMachine:       MergeOlvmMachine(&def.WorkerMachine, &ovr.WorkerMachine),
		LocalAPIEndpoint:    MergeOlvmLocalAPIEndpoint(&def.LocalAPIEndpoint, &ovr.LocalAPIEndpoint),
	}
}

// MergeOlvmProviderPointer takes two OlvmProviders and merges
// them into a third.  The default values for the result come from
// the first argument.  If a value is set in the second argument, that
// values takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmProviderPointer(def *OlvmProvider, ovr *OlvmProvider) OlvmProvider {
	return OlvmProvider{
		Namespace:           iespn(def.Namespace, ovr.Namespace),
		SelfManaged:         iebpn(def.SelfManaged, ovr.SelfManaged),
		Proxy:               MergeProxyPointer(&def.Proxy, &ovr.Proxy),
		NetworkInterface:    iespn(def.NetworkInterface, ovr.NetworkInterface),
		OlvmCluster:         MergeOlvmClusterPointer(&def.OlvmCluster, &ovr.OlvmCluster),
		ControlPlaneMachine: MergeOlvmMachinePointer(&def.ControlPlaneMachine, &ovr.ControlPlaneMachine),
		WorkerMachine:       MergeOlvmMachinePointer(&def.WorkerMachine, &ovr.WorkerMachine),
		LocalAPIEndpoint:    MergeOlvmLocalAPIEndpointPointer(&def.LocalAPIEndpoint, &ovr.LocalAPIEndpoint),
	}
}

// MergeOlvmCluster takes two OlvmClusters and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmCluster(def *OlvmCluster, ovr *OlvmCluster) OlvmCluster {
	return OlvmCluster{
		ControlPlaneEndpoint: MergeOlvmControlPlaneEndpoint(&def.ControlPlaneEndpoint, &ovr.ControlPlaneEndpoint),
		DatacenterName:       iesp(def.DatacenterName, ovr.DatacenterName),
		OVirtAPI:             MergeOlvmOvirtAPI(&def.OVirtAPI, &ovr.OVirtAPI),
		OVirtOck:             MergeOlvmOvirtOck(&def.OVirtOck, &ovr.OVirtOck),
		OlvmVmIpProfile:      MergeOlvmVmIpProfile(&def.OlvmVmIpProfile, &ovr.OlvmVmIpProfile),
	}
}

// MergeOlvmClusterPointer takes two OlvmClusters and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmClusterPointer(def *OlvmCluster, ovr *OlvmCluster) OlvmCluster {
	return OlvmCluster{
		ControlPlaneEndpoint: MergeOlvmControlPlaneEndpointPointer(&def.ControlPlaneEndpoint, &ovr.ControlPlaneEndpoint),
		DatacenterName:       iespn(def.DatacenterName, ovr.DatacenterName),
		OVirtAPI:             MergeOlvmOvirtAPIPointer(&def.OVirtAPI, &ovr.OVirtAPI),
		OVirtOck:             MergeOlvmOvirtOckPointer(&def.OVirtOck, &ovr.OVirtOck),
		OlvmVmIpProfile:      MergeOlvmVmIpProfilePointer(&def.OlvmVmIpProfile, &ovr.OlvmVmIpProfile),
	}
}

// MergeOlvmVmIpProfile takes two OlvmVmIpProfiles and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmVmIpProfile(def *OlvmVmIpProfile, ovr *OlvmVmIpProfile) OlvmVmIpProfile {
	return OlvmVmIpProfile{
		Name:              iesp(def.Name, ovr.Name),
		StartingIpAddress: iesp(def.StartingIpAddress, ovr.StartingIpAddress),
		Device:            iesp(def.Device, ovr.Device),
		Gateway:           iesp(def.Gateway, ovr.Gateway),
		Netmask:           iesp(def.Netmask, ovr.Netmask),
	}
}

// MergeOlvmVmIpProfilePointer takes two OlvmVmIpProfiles and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmVmIpProfilePointer(def *OlvmVmIpProfile, ovr *OlvmVmIpProfile) OlvmVmIpProfile {
	return OlvmVmIpProfile{
		Name:              iespn(def.Name, ovr.Name),
		StartingIpAddress: iespn(def.StartingIpAddress, ovr.StartingIpAddress),
		Device:            iespn(def.Device, ovr.Device),
		Gateway:           iespn(def.Gateway, ovr.Gateway),
		Netmask:           iespn(def.Netmask, ovr.Netmask),
	}
}

// MergeOlvmControlPlaneEndpoint takes two OlvmControlPlaneEndpoints and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmControlPlaneEndpoint(def *OlvmControlPlaneEndpoint, ovr *OlvmControlPlaneEndpoint) OlvmControlPlaneEndpoint {
	return OlvmControlPlaneEndpoint{
		Host: iesp(def.Host, ovr.Host),
		Port: iesp(def.Port, ovr.Port),
	}
}

// MergeOlvmControlPlaneEndpointPointer takes two OlvmControlPlaneEndpoints and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmControlPlaneEndpointPointer(def *OlvmControlPlaneEndpoint, ovr *OlvmControlPlaneEndpoint) OlvmControlPlaneEndpoint {
	return OlvmControlPlaneEndpoint{
		Host: iespn(def.Host, ovr.Host),
		Port: iespn(def.Port, ovr.Port),
	}
}

// MergeOlvmOvirtAPI takes two OlvmOvirtAPIs and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmOvirtAPI(def *OlvmOvirtAPI, ovr *OlvmOvirtAPI) OlvmOvirtAPI {
	return OlvmOvirtAPI{
		ServerURL:    iesp(def.ServerURL, ovr.ServerURL),
		ServerCA:     iesp(def.ServerCA, ovr.ServerCA),
		ServerCAPath: iesp(def.ServerCAPath, ovr.ServerCAPath),
	}
}

// MergeOlvmOvirtAPIPointer takes two OlvmOvirtAPIs and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmOvirtAPIPointer(def *OlvmOvirtAPI, ovr *OlvmOvirtAPI) OlvmOvirtAPI {
	return OlvmOvirtAPI{
		ServerURL:    iespn(def.ServerURL, ovr.ServerURL),
		ServerCA:     iespn(def.ServerCA, ovr.ServerCA),
		ServerCAPath: iespn(def.ServerCAPath, ovr.ServerCAPath),
	}
}

// MergeOlvmOvirtOck takes two OlvmOvirtOcks and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmOvirtOck(def *OlvmOvirtOck, ovr *OlvmOvirtOck) OlvmOvirtOck {
	return OlvmOvirtOck{
		DiskName:          iesp(def.DiskName, ovr.DiskName),
		DiskSize:          iesp(def.DiskSize, ovr.DiskSize),
		StorageDomainName: iesp(def.StorageDomainName, ovr.StorageDomainName),
	}
}

// MergeOlvmOvirtOckPointer takes two OlvmOvirtOcks and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmOvirtOckPointer(def *OlvmOvirtOck, ovr *OlvmOvirtOck) OlvmOvirtOck {
	return OlvmOvirtOck{
		DiskName:          iespn(def.DiskName, ovr.DiskName),
		DiskSize:          iespn(def.DiskSize, ovr.DiskSize),
		StorageDomainName: iespn(def.StorageDomainName, ovr.StorageDomainName),
	}
}

// MergeOlvmMachine takes two OlvmMachines and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmMachine(def *OlvmMachine, ovr *OlvmMachine) OlvmMachine {
	return OlvmMachine{
		OVirtClusterName:    iesp(def.OVirtClusterName, ovr.OVirtClusterName),
		OlvmVmIpProfileName: iesp(def.OlvmVmIpProfileName, ovr.OlvmVmIpProfileName),
		Memory:              iesp(def.Memory, ovr.Memory),
		Network:             MergeOlvmMachineNetwork(&def.Network, &ovr.Network),
		Cpu:                 MergeOlvmMachineCpu(&def.Cpu, &ovr.Cpu),
		VMTemplateName:      iesp(def.VMTemplateName, ovr.VMTemplateName),
	}
}

// MergeOlvmMachinePointer takes two OlvmMachines and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmMachinePointer(def *OlvmMachine, ovr *OlvmMachine) OlvmMachine {
	return OlvmMachine{
		OVirtClusterName:    iespn(def.OVirtClusterName, ovr.OVirtClusterName),
		OlvmVmIpProfileName: iespn(def.OlvmVmIpProfileName, ovr.OlvmVmIpProfileName),
		Memory:              iespn(def.Memory, ovr.Memory),
		Network:             MergeOlvmMachineNetworkPointer(&def.Network, &ovr.Network),
		Cpu:                 MergeOlvmMachineCpuPointer(&def.Cpu, &ovr.Cpu),
		VMTemplateName:      iespn(def.VMTemplateName, ovr.VMTemplateName),
	}
}

// MergeOlvmMachineNetwork takes two OlvmMachineNetworks and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmMachineNetwork(def *OlvmMachineNetwork, ovr *OlvmMachineNetwork) OlvmMachineNetwork {
	return OlvmMachineNetwork{
		NetworkName:     iesp(def.NetworkName, ovr.NetworkName),
		InterfaceType:   iesp(def.InterfaceType, ovr.InterfaceType),
		VnicName:        iesp(def.VnicName, ovr.VnicName),
		VnicProfileName: iesp(def.VnicProfileName, ovr.VnicProfileName),
	}
}

// MergeOlvmMachineNetworkPointer takes two OlvmMachineNetworks and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmMachineNetworkPointer(def *OlvmMachineNetwork, ovr *OlvmMachineNetwork) OlvmMachineNetwork {
	return OlvmMachineNetwork{
		NetworkName:     iespn(def.NetworkName, ovr.NetworkName),
		InterfaceType:   iespn(def.InterfaceType, ovr.InterfaceType),
		VnicName:        iespn(def.VnicName, ovr.VnicName),
		VnicProfileName: iespn(def.VnicProfileName, ovr.VnicProfileName),
	}
}

// MergeOlvmMachineCpu takes two OlvmMachineCpus and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmMachineCpu(def *OlvmMachineCpu, ovr *OlvmMachineCpu) OlvmMachineCpu {
	return OlvmMachineCpu{
		Architecture: iesp(def.Architecture, ovr.Architecture),
		Topology:     MergeOlvmMachineCpuToplogy(&def.Topology, &ovr.Topology),
	}
}

// MergeOlvmMachineCpuPointer takes two OlvmMachineCpus and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmMachineCpuPointer(def *OlvmMachineCpu, ovr *OlvmMachineCpu) OlvmMachineCpu {
	return OlvmMachineCpu{
		Architecture: iespn(def.Architecture, ovr.Architecture),
		Topology:     MergeOlvmMachineCpuToplogyPointer(&def.Topology, &ovr.Topology),
	}
}

// MergeOlvmMachineCpuToplogy takes two OlvmMachineCpuToplogies and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmMachineCpuToplogy(def *OlvmMachineCpuToplogy, ovr *OlvmMachineCpuToplogy) OlvmMachineCpuToplogy {
	return OlvmMachineCpuToplogy{
		Cores:   ieip(def.Cores, ovr.Cores),
		Sockets: ieip(def.Sockets, ovr.Sockets),
		Threads: ieip(def.Threads, ovr.Threads),
	}
}

// MergeOlvmMachineCpuToplogyPointer takes two OlvmMachineCpuToplogies and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeOlvmMachineCpuToplogyPointer(def *OlvmMachineCpuToplogy, ovr *OlvmMachineCpuToplogy) OlvmMachineCpuToplogy {
	return OlvmMachineCpuToplogy{
		Cores:   ieipn(def.Cores, ovr.Cores),
		Sockets: ieipn(def.Sockets, ovr.Sockets),
		Threads: ieipn(def.Threads, ovr.Threads),
	}
}

// MergeOlvmLocalAPIEndpoint takes two OlvmLocalAPIEndpoints and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmLocalAPIEndpoint(def *OlvmLocalAPIEndpoint, ovr *OlvmLocalAPIEndpoint) OlvmLocalAPIEndpoint {
	return OlvmLocalAPIEndpoint{
		BindPort:         ieip(def.BindPort, ovr.BindPort),
		AdvertiseAddress: iesp(def.AdvertiseAddress, ovr.AdvertiseAddress),
	}
}

// MergeOlvmLocalAPIEndpointPointer takes two OlvmLocalAPIEndpoints and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOlvmLocalAPIEndpointPointer(def *OlvmLocalAPIEndpoint, ovr *OlvmLocalAPIEndpoint) OlvmLocalAPIEndpoint {
	return OlvmLocalAPIEndpoint{
		BindPort:         ieipn(def.BindPort, ovr.BindPort),
		AdvertiseAddress: iespn(def.AdvertiseAddress, ovr.AdvertiseAddress),
	}
}

// MergeByoProvider takes two ByoProviders and merged them into a
// third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeByoProvider(def *ByoProvider, ovr *ByoProvider) ByoProvider {
	return ByoProvider{
		AutomaticTokenCreation: iebp(def.AutomaticTokenCreation, ovr.AutomaticTokenCreation, false),
		NetworkInterface:       iesp(def.NetworkInterface, ovr.NetworkInterface),
	}
}

// MergeByoProviderPointer takes two ByoProviders and merged them into a
// third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeByoProviderPointer(def *ByoProvider, ovr *ByoProvider) ByoProvider {
	return ByoProvider{
		AutomaticTokenCreation: iebpn(def.AutomaticTokenCreation, ovr.AutomaticTokenCreation),
		NetworkInterface:       iespn(def.NetworkInterface, ovr.NetworkInterface),
	}
}

// MergeProviders takes two Providers and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeProviders(def *Providers, ovr *Providers) Providers {
	return Providers{
		Libvirt: MergeLibvirtProvider(&def.Libvirt, &ovr.Libvirt),
		Oci:     MergeOciProvider(&def.Oci, &ovr.Oci),
		Byo:     MergeByoProvider(&def.Byo, &ovr.Byo),
		Olvm:    MergeOlvmProvider(&def.Olvm, &ovr.Olvm),
	}
}

// MergeProvidersPointer takes two Providers and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
// If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeProvidersPointer(def *Providers, ovr *Providers) Providers {
	return Providers{
		Libvirt: MergeLibvirtProviderPointer(&def.Libvirt, &ovr.Libvirt),
		Oci:     MergeOciProviderPointer(&def.Oci, &ovr.Oci),
		Byo:     MergeByoProviderPointer(&def.Byo, &ovr.Byo),
		Olvm:    MergeOlvmProviderPointer(&def.Olvm, &ovr.Olvm),
	}
}

// MergeEphemeralConfig takes two EphemeralClusterConfigs and merges them
// into a third.  If a value is set in the second argument, that value
// takes precedence.  "Preserve" is ignored so as not to accidentally
// delete something.
func MergeEphemeralConfig(def *EphemeralClusterConfig, ovr *EphemeralClusterConfig) EphemeralClusterConfig {
	return EphemeralClusterConfig{
		Name:     iesp(def.Name, ovr.Name),
		Preserve: iebp(def.Preserve, ovr.Preserve, constants.EphemeralClusterPreserve),
		Node:     MergeNode(&def.Node, &ovr.Node),
	}
}

// MergeEphemeralConfigPointer takes two EphemeralClusterConfigs and merges them
// into a third.  If a value is set in the second argument, that value
// takes precedence. If a pointer to a scalar value (int, uint, string, bool) is nil in both configs, then nil is returned.
func MergeEphemeralConfigPointer(def *EphemeralClusterConfig, ovr *EphemeralClusterConfig) EphemeralClusterConfig {
	return EphemeralClusterConfig{
		Name:     iespn(def.Name, ovr.Name),
		Preserve: iebpn(def.Preserve, ovr.Preserve),
		Node:     MergeNodePointer(&def.Node, &ovr.Node),
	}
}

// MergeConfig takes two Configs and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
// The resulting struct does not have nil values
func MergeConfig(def *Config, ovr *Config) Config {
	return Config{
		Providers:                MergeProvidersPointer(&def.Providers, &ovr.Providers),
		KubeConfig:               iespn(def.KubeConfig, ovr.KubeConfig),
		AutoStartUI:              iespn(def.AutoStartUI, ovr.AutoStartUI),
		Proxy:                    MergeProxyPointer(&def.Proxy, &ovr.Proxy),
		KubeAPIServerBindPort:    ieupn(def.KubeAPIServerBindPort, ovr.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieupn(def.KubeAPIServerBindPortAlt, ovr.KubeAPIServerBindPortAlt),
		PodSubnet:                iespn(def.PodSubnet, ovr.PodSubnet),
		ServiceSubnet:            iespn(def.ServiceSubnet, ovr.ServiceSubnet),
		Registry:                 iespn(def.Registry, ovr.Registry),
		CertificateInformation:   MergeCertificateInformationPointer(&def.CertificateInformation, &ovr.CertificateInformation),
		OsRegistry:               iespn(def.OsRegistry, ovr.OsRegistry),
		OsTag:                    iespn(def.OsTag, ovr.OsTag),
		KubeProxyMode:            iespn(def.KubeProxyMode, ovr.KubeProxyMode),
		BootVolumeContainerImage: iespn(def.BootVolumeContainerImage, ovr.BootVolumeContainerImage),
		CNI:                      iespn(def.CNI, ovr.CNI),
		Headless:                 iebpn(def.Headless, ovr.Headless),
		Catalog:                  iebpn(def.Catalog, ovr.Catalog),
		CommunityCatalog:         iebpn(def.CommunityCatalog, ovr.CommunityCatalog),
		EphemeralConfig:          MergeEphemeralConfigPointer(&def.EphemeralConfig, &ovr.EphemeralConfig),
		Quiet:                    iebpn(def.Quiet, ovr.Quiet),
		KubeVersion:              iespn(def.KubeVersion, ovr.KubeVersion),
		SshPublicKeyPath:         iespn(def.SshPublicKeyPath, ovr.SshPublicKeyPath),
		SshPublicKey:             iespn(def.SshPublicKey, ovr.SshPublicKey),
		Password:                 iespn(def.Password, ovr.Password),
		ExtraIgnition:            iespn(def.ExtraIgnition, ovr.ExtraIgnition),
		ExtraIgnitionInline:      iespn(def.ExtraIgnitionInline, ovr.ExtraIgnitionInline),
	}
}

// MergeClusterConfig takes two ClusterConfigs and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeClusterConfig(def *ClusterConfig, ovr *ClusterConfig) ClusterConfig {
	return ClusterConfig{
		WorkingDirectory:         iesp(def.WorkingDirectory, ovr.WorkingDirectory),
		Name:                     iesp(def.Name, ovr.Name),
		Provider:                 iesp(def.Provider, ovr.Provider),
		Providers:                MergeProviders(&def.Providers, &ovr.Providers),
		WorkerNodes:              ieup(def.WorkerNodes, ovr.WorkerNodes),
		ControlPlaneNodes:        ieup(def.ControlPlaneNodes, ovr.ControlPlaneNodes),
		EphemeralConfig:          MergeEphemeralConfig(&def.EphemeralConfig, &ovr.EphemeralConfig),
		Proxy:                    MergeProxy(&def.Proxy, &ovr.Proxy),
		KubeAPIServerBindPort:    ieup(def.KubeAPIServerBindPort, ovr.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieup(def.KubeAPIServerBindPortAlt, ovr.KubeAPIServerBindPortAlt),
		VirtualIp:                iesp(def.VirtualIp, ovr.VirtualIp),
		LoadBalancer:             iesp(def.LoadBalancer, ovr.LoadBalancer),
		PodSubnet:                iesp(def.PodSubnet, ovr.PodSubnet),
		ServiceSubnet:            iesp(def.ServiceSubnet, ovr.ServiceSubnet),
		Registry:                 iesp(def.Registry, ovr.Registry),
		CertificateInformation:   MergeCertificateInformation(&def.CertificateInformation, &ovr.CertificateInformation),
		OsTag:                    iesp(def.OsTag, ovr.OsTag),
		OsRegistry:               iesp(def.OsRegistry, ovr.OsRegistry),
		KubeProxyMode:            iesp(def.KubeProxyMode, ovr.KubeProxyMode),
		CNI:                      iesp(def.CNI, ovr.CNI),
		Headless:                 iebp(def.Headless, ovr.Headless, false),
		Catalog:                  iebp(def.Catalog, ovr.Catalog, true),
		CommunityCatalog:         iebp(def.CommunityCatalog, ovr.CommunityCatalog, false),
		BootVolumeContainerImage: iesp(def.BootVolumeContainerImage, ovr.BootVolumeContainerImage),
		Applications:             MergeApplications(def.Applications, ovr.Applications),
		Catalogs:                 MergeCatalogs(def.Catalogs, ovr.Catalogs),
		KubeVersion:              iesp(def.KubeVersion, ovr.KubeVersion),
		SshPublicKeyPath:         iesp(def.SshPublicKeyPath, ovr.SshPublicKeyPath),
		SshPublicKey:             iesp(def.SshPublicKey, ovr.SshPublicKey),
		Password:                 iesp(def.Password, ovr.Password),
		CipherSuites:             iesp(def.CipherSuites, ovr.CipherSuites),
		ClusterDefinitionInline:  iesp(def.ClusterDefinitionInline, ovr.ClusterDefinitionInline),
		ClusterDefinition:        iesp(def.ClusterDefinition, ovr.ClusterDefinition),
		ExtraIgnition:            iesp(def.ExtraIgnition, ovr.ExtraIgnition),
		ExtraIgnitionInline:      iesp(def.ExtraIgnitionInline, ovr.ExtraIgnitionInline),
		Quiet:                    iebp(def.Quiet, ovr.Quiet, false),
		AutoStartUI:              iesp(def.AutoStartUI, ovr.AutoStartUI),
		KubeConfig:               iesp(def.KubeConfig, ovr.KubeConfig),
	}
}

// OverlayConfig merges the values from a Config into a ClusterConfig.  Values
// from the ClusterConfig take precendence.
// Precalculate Kubeversion above the return, feed that value into the right field
func OverlayConfig(cc *ClusterConfig, c *Config) ClusterConfig {
	clusterConfigToReturn := ClusterConfig{
		WorkingDirectory:         iespn(cc.WorkingDirectory, nil),
		Name:                     iespn(cc.Name, nil),
		Provider:                 iespn(cc.Provider, nil),
		WorkerNodes:              ieupn(cc.WorkerNodes, nil),
		ControlPlaneNodes:        ieupn(cc.ControlPlaneNodes, nil),
		Providers:                MergeProvidersPointer(&c.Providers, &cc.Providers),
		Proxy:                    MergeProxyPointer(&c.Proxy, &cc.Proxy),
		EphemeralConfig:          MergeEphemeralConfigPointer(&c.EphemeralConfig, &cc.EphemeralConfig),
		KubeAPIServerBindPort:    ieupn(c.KubeAPIServerBindPort, cc.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieupn(c.KubeAPIServerBindPortAlt, cc.KubeAPIServerBindPortAlt),
		VirtualIp:                iespn(cc.VirtualIp, nil),
		LoadBalancer:             iespn(cc.LoadBalancer, nil),
		PodSubnet:                iespn(c.PodSubnet, cc.PodSubnet),
		ServiceSubnet:            iespn(c.ServiceSubnet, cc.ServiceSubnet),
		Registry:                 iespn(c.Registry, cc.Registry),
		CertificateInformation:   MergeCertificateInformationPointer(&c.CertificateInformation, &cc.CertificateInformation),
		OsRegistry:               iespn(c.OsRegistry, cc.OsRegistry),
		OsTag:                    iespn(c.OsTag, cc.OsTag),
		KubeProxyMode:            iespn(c.KubeProxyMode, cc.KubeProxyMode),
		BootVolumeContainerImage: iespn(c.BootVolumeContainerImage, cc.BootVolumeContainerImage),
		CNI:                      iespn(c.CNI, cc.CNI),
		Headless:                 iebpn(c.Headless, cc.Headless),
		Catalog:                  iebpn(c.Catalog, cc.Catalog),
		CommunityCatalog:         iebpn(c.CommunityCatalog, cc.CommunityCatalog),
		Applications:             MergeApplicationsIncludeNil(cc.Applications, nil),
		Catalogs:                 MergeCatalogsIncludeNil(cc.Catalogs, nil),
		KubeVersion:              iespn(c.KubeVersion, cc.KubeVersion),
		SshPublicKeyPath:         iespn(c.SshPublicKeyPath, cc.SshPublicKeyPath),
		SshPublicKey:             iespn(c.SshPublicKey, cc.SshPublicKey),
		Password:                 iespn(c.Password, cc.Password),
		CipherSuites:             iespn(cc.CipherSuites, nil),
		ClusterDefinitionInline:  iespn(cc.ClusterDefinitionInline, nil),
		ClusterDefinition:        iespn(cc.ClusterDefinition, nil),
		ExtraIgnition:            iespn(c.ExtraIgnition, cc.ExtraIgnition),
		ExtraIgnitionInline:      iespn(c.ExtraIgnitionInline, cc.ExtraIgnitionInline),
		Quiet:                    iebpn(c.Quiet, cc.Quiet),
		AutoStartUI:              iespn(c.AutoStartUI, cc.AutoStartUI),
		KubeConfig:               iespn(c.KubeConfig, cc.KubeConfig),
	}
	return clusterConfigToReturn
}

// CopyClusterConfig returns a deep copy of a ClusterConfig
func CopyClusterConfig(cc *ClusterConfig) ClusterConfig {
	return MergeClusterConfig(&ClusterConfig{}, cc)
}
