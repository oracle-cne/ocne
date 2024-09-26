// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

type LibvirtProvider struct {
	SessionURI                   string `yaml:"uri"`
	SshKey                       string `yaml:"sshKey"`
	StoragePool                  string `yaml:"storagePool"`
	Network                      string `yaml:"network"`
	ControlPlaneNode             Node   `yaml:"controlPlaneNode"`
	WorkerNode                   Node   `yaml:"workerNode"`
	BootVolumeName               string `yaml:"bootVolumeName"`
	BootVolumeContainerImagePath string `yaml:"bootVolumeContainerImagePath"`
}

type OciInstanceShape struct {
	Shape string `yaml:"shape"`
	Ocpus int    `yaml:"ocpus"`
}

type LoadBalancer struct {
	Subnet1 string `yaml:"subnet1"`
	Subnet2 string `yaml:"subnet2"`
}

type OciImageSet struct {
	Amd64 string `yaml:"amd64"`
	Arm64 string `yaml:"arm64"`
}

type OciProvider struct {
	KubeConfigPath    string           `yaml:"kubeconfig"`
	Compartment       string           `yaml:"compartment"`
	Namespace         string           `yaml:"namespace"`
	ControlPlaneShape OciInstanceShape `yaml:"controlPlaneShape"`
	Images            OciImageSet      `yaml:"images"`
	WorkerShape       OciInstanceShape `yaml:"workerShape"`
	SelfManaged       bool             `yaml:"selfmanagedfake"`
	SelfManagedPtr    *bool            `yaml:"selfManaged,omitempty"`
	LoadBalancer      LoadBalancer     `yaml:"loadBalancer"`
	Vcn               string           `yaml:"vcn"`
	ImageBucket       string           `yaml:"imageBucket"`
	Proxy             Proxy            `yaml:"proxy"`
}

type ByoProvider struct {
	AutomaticTokenCreation    bool   `yaml:"automaticTokenCreationfake"`
	AutomaticTokenCreationPtr *bool  `yaml:"automaticTokenCreation"`
	NetworkInterface          string `yaml:"networkInterface"`
}

type Node struct {
	Memory  string `yaml:"memory"`
	CPUs    int    `yaml:"cpu"`
	Storage string `yaml:"storage"`
}

type CertificateInformation struct {
	Country string `yaml:"country"`
	Org     string `yaml:"org"`
	OrgUnit string `yaml:"orgUnit"`
	State   string `yaml:"state"`
}

type Providers struct {
	Libvirt LibvirtProvider `yaml:"libvirt"`
	Oci     OciProvider     `yaml:"oci"`
	Byo     ByoProvider     `yaml:"byo"`
}

type Proxy struct {
	HttpsProxy string `yaml:"httpsProxy"`
	HttpProxy  string `yaml:"httpProxy"`
	NoProxy    string `yaml:"noProxy"`
}

