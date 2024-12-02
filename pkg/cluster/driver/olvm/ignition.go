// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/cluster/types"
	"os"
	"path/filepath"
)

func (d *OlvmDriver) ignitionForNode(role types.NodeRole, join bool, joinToken string, caCertHashes []string) ([]byte, error) {
	var ign *igntypes.Config
	var err error

	internalLB := d.ClusterConfig.VirtualIp != ""
	kubeAPIServerIP := d.getKubeAPIServerIP()

	// Make sure there is a network interface
	if d.Config.Providers.Byo.NetworkInterface == "" {
		return nil, fmt.Errorf("A network interface must be provided")
	}

	if !join {
		// If a cluster is being initialized, then the CA certificate
		// and key need to be passed in to the new instance.
		caCert, err := os.ReadFile(d.PKIInfo.CACertPath)
		if err != nil {
			return nil, err
		}
		//caKey, err = util.ToBase64(d.PKIInfo.CAKeyPath)
		caKey, err := os.ReadFile(d.PKIInfo.CAKeyPath)
		if err != nil {
			return nil, err
		}

		expectingWorkerNodes := d.ClusterConfig.WorkerNodes > 0
		ign, err = ignition.InitializeCluster(&ignition.ClusterInit{
			OsTag:                d.Config.OsTag,
			OsRegistry:           d.Config.OsRegistry,
			KubeAPIServerIP:      kubeAPIServerIP,
			KubeAPIBindPort:      d.Config.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   d.Config.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                d.Config.Proxy,
			KubeAPIExtraSans:     []string{},
			KubePKICert:          string(caCert),
			KubePKIKey:           string(caKey),
			ServiceSubnet:        d.Config.ServiceSubnet,
			PodSubnet:            d.Config.PodSubnet,
			ExpectingWorkerNodes: expectingWorkerNodes,
			ProxyMode:            d.Config.KubeProxyMode,
			ImageRegistry:        d.Config.Registry,
			NetInterface:         d.Config.Providers.Byo.NetworkInterface,
			UploadCertificateKey: d.UploadCertificateKey,
			KubeVersion:          d.Config.KubeVersion,
			TLSCipherSuites:      d.ClusterConfig.CipherSuites,
		})
	} else {
		// Worker nodes do not get two networks.  On remote clusters,
		// they only have a bridge network.  On local clusters, they
		// only have the user network.  The result is that they are
		// not impacted by the conflicting route problem that control
		// plane nodes suffer from.  Override the gateway so that
		// the real default route is not deleted.
		ign, err = ignition.JoinCluster(&ignition.ClusterJoin{
			Role:                 role,
			OsTag:                d.Config.OsTag,
			OsRegistry:           d.Config.OsRegistry,
			KubeAPIServerIP:      kubeAPIServerIP,
			JoinToken:            joinToken,
			KubePKICertHashes:    caCertHashes,
			ImageRegistry:        d.Config.Registry,
			KubeAPIBindPort:      d.Config.KubeAPIServerBindPort,
			KubeAPIBindPortAlt:   d.Config.KubeAPIServerBindPortAlt,
			InternalLB:           internalLB,
			Proxy:                d.Config.Proxy,
			ProxyMode:            d.Config.KubeProxyMode,
			NetInterface:         d.Config.Providers.Byo.NetworkInterface,
			UploadCertificateKey: d.UploadCertificateKey,
			TLSCipherSuites:      d.ClusterConfig.CipherSuites,
		})
	}

	if err != nil {
		return nil, err
	}

	// Respect any proxy configuration that may be defined
	proxy, err := ignition.Proxy(&d.Config.Proxy, kubeAPIServerIP, d.Config.ServiceSubnet, d.Config.PodSubnet)
	if err != nil {
		return nil, err
	}

	ign = ignition.Merge(ign, proxy)

	usrIgn, err := ignition.OcneUser(d.Config.SshPublicKey, d.Config.SshPublicKeyPath, d.Config.Password)
	if err != nil {
		return nil, err
	}
	ign = ignition.Merge(ign, usrIgn)

	// Add any additional configuration
	if d.Config.ExtraIgnition != "" {
		ei := d.Config.ExtraIgnition
		if !filepath.IsAbs(ei) {
			ei, err = filepath.Abs(filepath.Join(d.ClusterConfig.WorkingDirectory, ei))
			if err != nil {
				return nil, err
			}
		}
		fromExtra, err := ignition.FromPath(ei)
		if err != nil {
			return nil, err
		}
		ign = ignition.Merge(ign, fromExtra)
	}
	if d.Config.ExtraIgnitionInline != "" {
		fromExtra, err := ignition.FromString(d.Config.ExtraIgnitionInline)
		if err != nil {
			return nil, err
		}
		ign = ignition.Merge(ign, fromExtra)
	}

	return ignition.MarshalIgnition(ign)
}

func (d *OlvmDriver) getKubeAPIServerIP() string {
	if d.ClusterConfig.VirtualIp != "" {
		return d.ClusterConfig.VirtualIp
	} else {
		return d.ClusterConfig.LoadBalancer
	}
}
