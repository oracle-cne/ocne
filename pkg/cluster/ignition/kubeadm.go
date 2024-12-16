// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/types"
)

// Why all the structs?  Can't you just use the structs from the Kubernetes
// Go libraries and avoid having to redefine all this stuff?
//
// Yes, techically.  However, when you marshal those structures into JSON
// the emit thousands and thousands of characters.  The resulting string
// is too large to fit into an ignition file for even the simplest
// configuration.  It would require that users always reference an external
// URL to configure things.

type InitConfig struct {
	ApiVersion       string           `yaml:"apiVersion"`
	Kind             string           `yaml:"kind"`
	LocalAPIEndpoint LocalAPIEndpoint `yaml:"localAPIEndpoint,omitempty"`
	NodeRegistration NodeRegistration `yaml:"nodeRegistration,omitempty"`
	CertificateKey   string           `yaml:"certificateKey,omitempty"`
	SkipPhases       []string         `yaml:"skipPhases,omitempty"`
	Patches          *Patches          `yaml:"patches,omitempty"`
}

type Patches struct {
	Directory string `yaml:"directory,omitempty"`
}

type LocalAPIEndpoint struct {
	AdvertiseAddress string `yaml:"advertiseAddress,omitempty"`
	BindPort         uint16 `yaml:"bindPort,omitempty"`
}

// https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/control-plane-flags/
// https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/
// https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/#kubeadm-k8s-io-v1beta3-ControlPlaneComponent
type ClusterConfig struct {
	ApiVersion           string            `yaml:"apiVersion"`
	Kind                 string            `yaml:"kind"`
	ApiServer            ApiServer         `yaml:"apiServer,omitempty"`
	ControllerManager    ControllerManager `yaml:"controllerManager,omitempty"`
	Scheduler            Scheduler         `yaml:"scheduler,omitempty"`
	Networking           Networking        `yaml:"networking"`
	ImageRepository      string            `yaml:"imageRepository"`
	KubernetesVersion    string            `yaml:"kubernetesVersion"`
	ControlPlaneEndpoint string            `yaml:"controlPlaneEndpoint,omitempty"`
	Etcd                 Etcd              `yaml:"etcd"`
	DNS                  DNS               `yaml:"dns"`
}

// Etcd represents the etcd object in the cluster config. This is used to modify
// the etcd image registy and tag
type Etcd struct {
	Local EtcdLocal `yaml:"local"`
}

// EtcdLocal defines configuration options for etcd to be used in
// a kubeadm config definition
type EtcdLocal struct {
	CustomImage  `yaml:",inline"`
	ExtraArgs    EtcdExtraArgs `yaml:"extraArgs"`
	PeerCertSans []string      `yaml:"peerCertSANs"`
}

type EtcdExtraArgs struct {
	TLSCipherSuites           string `yaml:"cipher-suites,omitempty"`
	ListenClientURLs          string `yaml:"listen-client-urls,omitempty"`
	ListenPeerURLs            string `yaml:"listen-peer-urls,omitempty"`
	ListenMetricsURLs         string `yaml:"listen-metrics-urls,omitempty"`
	AdvertiseClientURLs       string `yaml:"advertise-client-urls,omitempty"`
	InitialAdvertisePeersURLs string `yaml:"initial-advertise-peer-urls,omitempty"`
}

// DNS holds the configuration for the dns
type DNS struct {
	CustomImage `yaml:",inline"`
}

// CustomImage defines the custom image and tag
type CustomImage struct {
	ImageRepository string `yaml:"imageRepository"`
	ImageTag        string `yaml:"imageTag"`
}

type ApiServer struct {
	CertSans  []string           `yaml:"certSANs,omitempty"`
	ExtraArgs ApiServerExtraArgs `yaml:"extraArgs,omitempty"`
}

type ApiServerExtraArgs struct {
	TLSMinVersion   string `yaml:"tls-min-version,omitempty"`
	TLSCipherSuites string `yaml:"tls-cipher-suites,omitempty"`
}

type ControllerManager struct {
	ExtraArgs ControllerManagerExtraArgs `yaml:"extraArgs,omitempty"`
}

type ControllerManagerExtraArgs struct {
	TLSMinVersion   string `yaml:"tls-min-version,omitempty"`
	TLSCipherSuites string `yaml:"tls-cipher-suites,omitempty"`
	CloudProvider   string `yaml:"cloud-provider,omitempty"`
	BindAddress     string `yaml:"bind-address,omitempty"`
}

