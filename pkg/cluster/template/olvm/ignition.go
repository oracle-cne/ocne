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
)

// TODO USE LIBVIRT IGNITION !!!
func getExtraIgnition(confg *types.Config, clusterConfig *types.ClusterConfig) (string, error) {

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

	// Add a systemd dropin to force crio to start before
	// kubeadm.service.  That service is included in the
	// ignition configuration from the CAPI provider.
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:    ignition.CrioServiceName,
		Enabled: util.BoolPtr(true),
		Dropins: []igntypes.Dropin{
			{
				Name:     "pre-kubeadm.conf",
				Contents: util.StrPtr(preKubeadmDropin),
			},
			{
				Name: "ocid-populate.conf",
				Contents: util.StrPtr(`[Service]
ExecStartPre=sh -c 'export OCID=$(curl -H "Authorization: Bearer Oracle" -L http://169.254.169.254/opc/v2/instance/id); sed -i "s/{{ ds\\[\\"id\\"\\] }}/$OCID/g" /etc/kubeadm.yml'
ExecStartPost=sh -c 'mv /etc/systemd/system/crio.service.d/ocid-populate.conf /tmp/'
`),
			},
		},
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

	// Merge everything together
	ign = ignition.Merge(ign, container)
	ign = ignition.Merge(ign, proxy)
	ign = ignition.Merge(ign, usr)

	// TEMP - For Now assume internal LB
	//internalLB := true
	//if internalLB {
	//
	//	//ret, err = ignition.GenerateAssetsForVirtualIp(ign\, ci.KubeAPIBindPort, ci.KubeAPIBindPortAlt, ci.KubeAPIServerIP, &ci.Proxy, ci.NetInterface)
	//	//if err != nil {
	//	//	return nil, err
	//	//}
	//}

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
