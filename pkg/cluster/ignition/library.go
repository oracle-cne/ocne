// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"fmt"
	"os"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"

	clustertypes "github.com/oracle-cne/ocne/pkg/cluster/types"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

const (
	Country = "US"
	Org     = "OCNE"
	OrgUnit = "OCNE"
	State   = "TX"

	InitConfigAPIVersion = "kubeadm.k8s.io/v1beta3"
	InitConfigKind       = "InitConfiguration"
	JoinConfigAPIVersion = "kubeadm.k8s.io/v1beta3"
	JoinConfigKind       = "JoinConfiguration"
	KubeadmFilePath      = "/etc/kubernetes/kubeadm.conf"
	CaCrtFilePath        = "/etc/kubernetes/pki/ca.crt"
	CaKeyFilePath        = "/etc/kubernetes/pki/ca.key"
	VolumePluginDir      = "/var/lib/kubelet/volumeplugins"

	KubeProxyMode = "iptables"
	ActionInit    = "init"
	ActionJoin    = "join"

	KubeletServiceName    = "kubelet.service"
	CrioServiceName       = "crio.service"
	IscsidServiceName     = "iscsid.service"
	KeepalivedServiceName = "keepalived.service"

	// Note that OcneServiceCommonBootstrapPatthen has and seemingly
	// pointless endline.  That endline is actually very important.
	// There is a bug in the coreos/go-systemd library used by ignition
	// that is not capable of handling unit files that do not with with
	// and endline.  It counts all such lines as "too long".
	//
	// Please refer to this: https://github.com/coreos/go-systemd/blob/v22.5.0/unit/deserialize.go#L153
	OcneServiceName                   = "ocne.service"
	OcneServiceCommonBootstrapPattern = `[Service]
Environment=ACTION={{.Action}}
Environment=NET_INTERFACE={{.NetInterface}}
`

	OcneUpdateServiceName = "ocne-update.service"
	OcneUpdateConfigPath  = "/etc/ocne/update.yaml"
	OcneUpdateYamlPattern = `registry: %s
tag: %s
transport: %s
`

	// Populating core configuration files, such as crio.conf and the
	// kubeadm init/join files have been moved to the "files" section
	// of ignition.  The script baked in to the OS image are not aware
	// of this and continue to assume that it needs to generate them.
	// Until the OS image is updated, overwrite that script with this
	// simper one.
	OcneSh = `#! /bin/bash
set -x
set -e

if [[ -f "/etc/ocne/reset-kubeadm" ]]; then
	echo Performing kubeadm reset
	kubeadm reset -f && rm /etc/ocne/reset-kubeadm
fi

NODE_IP=$(ip addr show $NET_INTERFACE | grep 'inet\b' | awk '{print $2}' | cut -d/ -f1 | head -n 1)

# If Kubernetes is already configured, don't bother doing anything
if [[ -f "/etc/kubernetes/kubelet.conf" ]]; then
	echo Kubernetes is already initialized
	exit 0
fi

K8S=/etc/kubernetes
PKI=$K8S/pki

systemctl enable --now crio.service
systemctl enable kubelet.service

# update kubeadm.conf to replace NODE_IP with $NODE_IP
sed -i -e 's/NODE_IP/'"$NODE_IP"'/g' ${K8S}/kubeadm.conf

if [[ "$ACTION" == "" ]]; then
	kubeadm init --config /etc/ocne/kubeadm-default.conf
	KUBECONFIG=/etc/kubernetes/admin.conf kubectl taint node $(hostname)  node-role.kubernetes.io/control-plane:NoSchedule-
elif [[ "$ACTION" == "init" ]]; then
	echo Initalizing new Kubernetes cluster
	mkdir -p $PKI

	kubeadm init --config ${K8S}/kubeadm.conf --upload-certs
elif [[ "$ACTION" == "join" ]]; then
	echo Joining existing Kubernetes cluster
	kubeadm join --config ${K8S}/kubeadm.conf
else
	echo "Action '$ACTION' is invalid.  Valid values are 'init' and 'join'"
	exit 1
fi

# keepalived track script user keepalived_script needs to read this file
if [ -f "/etc/kubernetes/admin.conf" ]; then
	cp /etc/kubernetes/admin.conf /etc/keepalived/kubeconfig
	chown keepalived_script:keepalived_script /etc/keepalived/kubeconfig
	chmod 400 /etc/keepalived/kubeconfig
fi
`

	ContainerRegistryPath    = "/etc/containers/registries.conf"
	ContainerRegistryPattern = `unqualified-search-registries = ["{{.}}"]
`

	NetworkScriptPattern = `TYPE={{.Type}}
PROXY_METHOD={{.ProxyMode}}
BROWSER_ONLY={{yesno .BrowserOnly}}
BOOTPROTO={{.BootProto}}
DEFROUTE={{yesno .DefaultRoute}}
IPV4_FAILURE_FATAL={{yesno .IPV4FailureFatal}}
IPV6INIT={{yesno .IPV6Init}}
IPV6_AUTOCONF={{yesno .IPV6Autoconf}}
IPV6_DEFROUTE={{yesno .IPV6DefaultRoute}}
IPV6_FAILURE_FATAL={{yesno .IPV6FailureFatal}}
IPV6_ADDR_GEN_MODE={{.IPV6AddrGenMode}}
NAME={{.Name}}
DEVICE={{.Name}}
ONBOOT={{yesno .OnBoot}}
`

	ProxyDropinPattern = `[Service]
{{- if .HttpsProxy}}
Environment=HTTPS_PROXY={{.HttpsProxy}}
Environment=https_proxy={{.HttpsProxy}}
{{- end}}
{{- if .HttpProxy}}
Environment=HTTP_PROXY={{.HttpProxy}}
Environment=http_proxy={{.HttpProxy}}
{{- end}}
{{- if .NoProxy}}
Environment=no_proxy={{.NoProxy}}
{{- end}}
`
)