// https://kubernetes.io/docs/reference/config-api/kube-scheduler-config.v1beta3/#kubescheduler-config-k8s-io-v1beta2-KubeSchedulerConfiguration
type Scheduler struct {
	ExtraArgs SchedulerExtraArgs `yaml:"extraArgs,omitempty"`
}

type SchedulerExtraArgs struct {
	TLSMinVersion   string `yaml:"tls-min-version,omitempty"`
	TLSCipherSuites string `yaml:"tls-cipher-suites,omitempty"`
	BindAddress     string `yaml:"bind-address,omitempty"`
}

type Networking struct {
	ServiceCIDR string `yaml:"serviceSubnet"`
	PodCIDR     string `yaml:"podSubnet"`
}

type JoinConfig struct {
	ApiVersion       string           `yaml:"apiVersion"`
	Kind             string           `yaml:"kind"`
	ControlPlane     ControlPlane     `yaml:"controlPlane,omitempty"`
	NodeRegistration NodeRegistration `yaml:"nodeRegistration,omitempty"`
	Discovery        Discovery        `yaml:"discovery,omitempty"`
	Patches          *Patches          `yaml:"patches,omitempty"`
}

type ControlPlane struct {
	LocalAPIEndPoint LocalAPIEndpoint `yaml:"localAPIEndpoint"`
	CertificateKey   string           `yaml:"certificateKey,omitempty"`
}

type NodeRegistration struct {
	KubeletExtraArgs KubeletExtraArgs `yaml:"kubeletExtraArgs"`
	Taints           *[]string        `yaml:"taints"`
}

type KubeletExtraArgs struct {
	NodeIP            string `yaml:"node-ip,omitempty"`
	TLSMinVersion     string `yaml:"tls-min-version,omitempty"`
	TLSCipherSuites   string `yaml:"tls-cipher-suites,omitempty"`
	Address           string `yaml:"address,omitempty"`
	AuthorizationMode string `yaml:"authorization-mode,omitempty"`
	VolumePluginDir   string `yaml:"volume-plugin-dir,omitempty"`
}

type Discovery struct {
	BootstrapToken BootstrapTokenDiscovery `yaml:"bootstrapToken,omitempty"`
}

type BootstrapTokenDiscovery struct {
	ApiServerEndpoint string   `yaml:"apiServerEndpoint,omitempty"`
	Token             string   `yaml:"token"`
	CACertHashes      []string `yaml:"caCertHashes,omitempty"`
}

func GenerateKubeadmInit(ci *ClusterInit) *InitConfig {
	// If the number of worker nodes is zero, initialize an empty list of taints, so the noSchedule taint is not applied
	ret := &InitConfig{
		ApiVersion: InitConfigAPIVersion,
		Kind:       InitConfigKind,
		LocalAPIEndpoint: LocalAPIEndpoint{
			AdvertiseAddress: "NODE_IP",
			BindPort:         getInitLocalAPIEndpointBindPort(ci),
		},
		NodeRegistration: NodeRegistration{
			KubeletExtraArgs: KubeletExtraArgs{
				NodeIP:            "NODE_IP",
				TLSMinVersion:     "VersionTLS12",
				Address:           "0.0.0.0",
				AuthorizationMode: "AlwaysAllow",
				VolumePluginDir:   VolumePluginDir,
				TLSCipherSuites:   ci.TLSCipherSuites,
			},
		},
		CertificateKey: ci.UploadCertificateKey,
		SkipPhases: []string{
			"addon/kube-proxy",
			"preflight",
		},
		Patches: &Patches{
			Directory: "/etc/ocne/ock",
		},
	}
	if !ci.ExpectingWorkerNodes {
		ret.NodeRegistration.Taints = &[]string{}
	}
	return ret
}

func GenerateKubeadmInitYaml(ci *ClusterInit) (string, error) {
	ki := GenerateKubeadmInit(ci)
	outBytes, err := yaml.Marshal(ki)
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}

