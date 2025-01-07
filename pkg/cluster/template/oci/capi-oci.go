// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"errors"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"regexp"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/catalog/versions"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/oci"
)

type ociData struct {
	ClusterConfig   *types.ClusterConfig
	ExtraConfig     string
	KubeVersions    *versions.KubernetesVersions
	VolumePluginDir string
	CipherSuite     string
}

const (
	// Used whenever a service has to start before the CAPI
	// service used to start Kubernetes services on a node
	preKubeadmDropin = `[Unit]
Before=kubeadm.service
`

	// OCI requires custom scripting to set the hostname, as well
	// set other networking parameters.  These are gathered from
	// the instance metadata.
	//
	// Note: this is lifted verbatim from a regular OL instance.
	dhclientPath = "/etc/NetworkManager/dispatcher.d/11-dhclient"
	dhclient     = `#!/bin/bash
# run dhclient.d scripts in an emulated environment

PATH=/bin:/usr/bin:/sbin
ETCDIR=/etc/dhcp
SAVEDIR=/var/lib/dhclient
interface=$1

for optname in "${!DHCP4_@}"; do
    newoptname=${optname,,};
    newoptname=new_${newoptname#dhcp4_};
    export "${newoptname}"="${!optname}";
done

[ -f /etc/sysconfig/network ] && . /etc/sysconfig/network

[ -f /etc/sysconfig/network-scripts/ifcfg-"${interface}" ] && \
    . /etc/sysconfig/network-scripts/ifcfg-"${interface}"

if [ -d $ETCDIR/dhclient.d ]; then
    for f in $ETCDIR/dhclient.d/*.sh; do
	if [ -x "${f}" ]; then
	    subsystem="${f%.sh}"
	    subsystem="${subsystem##*/}"
	    . "${f}"
	    if [ "$2" = "up" ]; then
	        "${subsystem}_config"
	    elif [ "$2" = "dhcp4-change" ]; then
	        if [ "$subsystem" = "chrony" -o "$subsystem" = "ntp" ]; then
	            "${subsystem}_config"
	        fi
	    elif [ "$2" = "down" ]; then
	        "${subsystem}_restore"
	    fi
	fi
    done
fi
`

	proxyConfigServiceName = "ocne-proxy-config.service"
	proxyConfigContents    = `[Unit]
Description=Add API server IP address to no_proxy configuration
After=kubeadm.service

[Service]
User=root
Type=oneshot
ExecStart=/etc/ocne/proxy-config.sh

[Install]
WantedBy=multi-user.target
`
	proxyConfigScriptPattern = `#!/bin/bash

# Update the no_proxy configuration to include the IP address of the API server.

export KUBECONFIG=/etc/kubernetes/kubelet.conf
CLUSTER_NAME=%s
PROXY_CONF=%s
PROXY_CONF2=%s

# Parse the API Server IP address
#  - Get the full URL
#  - Strip off the protocol (https://)
#  - Split the resulting string at double colon
url=$(kubectl config view --cluster $CLUSTER_NAME -o=jsonpath="{.clusters[0].cluster.server}")
protocol="$(echo $url | grep :// | sed -e's,^\(.*://\).*,\1,g')"
url2="$(echo ${url/$protocol/})"
IFS=":"
read -ra ipaddr <<< "$url2"
unset IFS

# Add the IP address to the no_proxy list if not already there
echo Checking if $ipaddr exists in $PROXY_CONF
grep -q "${ipaddr}" $PROXY_CONF
if [[ $? -ne 0 ]]; then
	echo Adding $ipaddr to no_proxy setting in $PROXY_CONF
	sed -i "s/no_proxy=/no_proxy=$ipaddr,/g" $PROXY_CONF
	systemctl daemon-reload
fi
echo Checking if $ipaddr exists in $PROXY_CONF2
grep -q "${ipaddr}" $PROXY_CONF2
if [[ $? -ne 0 ]]; then
	echo Adding $ipaddr to no_proxy setting in $PROXY_CONF2
	sed -i "s/no_proxy=/no_proxy=$ipaddr,/g" $PROXY_CONF2
	systemctl daemon-reload
fi
`
)