type ClusterInit struct {
	OsTag                string
	OsRegistry           string
	ImageRegistry        string
	KubeAPIServerIP      string
	KubeAPIBindPort      uint16
	KubeAPIBindPortAlt   uint16
	InternalLB           bool
	Proxy                types.Proxy
	KubeAPIExtraSans     []string
	KubePKICert          string
	KubePKIKey           string
	ServiceSubnet        string
	PodSubnet            string
	ExpectingWorkerNodes bool
	ProxyMode            string
	NetInterface         string
	UploadCertificateKey string
	KubeVersion          string
	TLSCipherSuites      string
}

type ClusterJoin struct {
	Role                 clustertypes.NodeRole
	OsTag                string
	OsRegistry           string
	ImageRegistry        string
	KubeAPIServerIP      string
	JoinToken            string
	KubePKICertHashes    []string
	KubeAPIBindPort      uint16
	KubeAPIBindPortAlt   uint16
	InternalLB           bool
	Proxy                types.Proxy
	ProxyMode            string
	NetInterface         string
	UploadCertificateKey string
	TLSCipherSuites      string
}

type combinedConfig struct {
	Action          string
	KubeAPIServerIP string
	NetInterface    string
	OsTag           string
	OsRegistry      string
}

type NetworkScript struct {
	Name             string
	Type             string
	OnBoot           bool
	BrowserOnly      bool
	BootProto        string
	DefaultRoute     bool
	ProxyMode        string
	IPV4FailureFatal bool
	IPV6Init         bool
	IPV6Autoconf     bool
	IPV6DefaultRoute bool
	IPV6FailureFatal bool
	IPV6AddrGenMode  string
}

type clusterCommonConfig struct {
	OsRegistry    string
	OsTag         string
	ImageRegistry string
	NetInterface  string
}

func clusterCommon(cc *clusterCommonConfig, action string) (*igntypes.Config, error) {
	ret := NewIgnition()

	combinedConfig := combinedConfig{
		Action:       action,
		NetInterface: cc.NetInterface,
	}

	bootstrapTmpl := OcneServiceCommonBootstrapPattern
	bootstrapFile, err := util.TemplateToString(bootstrapTmpl, &combinedConfig)
	if err != nil {
		return nil, err
	}

	// Unit for ocne.service
	ocneUnit := &igntypes.Unit{
		Name:    OcneServiceName,
		Enabled: util.BoolPtr(true),
		Dropins: []igntypes.Dropin{
			{
				Name:     "bootstrap.conf",
				Contents: &bootstrapFile,
			},
		},
	}

	// Unit for ocne-update.service
	ocneUpdateUnit := &igntypes.Unit{
		Name:    OcneUpdateServiceName,
		Enabled: util.BoolPtr(true),
	}

	// Update service configuration file
	ostreeTransport, registry, tag, err := image.ParseOstreeReference(cc.OsRegistry)
	if err != nil {
		return nil, err
	}
	if tag != "" {
		return nil, fmt.Errorf("osRegistry field cannot have a tag")
	}
	updateFile := &File{
		Path: OcneUpdateConfigPath,
		Mode: 0400,
		Contents: FileContents{
			Source: fmt.Sprintf(OcneUpdateYamlPattern, registry, cc.OsTag, ostreeTransport),
		},
	}
	ocneShFile := &File{
		Path: "/etc/ocne/ocne.sh",
		Mode: 0555,
		Contents: FileContents{
			Source: OcneSh,
		},
	}

	container, err := ContainerConfiguration(cc.ImageRegistry)
	if err != nil {
		return nil, err
	}

	// Take all of the resources that were made and stick them into the
	// ignition structure.  Errors can be ignored because this is a fresh
	// struct and it is guaranteed that no errors will happen so long as
	// there are no conflicts made specifically in this function.
	AddFile(ret, updateFile)
	AddFile(ret, ocneShFile)
	ret = AddUnit(ret, ocneUpdateUnit)
	ret = AddUnit(ret, ocneUnit)
	ret = Merge(ret, container)

	return ret, nil
}

