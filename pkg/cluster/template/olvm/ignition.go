// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/util"
	"strings"
)

const (
	// Used whenever a service has to start before the CAPI
	// service used to start Kubernetes services on a node
	preKubeadmDropin = `[Unit]
Before=kubeadm.service
`
	// Used to start services needed for kubeadm service
	enableServicesDropinFile = "enable-services.conf"
	enableServicesDropin     = `[Service]
	ExecStartPre=/bin/bash -c "/etc/ocne/enableServices.sh &"
`
	enableServicesScriptPath = "/etc/ocne/enableServices.sh"
	enableServicesScript     = `#! /bin/bash
set -x
set -e
systemctl enable --now crio.service 
systemctl enable kubelet.service
systemctl enable --now kubeadm.service
`
)

// TODO USE LIBVIRT IGNITION !!!
func getExtraIgnition(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {

	// Accept proxy configuration
	proxy, err := ignition.Proxy(&clusterConfig.Proxy, clusterConfig.ServiceSubnet, clusterConfig.PodSubnet, constants.InstanceMetadata)
	if err != nil {
		return "", err
	}

	// Get the basic container configuration
	container, err := ignition.ContainerConfiguration(clusterConfig.Registry)
	if err != nil {
		return "", err
	}

	// Set up the user
	usr, err := ignition.OcneUser(clusterConfig.SshPublicKey, clusterConfig.SshPublicKeyPath, clusterConfig.Password)
	if err != nil {
		return "", err
	}

	ign := ignition.NewIgnition()

	// Cluster API has its own service to start
	// kublet.  Use that one.
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:    "ocne.service",
		Enabled: util.BoolPtr(false),
	})

	// Update service configuration file
	updateFile := &ignition.File{
		Path: ignition.OcneUpdateConfigPath,
		Mode: 0400,
		Contents: ignition.FileContents{
			Source: fmt.Sprintf(ignition.OcneUpdateYamlPattern, clusterConfig.OsRegistry, clusterConfig.OsTag),
		},
	}
	ignition.AddFile(ign, updateFile)

	// Start the iscsi service
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:    ignition.IscsidServiceName,
		Enabled: util.BoolPtr(true),
	})

	// **NOTE: This is a temporary workaroud to enable/start certain services that are
	// enabled in ignition but don't start for some reason.
	// Piggyback on the ocne-update service which is always started
	// Add script to enable services
	enableServicesFile := &ignition.File{
		Path: enableServicesScriptPath,
		Mode: 0555,
		Contents: ignition.FileContents{
			Source: enableServicesScript,
		},
	}
	ignition.AddFile(ign, enableServicesFile)

	// Add drop-in to run enable services script
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:    ignition.OcneUpdateServiceName,
		Enabled: util.BoolPtr(true),
		Dropins: []igntypes.Dropin{
			{
				Name:     "pre-kubeadm.conf",
				Contents: util.StrPtr(preKubeadmDropin),
			},
			{
				Name:     enableServicesDropinFile,
				Contents: util.StrPtr(enableServicesDropin),
			},
		},
	})

	// Add a systemd service to start crio and enable kubelet before
	// kubeadm.service.
	//ign = ignition.AddUnit(ign, &igntypes.Unit{
	//	Name:     ignition.PreKubeAdmServiceName,
	//	Enabled:  util.BoolPtr(true),
	//	Contents: util.StrPtr(preKubeadmService),
	//})

	// Merge everything together
	ign = ignition.Merge(ign, container)
	ign = ignition.Merge(ign, proxy)
	ign = ignition.Merge(ign, usr)

	// TEMP - For Now assume internal LB
	internalLB := true
	if internalLB {
		ign, err = ignition.IgnitionForVirtualIp(ign, config.KubeAPIServerBindPort, config.KubeAPIServerBindPortAlt,
			clusterConfig.VirtualIp, &clusterConfig.Proxy, clusterConfig.Providers.Olvm.NetworkInterface)
		if err != nil {
			return "", err
		}
	}

	// Add any additional configuration
	if clusterConfig.ExtraIgnition != "" {
		fromExtra, err := ignition.FromPath(clusterConfig.ExtraIgnition)
		if err != nil {
			return "", err
		}
		ign = ignition.Merge(ign, fromExtra)
	}
	if clusterConfig.ExtraIgnitionInline != "" {
		fromExtra, err := ignition.FromString(clusterConfig.ExtraIgnitionInline)
		if err != nil {
			return "", err
		}
		ign = ignition.Merge(ign, fromExtra)
	}

	ignBytes, err := ignition.MarshalIgnition(ign)
	if err != nil {
		return "", nil
	}

	ret := string(ignBytes)

	// Indent the string to match the template.  12 spaces
	retLines := strings.Split(ret, "\n")
	for i, l := range retLines {
		retLines[i] = fmt.Sprintf("            %s", l)
	}
	ret = strings.Join(retLines, "\n")

	return ret, nil
}