type Catalog struct {
	Protocol  string `yaml:"protocol"`
	URI       string `yaml:"uri"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Application struct {
	Name       string      `yaml:"name"`
	Release    string      `yaml:"release"`
	Version    string      `yaml:"version"`
	Catalog    string      `yaml:"catalog"`
	Namespace  string      `yaml:"namespace"`
	Config     interface{} `yaml:"config"`
	ConfigFrom string      `yaml:"configFrom"`
}

type EphemeralClusterConfig struct {
	Name     string `yaml:"name"`
	Preserve bool   `yaml:"preserve"`
	Node     Node   `yaml:"node"`
}

type Config struct {
	Providers                Providers              `yaml:"providers"`
	KubeConfig               string                 `yaml:"kubeconfig"`
	AutoStartUI              string                 `yaml:"autoStartUI"`
	Proxy                    Proxy                  `yaml:"proxy"`
	KubeAPIServerBindPort    uint16                 `yaml:"kubeApiServerBindPort"`
	KubeAPIServerBindPortAlt uint16                 `yaml:"kubeApiServerBindPortAlt"`
	PodSubnet                string                 `yaml:"podSubnet"`
	ServiceSubnet            string                 `yaml:"serviceSubnet"`
	Registry                 string                 `yaml:"registry"`
	CertificateInformation   CertificateInformation `yaml:"certificateInformation"`
	OsTag                    string                 `yaml:"osTag"`
	OsRegistry               string                 `yaml:"osRegistry"`
	KubeProxyMode            string                 `yaml:"kubeProxyMode"`
	BootVolumeContainerImage string                 `yaml:"bootVolumeContainerImage"`
	CNI                      string                 `yaml:"cni"`
	Headless                 bool                   `yaml:"headlessfake"`
	HeadlessPtr              *bool                  `yaml:"headless,omitempty"`
	Catalog                  bool                   `yaml:"catalogfake"`
	CatalogPtr               *bool                  `yaml:"catalog,omitempty"`
	EphemeralConfig          EphemeralClusterConfig `yaml:"ephemeralCluster"`
	Quiet                    bool                   `yaml:"quiteFake"`
	QuietPtr                 *bool                  `yaml:"quiet,omitempty"`
	KubeVersion              string                 `yaml:"kubernetesVersion"`
	SshPublicKeyPath         string                 `yaml:"sshPublicKeyPath"`
	SshPublicKey             string                 `yaml:"sshPublicKey"`
	Password                 string                 `yaml:"password"`
	CipherSuites             string                 `yaml:"cipherSuites"`
	ExtraIgnitionInline      string                 `yaml:"extraIgnitionInline"`
	ExtraIgnition            string                 `yaml:"extraIgnition"`
}

type ClusterConfig struct {
	WorkingDirectory         string                 `yaml:"directory"`
	Name                     string                 `yaml:"name"`
	Provider                 string                 `yaml:"provider"`
	Providers                Providers              `yaml:"providers"`
	Proxy                    Proxy                  `yaml:"proxy"`
	Registry                 string                 `yaml:"registry"`
	WorkerNodes              uint16                 `yaml:"workerNodes"`
	ControlPlaneNodes        uint16                 `yaml:"controlPlaneNodes"`
	KubeAPIServerBindPort    uint16                 `yaml:"kubeApiServerBindPort"`
	KubeAPIServerBindPortAlt uint16                 `yaml:"kubeApiServerBindPortAlt"`
	VirtualIp                string                 `yaml:"virtualIp"`
	LoadBalancer             string                 `yaml:"loadBalancer"`
	PodSubnet                string                 `yaml:"podSubnet"`
	ServiceSubnet            string                 `yaml:"serviceSubnet"`
	CertificateInformation   CertificateInformation `yaml:"certificateInformation"`
	OsTag                    string                 `yaml:"osTag"`
	OsRegistry               string                 `yaml:"osRegistry"`
	KubeProxyMode            string                 `yaml:"kubeProxyMode"`
	BootVolumeContainerImage string                 `yaml:"bootVolumeContainerImage"`
	CNI                      string                 `yaml:"cni"`
	Headless                 bool                   `yaml:"headlessfake"`
	HeadlessPtr              *bool                  `yaml:"headless,omitempty"`
	Catalog                  bool                   `yaml:"catalogfake"`
	CatalogPtr               *bool                  `yaml:"catalog,omitempty"`
	Catalogs                 []Catalog              `yaml:"catalogs"`
	Applications             []Application          `yaml:"applications"`
	KubeVersion              string                 `yaml:"kubernetesVersion"`
	SshPublicKeyPath         string                 `yaml:"sshPublicKeyPath"`
	SshPublicKey             string                 `yaml:"sshPublicKey"`
	Password                 string                 `yaml:"password"`
	CipherSuites             string                 `yaml:"cipherSuites"`
	ClusterDefinitionInline  string                 `yaml:"clusterDefinitionInline"`
	ClusterDefinition        string                 `yaml:"clusterDefinition"`
	ExtraIgnitionInline      string                 `yaml:"extraIgnitionInline"`
	ExtraIgnition            string                 `yaml:"extraIgnition"`
}

type ImageInfo struct {
	BaseImage string
	Tag       string
	Digest    string
}

// ies is short for "If Else String".  If the second argument is
// non-empty, it is returned.  Otherwise, the first argument
// is returned.
func ies(i string, e string) string {
	if e != "" {
		return e
	}
	return i
}

// ieu is short for "If Else Uint".  If the second argument is
// non-zero, it is returned.  Otherwise, the first argument
// is returned.
func ieu(i uint16, e uint16) uint16 {
	if e != 0 {
		return e
	}
	return i
}

// ieu is short for "If Else Int".  If the second argument is
// non-zero, it is returned.  Otherwise, the first argument
// is returned.
func iei(i int, e int) int {
	if e != 0 {
		return e
	}
	return i
}

// iebp is short for "If Else Bool Pointer".  If one of the values
// is non-nil but the other is nil, the non-nil one is returned.
// Otherwise, the value of the second argument is returned.
func iebp(i *bool, e *bool, def bool) bool {
	if e != nil {
		return *e
	}
	if i != nil {
		return *i
	}
	return def
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
		HttpsProxy: ies(def.HttpsProxy, ovr.HttpsProxy),
		HttpProxy:  ies(def.HttpProxy, ovr.HttpProxy),
		NoProxy:    ies(def.NoProxy, ovr.NoProxy),
	}
}

// MergeCatalogs takes two Catalogs and merges them into a third.
// The two lists are appended to one another, and a unique slice
// is returned
func MergeCatalogs(def []Catalog, ovr []Catalog) []Catalog {
	return append(append([]Catalog{}, def...), ovr...)
}

// MergeApplications takes two Applications and merges them into
// a third.  The two lists are appended to one another, and a unique
// slice is returned
func MergeApplications(def []Application, ovr []Application) []Application {
	return append(append([]Application{}, def...), ovr...)
}

// MergeCertificationInformation takes two CertificateInformations and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeCertificateInformation(def *CertificateInformation, ovr *CertificateInformation) CertificateInformation {
	// This is safe to shallow copy because all values are scalars
	if ovr == nil {
		return *def
	}

	return CertificateInformation{
		Country: ies(def.Country, ovr.Country),
		Org:     ies(def.Org, ovr.Org),
		OrgUnit: ies(def.OrgUnit, ovr.OrgUnit),
		State:   ies(def.State, ovr.State),
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
		Memory:  ies(def.Memory, ovr.Memory),
		Storage: ies(def.Storage, ovr.Storage),
		CPUs:    iei(def.CPUs, ovr.CPUs),
	}
}

// MergeLibvirtProvider takes two LibvirtProviders and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeLibvirtProvider(def *LibvirtProvider, ovr *LibvirtProvider) LibvirtProvider {
	// It is currently safe to shallow copy this because all the
	// values are scalars.  If that changes, a deep copy will have
	// to be performed instead.
	if ovr == nil {
		return *def
	}
	return LibvirtProvider{
		SessionURI:                   ies(def.SessionURI, ovr.SessionURI),
		SshKey:                       ies(def.SshKey, ovr.SshKey),
		StoragePool:                  ies(def.StoragePool, ovr.StoragePool),
		Network:                      ies(def.Network, ovr.Network),
		ControlPlaneNode:             MergeNode(&def.ControlPlaneNode, &ovr.ControlPlaneNode),
		WorkerNode:                   MergeNode(&def.WorkerNode, &ovr.WorkerNode),
		BootVolumeName:               ies(def.BootVolumeName, ovr.BootVolumeName),
		BootVolumeContainerImagePath: ies(def.BootVolumeContainerImagePath, ovr.BootVolumeContainerImagePath),
	}
}

// MergeOciInstanceShape takes two OciInstanceShapes and merges them into
// a third.  The default values from the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precendence.
func MergeOciInstanceShape(def *OciInstanceShape, ovr *OciInstanceShape) OciInstanceShape {
	return OciInstanceShape{
		Shape: ies(def.Shape, ovr.Shape),
		Ocpus: iei(def.Ocpus, ovr.Ocpus),
	}
}

// MergeLoadBalancer takes two LoadBalancers and merges them into
// a third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeLoadBalancer(def *LoadBalancer, ovr *LoadBalancer) LoadBalancer {
	return LoadBalancer{
		Subnet1: ies(def.Subnet1, ovr.Subnet1),
		Subnet2: ies(def.Subnet2, ovr.Subnet2),
	}
}

// MergeOciImageSet takes two OciImageSets and merges them into
// a third.  The default value for the result comes from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeOciImageSet(def *OciImageSet, ovr *OciImageSet) OciImageSet {
	return OciImageSet{
		Amd64: ies(def.Amd64, ovr.Amd64),
		Arm64: ies(def.Arm64, ovr.Arm64),
	}
}

// MergeOciProvider takes two OciProviders and merges
// them into a third.  The default values for the result come from
// the first argument.  If a value is set in the second argument, that
// values takes precedence.
func MergeOciProvider(def *OciProvider, ovr *OciProvider) OciProvider {
	return OciProvider{
		KubeConfigPath:    ies(def.KubeConfigPath, ovr.KubeConfigPath),
		Compartment:       ies(def.Compartment, ovr.Compartment),
		Namespace:         ies(def.Namespace, ovr.Namespace),
		Images:            MergeOciImageSet(&def.Images, &ovr.Images),
		ImageBucket:       ies(def.ImageBucket, ovr.ImageBucket),
		ControlPlaneShape: MergeOciInstanceShape(&def.ControlPlaneShape, &ovr.ControlPlaneShape),
		WorkerShape:       MergeOciInstanceShape(&def.WorkerShape, &ovr.WorkerShape),
		LoadBalancer:      MergeLoadBalancer(&def.LoadBalancer, &ovr.LoadBalancer),
		SelfManaged:       iebp(def.SelfManagedPtr, ovr.SelfManagedPtr, false),
		SelfManagedPtr:    iebpp(def.SelfManagedPtr, ovr.SelfManagedPtr),
		Vcn:               ies(def.Vcn, ovr.Vcn),
		Proxy:             MergeProxy(&def.Proxy, &ovr.Proxy),
	}
}

// MergeByoProvider takes two ByoProviders and merged them into a
// third.  The default values for the result come from the first
// argument.  If a value is set in the second argument, that value
// takes precedence.
func MergeByoProvider(def *ByoProvider, ovr *ByoProvider) ByoProvider {
	return ByoProvider{
		AutomaticTokenCreation:    iebp(def.AutomaticTokenCreationPtr, ovr.AutomaticTokenCreationPtr, false),
		AutomaticTokenCreationPtr: iebpp(def.AutomaticTokenCreationPtr, ovr.AutomaticTokenCreationPtr),
		NetworkInterface:          ies(def.NetworkInterface, ovr.NetworkInterface),
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
	}
}

// MergeEphemeralConfig takes two EphemeralClusterConfigs and merges them
// into a third.  If a value is set in the second argument, that value
// takes precedence.  "Preserve" is ignored so as not to accidentally
// delete something.
func MergeEphemeralConfig(def *EphemeralClusterConfig, ovr *EphemeralClusterConfig) EphemeralClusterConfig {
	return EphemeralClusterConfig{
		Name: ies(def.Name, ovr.Name),
		Node: MergeNode(&def.Node, &ovr.Node),
	}
}

// MergeConfig takes two Configs and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeConfig(def *Config, ovr *Config) Config {
	return Config{
		Providers:                MergeProviders(&def.Providers, &ovr.Providers),
		KubeConfig:               ies(def.KubeConfig, ovr.KubeConfig),
		AutoStartUI:              ies(def.AutoStartUI, ovr.AutoStartUI),
		Proxy:                    MergeProxy(&def.Proxy, &ovr.Proxy),
		KubeAPIServerBindPort:    ieu(def.KubeAPIServerBindPort, ovr.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieu(def.KubeAPIServerBindPortAlt, ovr.KubeAPIServerBindPortAlt),
		PodSubnet:                ies(def.PodSubnet, ovr.PodSubnet),
		ServiceSubnet:            ies(def.ServiceSubnet, ovr.ServiceSubnet),
		Registry:                 ies(def.Registry, ovr.Registry),
		CertificateInformation:   MergeCertificateInformation(&def.CertificateInformation, &ovr.CertificateInformation),
		OsRegistry:               ies(def.OsRegistry, ovr.OsRegistry),
		OsTag:                    ies(def.OsTag, ovr.OsTag),
		KubeProxyMode:            ies(def.KubeProxyMode, ovr.KubeProxyMode),
		BootVolumeContainerImage: ies(def.BootVolumeContainerImage, ovr.BootVolumeContainerImage),
		CNI:                      ies(def.CNI, ovr.CNI),
		Headless:                 iebp(def.HeadlessPtr, ovr.HeadlessPtr, false),
		HeadlessPtr:              iebpp(def.HeadlessPtr, ovr.HeadlessPtr),
		Catalog:                  iebp(def.CatalogPtr, ovr.CatalogPtr, true),
		CatalogPtr:               iebpp(def.CatalogPtr, ovr.CatalogPtr),
		EphemeralConfig:          MergeEphemeralConfig(&def.EphemeralConfig, &ovr.EphemeralConfig),
		Quiet:                    iebp(def.QuietPtr, ovr.QuietPtr, false),
		QuietPtr:                 iebpp(def.QuietPtr, ovr.QuietPtr),
		KubeVersion:              ies(def.KubeVersion, ovr.KubeVersion),
		SshPublicKeyPath:         ies(def.SshPublicKeyPath, ovr.SshPublicKeyPath),
		SshPublicKey:             ies(def.SshPublicKey, ovr.SshPublicKey),
		Password:                 ies(def.Password, ovr.Password),
		ExtraIgnition:            ies(def.ExtraIgnition, ovr.ExtraIgnition),
		ExtraIgnitionInline:      ies(def.ExtraIgnitionInline, ovr.ExtraIgnitionInline),
	}
}

// MergeClusterConfig takes two ClusterConfigs and merges them into a third.
// The default values for the result come from the first argument.  If a value
// is set in the second argument, that value takes precedence.
func MergeClusterConfig(def *ClusterConfig, ovr *ClusterConfig) ClusterConfig {
	return ClusterConfig{
		WorkingDirectory:         ies(def.WorkingDirectory, ovr.WorkingDirectory),
		Name:                     ies(def.Name, ovr.Name),
		Provider:                 ies(def.Provider, ovr.Provider),
		Providers:                MergeProviders(&def.Providers, &ovr.Providers),
		WorkerNodes:              ieu(def.WorkerNodes, ovr.WorkerNodes),
		ControlPlaneNodes:        ieu(def.ControlPlaneNodes, ovr.ControlPlaneNodes),
		Proxy:                    MergeProxy(&def.Proxy, &ovr.Proxy),
		KubeAPIServerBindPort:    ieu(def.KubeAPIServerBindPort, ovr.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieu(def.KubeAPIServerBindPortAlt, ovr.KubeAPIServerBindPortAlt),
		VirtualIp:                ies(def.VirtualIp, ovr.VirtualIp),
		LoadBalancer:             ies(def.LoadBalancer, ovr.LoadBalancer),
		PodSubnet:                ies(def.PodSubnet, ovr.PodSubnet),
		ServiceSubnet:            ies(def.ServiceSubnet, ovr.ServiceSubnet),
		Registry:                 ies(def.Registry, ovr.Registry),
		CertificateInformation:   MergeCertificateInformation(&def.CertificateInformation, &ovr.CertificateInformation),
		OsTag:                    ies(def.OsTag, ovr.OsTag),
		OsRegistry:               ies(def.OsRegistry, ovr.OsRegistry),
		KubeProxyMode:            ies(def.KubeProxyMode, ovr.KubeProxyMode),
		CNI:                      ies(def.CNI, ovr.CNI),
		Headless:                 iebp(def.HeadlessPtr, ovr.HeadlessPtr, false),
		HeadlessPtr:              iebpp(def.HeadlessPtr, ovr.HeadlessPtr),
		Catalog:                  iebp(def.CatalogPtr, ovr.CatalogPtr, true),
		BootVolumeContainerImage: ies(def.BootVolumeContainerImage, ovr.BootVolumeContainerImage),
		Applications:             MergeApplications(def.Applications, ovr.Applications),
		Catalogs:                 MergeCatalogs(def.Catalogs, ovr.Catalogs),
		CatalogPtr:               iebpp(def.CatalogPtr, ovr.CatalogPtr),
		KubeVersion:              ies(def.KubeVersion, ovr.KubeVersion),
		SshPublicKeyPath:         ies(def.SshPublicKeyPath, ovr.SshPublicKeyPath),
		SshPublicKey:             ies(def.SshPublicKey, ovr.SshPublicKey),
		Password:                 ies(def.Password, ovr.Password),
		CipherSuites:             ies(def.CipherSuites, ovr.CipherSuites),
		ClusterDefinitionInline:  ies(def.ClusterDefinitionInline, ovr.ClusterDefinitionInline),
		ClusterDefinition:        ies(def.ClusterDefinition, ovr.ClusterDefinition),
		ExtraIgnition:            ies(def.ExtraIgnition, ovr.ExtraIgnition),
		ExtraIgnitionInline:      ies(def.ExtraIgnitionInline, ovr.ExtraIgnitionInline),
	}
}

// OverlayConfig merges the values from a Config into a ClusterConfig.  Values
// from the ClusterConfig take precendence.
// Precalculate Kubeversion above the return, feed that value into the right field
func OverlayConfig(cc *ClusterConfig, c *Config) ClusterConfig {
	clusterConfigToReturn := ClusterConfig{
		WorkingDirectory:         cc.WorkingDirectory,
		Name:                     cc.Name,
		Provider:                 cc.Provider,
		WorkerNodes:              cc.WorkerNodes,
		ControlPlaneNodes:        cc.ControlPlaneNodes,
		Providers:                MergeProviders(&c.Providers, &cc.Providers),
		Proxy:                    MergeProxy(&c.Proxy, &cc.Proxy),
		KubeAPIServerBindPort:    ieu(c.KubeAPIServerBindPort, cc.KubeAPIServerBindPort),
		KubeAPIServerBindPortAlt: ieu(c.KubeAPIServerBindPortAlt, cc.KubeAPIServerBindPortAlt),
		VirtualIp:                cc.VirtualIp,
		LoadBalancer:             cc.LoadBalancer,
		PodSubnet:                ies(c.PodSubnet, cc.PodSubnet),
		ServiceSubnet:            ies(c.ServiceSubnet, cc.ServiceSubnet),
		Registry:                 ies(c.Registry, cc.Registry),
		CertificateInformation:   MergeCertificateInformation(&c.CertificateInformation, &cc.CertificateInformation),
		OsRegistry:               ies(c.OsRegistry, cc.OsRegistry),
		OsTag:                    ies(c.OsTag, cc.OsTag),
		KubeProxyMode:            ies(c.KubeProxyMode, cc.KubeProxyMode),
		BootVolumeContainerImage: ies(c.BootVolumeContainerImage, cc.BootVolumeContainerImage),
		CNI:                      ies(c.CNI, cc.CNI),
		Headless:                 iebp(c.HeadlessPtr, cc.HeadlessPtr, false),
		HeadlessPtr:              iebpp(c.HeadlessPtr, cc.HeadlessPtr),
		Catalog:                  iebp(c.CatalogPtr, cc.CatalogPtr, true),
		CatalogPtr:               iebpp(c.CatalogPtr, cc.CatalogPtr),
		Applications:             MergeApplications(cc.Applications, nil),
		Catalogs:                 MergeCatalogs(cc.Catalogs, nil),
		KubeVersion:              ies(c.KubeVersion, cc.KubeVersion),
		SshPublicKeyPath:         ies(c.SshPublicKeyPath, cc.SshPublicKeyPath),
		SshPublicKey:             ies(c.SshPublicKey, cc.SshPublicKey),
		Password:                 ies(c.Password, cc.Password),
		CipherSuites:             ies(c.CipherSuites, cc.CipherSuites),
		ClusterDefinitionInline:  cc.ClusterDefinitionInline,
		ClusterDefinition:        cc.ClusterDefinition,
		ExtraIgnition:            ies(c.ExtraIgnition, cc.ExtraIgnition),
		ExtraIgnitionInline:      ies(c.ExtraIgnitionInline, cc.ExtraIgnitionInline),
	}
	return clusterConfigToReturn
}

// CopyConfig returns a deep copy of a Config
func CopyConfig(c *Config) Config {
	return MergeConfig(&Config{}, c)
}

// CopyClusterConfig returns a deep copy of a ClusterConfig
func CopyClusterConfig(cc *ClusterConfig) ClusterConfig {
	return MergeClusterConfig(&ClusterConfig{}, cc)
}
