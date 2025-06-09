// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"fmt"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util"
)

const (
	// Used whenever a service has to start before the CAPI
	// service used to start Kubernetes services on a node
	preKubeadmDropin = `[Unit]
Before=kubeadm.service
`
	// Used whenever a service has to start after the CAPI
	// service used to start Kubernetes services on a node
	postKubeadmDropin = `[Unit]
After=kubeadm.service
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

	// Used to start services needed for kubeadm service
	copyKubeconfigDropinFile       = "copy-kubeconfig.conf"
	copyKubeconfigDropinKeepalived = `[Service]
ExecStartPre=/bin/bash -c "/etc/ocne/keepalived-copy-kubeconfig.sh"
`
	copyKubeconfigDropinNginx = `[Service]
ExecStartPre=/bin/bash -c "/etc/ocne/nginx-copy-kubeconfig.sh"
`
	// Copy kubeconfig and change ownership
	copyKubeconfigScriptPathKeepalived = "/etc/ocne/keepalived-copy-kubeconfig.sh"
	copyKubeconfigScriptPathNginx      = "/etc/ocne/nginx-copy-kubeconfig.sh"
	copyKubeconfigScriptTemplate       = `#! /bin/bash
set -x
set -e
while [ ! -f "/etc/kubernetes/kubelet.conf" ]; do
   echo "Waiting for /etc/kubernetes/kubelet.conf to exist"
   sleep 2
done

cp /etc/kubernetes/kubelet.conf {{ .BasePath }}/kubeconfig

if [[ $(grep "/var/lib/kubelet/pki/kubelet-client-current.pem" "{{ .BasePath }}/kubeconfig") ]]; then
	while [ ! -f "/var/lib/kubelet/pki/kubelet-client-current.pem" ]; do
		echo "Waiting for /var/lib/kubelet/pki/kubelet-client-current.pem to exist"
		sleep 2
	done
	cp /var/lib/kubelet/pki/kubelet-client-current.pem {{ .BasePath }}/kubelet-client-current.pem
	chown {{ .Owner }} {{ .BasePath }}/kubelet-client-current.pem
	chmod 400 {{ .BasePath }}/kubelet-client-current.pem
	sed -i 's|/var/lib/kubelet/pki/kubelet-client-current.pem|{{ .BasePath }}/kubelet-client-current.pem|g' {{ .BasePath }}/kubeconfig
fi

chown {{ .Owner }} {{ .BasePath }}/kubeconfig
chmod 400 {{ .BasePath }}/kubeconfig
`
	// Disable ocne.server with a preset file
	// These need to be disabled because the disable presets set by ignition are not
	// showing up in the /etc/systemd/system-preset files.
	// Also enable the service to disable ignition firstboot
	presetFilePathEtc = "/etc/systemd/system-preset/10-ocne.preset"
	presetFilePathLib = "/etc/systemd/system-preset/80-ocne.preset"
	presetFileData    = `disable ocne.service
disable kubeadm.service
disable crio.service
disable kubelet.service
enable keepalived.service
enable ocne-nginx.service
enable ocne-image-cleanup.service
enable ocne-disable-ignition.service
`
)

type copyKubeconfigArguments struct {
	BasePath string
	Owner    string
}

func generateCopyKubeconfigScript(basePath string, owner string) (string, error) {
	return util.TemplateToString(copyKubeconfigScriptTemplate, &copyKubeconfigArguments{
		BasePath: basePath,
		Owner:    owner,
	})
}

// getExtraIgnition creates the ignition string that will be passed to the VM.
func getExtraIgnition(config *types.Config, clusterConfig *types.ClusterConfig, internalLB bool) (string, error) {
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

	patches, err := ignition.KubeadmPatches()
	if err != nil {
		return "", err
	}

	ign := ignition.NewIgnition()

	// Cluster API has its own kubeadm service to start
	// kubelet.  Use that one instead of ocne.service.
	ign = ignition.AddUnit(ign, &igntypes.Unit{
		Name:    "ocne.service",
		Enabled: util.BoolPtr(false),
	})

	// Disable some services via preset file in /etc/systemd/system-preset/10-ocne.preset
	// to override 20-ignition.preset
	presetFileEtc := &ignition.File{
		Path: presetFilePathEtc,
		Mode: 0555,
		Contents: ignition.FileContents{
			Source: presetFileData,
		},
	}
	ignition.AddFile(ign, presetFileEtc)

	// Disable the same services via preset file in /etc/systemd/system-preset/80-ocne.preset
	// make sure the name matches /lib/systemd/system-preset/80-ocne.preset
	presetFileLib := &ignition.File{
		Path: presetFilePathLib,
		Mode: 0555,
		Contents: ignition.FileContents{
			Source: presetFileData,
		},
	}
	ignition.AddFile(ign, presetFileLib)

	// Update service configuration file
	ostreeTransport, registry, tag, err := image.ParseOstreeReference(clusterConfig.OsRegistry)
	if err != nil {
		return "", err
	}
	if tag != "" {
		return "", fmt.Errorf("osRegistry field cannot have a tag")
	}
	updateFile := &ignition.File{
		Path: ignition.OcneUpdateConfigPath,
		Mode: 0400,
		Contents: ignition.FileContents{
			Source: fmt.Sprintf(ignition.OcneUpdateYamlPattern, registry, clusterConfig.OsTag, ostreeTransport),
		},
	}
	ignition.AddFile(ign, updateFile)

	// **NOTE: This is a temporary workaround to enable/start certain services that are
	// enabled in ignition but don't start for some reason.
	// Piggyback on the enable service script to the update service script.
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

	// Merge everything together
	ign = ignition.Merge(ign, container)
	ign = ignition.Merge(ign, proxy)
	ign = ignition.Merge(ign, usr)
	ign = ignition.Merge(ign, patches)

	// If an internal LB is needed for the control plane then the kubeconfig file
	// needs to be copied to /etc/keepalived and /etc/ocne/nginx
	if internalLB {
		// Copy the kubeconfig file needed by keepalived service to get the
		// list of cluster nodes
		copyKubeconfigScriptSource, err := generateCopyKubeconfigScript("/etc/keepalived", "keepalived_script:keepalived_script")
		if err != nil {
			return "", err
		}
		copyKubeconfig := &ignition.File{
			Path: copyKubeconfigScriptPathKeepalived,
			Mode: 0555,
			Contents: ignition.FileContents{
				Source: copyKubeconfigScriptSource,
			},
		}
		ignition.AddFile(ign, copyKubeconfig)

		// Copy the kubeconfig file needed by ocne-nginx service to get the
		// list of cluster nodes
		copyKubeconfigScriptSource, err = generateCopyKubeconfigScript("/etc/ocne/nginx", "nginx_script:nginx_script")
		if err != nil {
			return "", err
		}
		copyKubeconfig = &ignition.File{
			Path: copyKubeconfigScriptPathNginx,
			Mode: 0555,
			Contents: ignition.FileContents{
				Source: copyKubeconfigScriptSource,
			},
		}
		ignition.AddFile(ign, copyKubeconfig)

		// Add drop-in to copy the kubeadm config file for keepalived
		ign = ignition.AddUnit(ign, &igntypes.Unit{
			Name:    ignition.KeepalivedServiceName,
			Enabled: util.BoolPtr(true),
			Dropins: []igntypes.Dropin{
				{
					Name:     "post-kubeadm.conf",
					Contents: util.StrPtr(postKubeadmDropin),
				},
				{
					Name:     copyKubeconfigDropinFile,
					Contents: util.StrPtr(copyKubeconfigDropinKeepalived),
				},
			},
		})

		// Add drop-in to copy the kubeadm config file for ocne-nginx
		ign = ignition.AddUnit(ign, &igntypes.Unit{
			Name:    ignition.NginxServiceName,
			Enabled: util.BoolPtr(true),
			Dropins: []igntypes.Dropin{
				{
					Name:     "post-kubeadm.conf",
					Contents: util.StrPtr(postKubeadmDropin),
				},
				{
					Name:     copyKubeconfigDropinFile,
					Contents: util.StrPtr(copyKubeconfigDropinNginx),
				},
			},
		})

		ign, err = ignition.IgnitionForVirtualIp(ign, config.KubeAPIServerBindPort, config.KubeAPIServerBindPortAlt,
			clusterConfig.VirtualIp, &clusterConfig.Proxy, clusterConfig.Providers.Olvm.ControlPlaneMachine.VirtualMachine.Network.Interface)
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