func GenerateKubeadmJoin(cj *ClusterJoin) *JoinConfig {
	ret := &JoinConfig{
		ApiVersion: JoinConfigAPIVersion,
		Kind:       JoinConfigKind,
		NodeRegistration: NodeRegistration{
			KubeletExtraArgs: KubeletExtraArgs{
				NodeIP:            "NODE_IP",
				Address:           "0.0.0.0",
				AuthorizationMode: "AlwaysAllow",
				VolumePluginDir:   VolumePluginDir,
				TLSCipherSuites:   cj.TLSCipherSuites,
			},
		},
		Discovery: Discovery{
			BootstrapToken: BootstrapTokenDiscovery{
				ApiServerEndpoint: fmt.Sprintf("%s:%d", cj.KubeAPIServerIP, cj.KubeAPIBindPort),
				Token:             cj.JoinToken,
				CACertHashes:      cj.KubePKICertHashes,
			},
		},
		Patches: &Patches{
			Directory: "/etc/ocne/ock",
		},
	}
	if cj.Role == types.ControlPlaneRole {
		ret.ControlPlane = ControlPlane{
			LocalAPIEndPoint: LocalAPIEndpoint{
				AdvertiseAddress: "NODE_IP",
				BindPort:         getJoinLocalAPIEndpointBindPort(cj),
			},
			CertificateKey: cj.UploadCertificateKey,
		}
	}

	return ret
}

func GenerateKubeadmJoinYaml(cj *ClusterJoin) (string, error) {
	kj := GenerateKubeadmJoin(cj)
	outBytes, err := yaml.Marshal(kj)
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}

func GenerateClusterConfiguration(ci *ClusterInit, kubeVersions versions.KubernetesVersions) *ClusterConfig {
	cro := "container-registry.oracle.com/olcne"
	tmv := "VersionTLS12"
	ret := &ClusterConfig{
		ApiVersion:           "kubeadm.k8s.io/v1beta3",
		Kind:                 "ClusterConfiguration",
		ImageRepository:      cro,
		KubernetesVersion:    kubeVersions.Kubernetes,
		ControlPlaneEndpoint: fmt.Sprintf("%s:%d", ci.KubeAPIServerIP, ci.KubeAPIBindPort),
		Networking: Networking{
			ServiceCIDR: ci.ServiceSubnet,
			PodCIDR:     ci.PodSubnet,
		},
		Etcd: Etcd{
			Local: EtcdLocal{
				CustomImage: CustomImage{
					ImageRepository: cro,
					ImageTag:        kubeVersions.Etcd,
				},
				ExtraArgs: EtcdExtraArgs{
					ListenMetricsURLs: "http://0.0.0.0:2381",
					TLSCipherSuites:   ci.TLSCipherSuites,
				},
			},
		},
		DNS: DNS{
			CustomImage: CustomImage{
				ImageRepository: cro,
				ImageTag:        kubeVersions.CoreDNS,
			},
		},
		ControllerManager: ControllerManager{
			ExtraArgs: ControllerManagerExtraArgs{
				TLSMinVersion:   tmv,
				TLSCipherSuites: ci.TLSCipherSuites,
			},
		},
		ApiServer: ApiServer{
			ExtraArgs: ApiServerExtraArgs{
				TLSMinVersion:   tmv,
				TLSCipherSuites: ci.TLSCipherSuites,
			},
			CertSans: ci.KubeAPIExtraSans,
		},
		Scheduler: Scheduler{
			ExtraArgs: SchedulerExtraArgs{
				TLSMinVersion:   tmv,
				TLSCipherSuites: ci.TLSCipherSuites,
			},
		},
	}

	return ret
}

func GenerateClusterConfigurationYaml(ci *ClusterInit) (string, error) {
	// Get image tags for etcd, coredns, etc. for the given k8s version
	kubeVersions, err := versions.GetKubernetesVersions(ci.KubeVersion)
	if err != nil {
		return "", err
	}

	kc := GenerateClusterConfiguration(ci, kubeVersions)
	outBytes, err := yaml.Marshal(kc)
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}

func getInitLocalAPIEndpointBindPort(ci *ClusterInit) uint16 {
	if ci.InternalLB {
		return ci.KubeAPIBindPortAlt
	} else {
		return ci.KubeAPIBindPort
	}
}

func getJoinLocalAPIEndpointBindPort(cj *ClusterJoin) uint16 {
	if cj.InternalLB {
		return cj.KubeAPIBindPortAlt
	} else {
		return cj.KubeAPIBindPort
	}
}