func getExtraIgnition(clusterConfig *types.ClusterConfig) (string, error) {
	// Accept proxy configuration
	proxy, err := ignition.Proxy(&clusterConfig.Proxy, *clusterConfig.ServiceSubnet, *clusterConfig.PodSubnet, constants.InstanceMetadata)
	if err != nil {
		return "", err
	}

	// Get the basic container configuration
	container, err := ignition.ContainerConfiguration(*clusterConfig.Registry)
	if err != nil {
		return "", err
	}

	// Set up the user
	usr, err := ignition.OcneUser(*clusterConfig.SshPublicKey, *clusterConfig.SshPublicKeyPath, *clusterConfig.Password)
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

	// Add a service to update the no_proxy configuration once the API Server is up (after kubeadm.service runs)
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:     proxyConfigServiceName,
		Enabled:  util.BoolPtr(true),
		Contents: util.StrPtr(proxyConfigContents),
	})
	proxyConfigFile := &ignition.File{
		Path: "/etc/ocne/proxy-config.sh",
		Mode: 0555,
		Contents: ignition.FileContents{
			Source: fmt.Sprintf(proxyConfigScriptPattern, clusterConfig.Name, "/etc/systemd/system/ocne-update.service.d/proxy.conf", "/etc/systemd/system/crio.service.d/proxy.conf"),
		},
	}
	ignition.AddFile(ign, proxyConfigFile)

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

	// Add any additional configuration
	if *clusterConfig.ExtraIgnition != "" {
		fromExtra, err := ignition.FromPath(*clusterConfig.ExtraIgnition)
		if err != nil {
			return "", err
		}
		ign = ignition.Merge(ign, fromExtra)
	}
	if *clusterConfig.ExtraIgnitionInline != "" {
		fromExtra, err := ignition.FromString(*clusterConfig.ExtraIgnitionInline)
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

func imageFromShape(shape string, imgs *types.OciImageSet) string {
	arch := oci.ArchitectureFromShape(shape)
	if arch == "arm64" {
		return *imgs.Arm64
	}
	return *imgs.Amd64
}

func GetOciTemplate(clusterConfig *types.ClusterConfig) (string, error) {
	tmplBytes, err := template.ReadTemplate("capi-oci.yaml")

	if err != nil {
		return "", err
	}

	if *clusterConfig.ControlPlaneNodes%2 == 0 {
		return "", errors.New("the number of control plane nodes must be odd")
	}

	// Get the Kubernetes version configuration
	kubeVer, err := versions.GetKubernetesVersions(*clusterConfig.KubeVersion)
	if err != nil {
		return "", err
	}

	// If the compartment name is non-empty
	// resolve it to an ID.
	cid := clusterConfig.Providers.Oci.Compartment
	if *cid != "" {
		newCid, err := oci.GetCompartmentId(*cid)
		if err != nil {
			return "", err
		}
		clusterConfig.Providers.Oci.Compartment = &newCid
		cid = &newCid

		// Try to resolve an Image ID.  Ignore errors.
		imageId, err := oci.GetImage(constants.OciImageName, *clusterConfig.KubeVersion, "amd64", *cid)
		if err == nil {
			*clusterConfig.Providers.Oci.Images.Amd64 = imageId
		}

		imageId, err = oci.GetImage(constants.OciImageName, *clusterConfig.KubeVersion, "arm64", *cid)
		if err == nil {
			*clusterConfig.Providers.Oci.Images.Arm64 = imageId
		}
	}

	// Build up the extra ignition structures
	ign, err := getExtraIgnition(clusterConfig)
	if err != nil {
		return "", err
	}

	return util.TemplateToStringWithFuncs(string(tmplBytes), &ociData{
		ClusterConfig:   clusterConfig,
		ExtraConfig:     ign,
		KubeVersions:    &kubeVer,
		VolumePluginDir: ignition.VolumePluginDir,
		CipherSuite:     *clusterConfig.CipherSuites,
	}, map[string]any{
		"shapeImage": imageFromShape,
	})
}

// ValidateClusterResources performs basic validation on cluster resources.
func ValidateClusterResources(clusterResources string) error {
	// validate that image OCIDs are not empty and have the correct prefix
	imageRegex := regexp.MustCompile(`imageId:(.*)`)

	matches := imageRegex.FindAllStringSubmatch(clusterResources, -1)
	for _, match := range matches {
		ocid := strings.Trim(match[1], `" `)
		if len(ocid) == 0 || !strings.HasPrefix(ocid, "ocid1.image") {
			return fmt.Errorf("Image ids in cluster resources must be valid OCI image OCIDs")
		}
	}
	return nil
}