func ContainerConfiguration(registry string) (*igntypes.Config, error) {
	ret := NewIgnition()

	// Enable the crio service so that it auto-starts at boot
	ret = AddUnit(ret, &igntypes.Unit{
		Name:    CrioServiceName,
		Enabled: util.BoolPtr(true),
	})

	// Enable the kubelet service so that it auto-starts at boot.
	// It's going to fail at first, but eventually it will succeed
	// as `kubeadm init/join` do their job.
	ret = AddUnit(ret, &igntypes.Unit{
		Name:    KubeletServiceName,
		Enabled: util.BoolPtr(true),
	})

	registryFile, err := util.TemplateToString(ContainerRegistryPattern, registry)
	if err != nil {
		return nil, err
	}
	containerRegistryFile := &File{
		Path: ContainerRegistryPath,
		Mode: 0644,
		Contents: FileContents{
			Source: registryFile,
		},
	}
	AddFile(ret, containerRegistryFile)
	return ret, nil
}

func InitializeCluster(ci *ClusterInit) (*igntypes.Config, error) {
	ccc := &clusterCommonConfig{
		OsRegistry:    ci.OsRegistry,
		OsTag:         ci.OsTag,
		ImageRegistry: ci.ImageRegistry,
		NetInterface:  ci.NetInterface,
	}
	ret, err := clusterCommon(ccc, ActionInit)
	if err != nil {
		fmt.Errorf("Have error: %+v", err)
		return nil, err
	}

	// Generate kubeadm configuration
	ki, err := GenerateKubeadmInitYaml(ci)
	if err != nil {
		return nil, err
	}
	cc, err := GenerateClusterConfigurationYaml(ci)
	if err != nil {
		return nil, err
	}
	kpc, err := GenerateKubeProxyConfigurationYaml(ci.ProxyMode)
	kubeadmConfigRaw := fmt.Sprintf("%s\n---\n%s---\n%s", ki, cc, kpc)

	kubeadmFile := &File{
		Path: KubeadmFilePath,
		Mode: 0600,
		Contents: FileContents{
			Source: kubeadmConfigRaw,
		},
	}
	caCertFile := &File{
		Path: CaCrtFilePath,
		Mode: 0600,
		Contents: FileContents{
			Source: ci.KubePKICert,
		},
	}
	caKeyFile := &File{
		Path: CaKeyFilePath,
		Mode: 0600,
		Contents: FileContents{
			Source: ci.KubePKIKey,
		},
	}

	// Take all of the resources that were made and stick them into the
	// ignition structure.  Errors can be ignored because this is a fresh
	// struct and it is guaranteed that no errors will happen so long as
	// there are no conflicts made specifically in this function.
	AddFile(ret, kubeadmFile)
	AddFile(ret, caCertFile)
	AddFile(ret, caKeyFile)

	if ci.InternalLB {
		ret, err = IgnitionForVirtualIp(ret, ci.KubeAPIBindPort, ci.KubeAPIBindPortAlt, ci.KubeAPIServerIP, &ci.Proxy, ci.NetInterface)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func JoinCluster(cj *ClusterJoin) (*igntypes.Config, error) {
	ccc := &clusterCommonConfig{
		OsRegistry:    cj.OsRegistry,
		OsTag:         cj.OsTag,
		ImageRegistry: cj.ImageRegistry,
		NetInterface:  cj.NetInterface,
	}
	ret, err := clusterCommon(ccc, ActionJoin)
	if err != nil {
		return nil, err
	}
	joinYaml, err := GenerateKubeadmJoinYaml(cj)
	if err != nil {
		return nil, err
	}
	kubeadmConfigRaw := joinYaml
	if cj.Role == clustertypes.ControlPlaneRole {
		kpc, _ := GenerateKubeProxyConfigurationYaml(cj.ProxyMode)
		kubeadmConfigRaw = fmt.Sprintf("%s\n---\n%s", joinYaml, kpc)
	}

	kubeadmFile := &File{
		Path: KubeadmFilePath,
		Mode: 0600,
		Contents: FileContents{
			Source: kubeadmConfigRaw,
		},
	}
	AddFile(ret, kubeadmFile)

	if cj.Role == clustertypes.ControlPlaneRole && cj.InternalLB {
		ret, err = IgnitionForVirtualIp(ret, cj.KubeAPIBindPort, cj.KubeAPIBindPortAlt, cj.KubeAPIServerIP, &cj.Proxy, cj.NetInterface)
		if err != nil {
			return nil, err
		}
	}

	return ret, err
}

func DefaultNetwork() *NetworkScript {
	return &NetworkScript{
		Type:             "Ethernet",
		OnBoot:           true,
		ProxyMode:        "none",
		BrowserOnly:      false,
		BootProto:        "dhcp",
		DefaultRoute:     true,
		IPV4FailureFatal: false,
		IPV6Init:         true,
		IPV6Autoconf:     true,
		IPV6DefaultRoute: true,
		IPV6FailureFatal: false,
		IPV6AddrGenMode:  "eui64",
	}
}

func (n *NetworkScript) ToFile() (*File, error) {
	netFile, err := util.TemplateToString(NetworkScriptPattern, n)
	if err != nil {
		return nil, err
	}

	return &File{
		Path:       fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-%s", n.Name),
		Filesystem: "root",
		Mode:       0444,
		Contents: FileContents{
			Source: netFile,
		},
	}, nil
}

// Proxy converts a proxy configuration into the correct set of
// ignition objects for any and all OCNE components and services.
func Proxy(inProxy *types.Proxy, noProxies ...string) (*igntypes.Config, error) {
	ret := NewIgnition()

	proxy := appendNoProxies(*inProxy, noProxies)

	// If there is no proxy configured, then
	if *proxy.HttpsProxy == "" && *proxy.HttpProxy == "" && *proxy.NoProxy == "" {
		return ret, nil
	}

	// Two services need proxy configuration, namely crio.service and
	// ocne-update.service.  Each of these pull container images.  The
	// same dropin contents can be used for each.  Render a string that
	// contains the correct configuration, and then make a dropin for each
	conf, err := util.TemplateToString(ProxyDropinPattern, proxy)
	if err != nil {
		return nil, err
	}

	// Add the dropin to the two services.
	ret = AddUnit(ret, &igntypes.Unit{
		Name:    "crio.service",
		Enabled: util.BoolPtr(true),
		Dropins: []igntypes.Dropin{
			{
				Name:     "proxy.conf",
				Contents: util.StrPtr(conf),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	ret = AddUnit(ret, &igntypes.Unit{
		Name:    "kublet.service",
		Enabled: util.BoolPtr(true),
	})
	if err != nil {
		return nil, err
	}

	ret = AddUnit(ret, &igntypes.Unit{
		Name:    "ocne-update.service",
		Enabled: util.BoolPtr(true),
		Dropins: []igntypes.Dropin{
			{
				Name:     "proxy.conf",
				Contents: util.StrPtr(conf),
			},
		},
	})

	ret = AddUnit(ret, &igntypes.Unit{
		Name: "rpm-ostreed.service",
		Dropins: []igntypes.Dropin{
			{
				Name:     "proxy.conf",
				Contents: util.StrPtr(conf),
			},
		},
	})

	return ret, nil
}

// OcneUser adds the default user to the ignition configuration
func OcneUser(sshKey string, sshKeyPath string, password string) (*igntypes.Config, error) {
	ret := NewIgnition()

	if sshKey != "" {
		logutils.Debug("User specified sshKey in configuration, ignoring sshKeyPath")
	} else if sshKeyPath != "" {
		logutils.Debug("User specified sshKeyPath in configuration, reading key file")
		keyBytes, err := os.ReadFile(sshKeyPath)
		if err != nil {
			return nil, err
		}

		sshKey = string(keyBytes)
	}

	err := AddUser(ret, &User{
		Name:     "ocne",
		SshKey:   sshKey,
		Password: password,
		Groups: []string{
			"wheel",
		},
		Shell: "/usr/bin/rescue.sh",
	})
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// append additional NoProxies to the proxy struct
func appendNoProxies(proxy types.Proxy, np []string) *types.Proxy {
	nps := strings.Join(np, ",")
	if *proxy.NoProxy != "" {
		nps = strings.Join([]string{*proxy.NoProxy, nps}, ",")
	}
	return &types.Proxy{
		HttpsProxy: proxy.HttpsProxy,
		HttpProxy:  proxy.HttpProxy,
		NoProxy:    &nps,
	}
}
