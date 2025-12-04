// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"fmt"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

const (
	keepAlivedUser  = "keepalived_script"
	keepAlivedGroup = "keepalived_script"

	keepAlivedConfigPath         = "/etc/keepalived/keepalived.conf"
	KeepAlivedConfigTemplatePath = "/etc/ocne/keepalived.conf.tmpl"
	KeepAlivedCheckScriptPath    = "/etc/keepalived/check_apiserver.sh"
	KeepAlivedStateScriptPath    = "/etc/keepalived/keepalived_state.sh"

	NginxUser  = "nginx_script"
	NginxGroup = "nginx_script"

	nginxConfigPath         = "/etc/ocne/nginx/nginx.conf"
	nginxConfigTemplatePath = "/etc/ocne/nginx/nginx.conf.tmpl"
	nginxCheckScriptPath    = "/etc/ocne/nginx-refresh/check_nginx.sh"
	nginxPullPath           = "/etc/ocne/nginx/pull_ocne_nginx"
	nginxStartPath          = "/etc/ocne/nginx/start_ocne_nginx"
	nginxImagePath          = "/etc/ocne/nginx/image"

	keepalivedConfigTemplate = `
global_defs {
  router_id LVS_DEVEL
  enable_script_security
}
vrrp_script check_apiserver {
  script "/etc/keepalived/check_apiserver.sh"
  interval 5
  weight 0 # 0 is needed for instance to transition in and out of FAULT state
  fall 10
  rise 2
}
vrrp_instance VI_1 {
  state BACKUP
  interface {{ .Iface }}
  virtual_router_id 51
  priority {{ .Priority }}
  unicast_peer {
{{ .Peers }}
  }
  virtual_ipaddress {
    {{ .VirtualIP }}
  }
  track_script {
    check_apiserver
  }
  notify /etc/keepalived/keepalived_state.sh
}
`

	// If this node is already hosting the VIP, just do the
	// check.  If it is not assigned, go sniffing around to
	// what sort of state is expected.  On first boot, the
	// unicast_peer list will be empty, indicating that
	// keepalived is unconfigured.  If that is the case, do
	// a few checks.  First, do an arping to see if the VIP
	// is assigned somewhere.  If no response is received then
	// assume that no node has the VIP.  If a response is
	// received but it is not possible to curl the endpoint
	// then assume that there is some stale arp cache entry
	// somewhere on the subnet.
	keepalivedCheckScript = `#!/bin/bash
if ! (ip addr | grep -q '{{ .VirtualIP }}/'); then
  if grep -zo 'unicast_peer[[:space:]]*{[[:space:]]*}' {{ .KeepalivedConfig }}; then
    if arping -f -I $(ip route get {{ .VirtualIP }} | cut -d' ' -f3 | head -1) -c 3 {{ .VirtualIP }}; then
      if curl -k https://{{ .VirtualIP }}:{{ .BindPort }}; then
        exit 1
      fi
    fi
  fi
fi

PORTS=$(netstat -nltp)
echo $PORTS | grep -q {{ .BindPort }}
if [ $? -ne 0 ]; then
  echo $(date): keepalived failed to find nginx bound to port >> /etc/keepalived/log
  exit 1
fi
`

	keepAlivedStateScript = `#!/bin/bash
echo $(date): keepalived state: "$@" >> /etc/keepalived/log
`

	nginxConfig = `
load_module /usr/lib64/nginx/modules/ngx_stream_module.so;
events {
  worker_connections 2048;
}
stream {
  upstream backend1 {
    server localhost:{{ .AltPort }} fail_timeout=10s max_fails=1;
    least_conn;
  }
  server {
    listen {{ .BindPort }};
    listen [::]:{{ .BindPort }};
    proxy_pass backend1;
    proxy_connect_timeout 500m;
  }
}
`

	keepalivedRefreshPathUnit = `
[Unit]
Description=Configuration checker for Keepalived

[Path]
PathChanged=%s
Unit=%s

[Install]
WantedBy=multi-user.target
`

	keepalivedRefreshUnit = `
[Unit]
Description=Refresh Keepalived on configuration changes

[Service]
ExecStart=systemctl reload %s
Type=oneshot

[Install]
WantedBy=multi-user.target
`

	nginxRefreshPathUnit = `
[Unit]
Description=Configuration checker for Nginx

[Path]
PathChanged=%s
Unit=%s

[Install]
WantedBy=multi-user.target
`

	nginxRefreshUnit = `
[Unit]
Description=Restart Nginx on configuration changes

[Service]
ExecStart=systemctl reload %s
Type=oneshot
`

	NginxService = `
[Unit]
Description=Nginx load balancer for Kubernetes control plane nodes in OCNE
Wants=network.target
After=network.target
Before=keepalived.service
StartLimitIntervalSec=0

[Service]
ExecStartPre=/etc/ocne/nginx/pull_ocne_nginx
ExecStart=/etc/ocne/nginx/start_ocne_nginx
ExecStop=podman stop ocne-nginx
ExecReload=podman exec ocne-nginx nginx -s reload
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
WantedBy=keepalived.service
`

	nginxPull = `#!/bin/bash

IMAGE=container-registry.oracle.com/olcne/nginx:1.17.7-1

if [ -f "/etc/ocne/nginx/image" ]; then
	. "/etc/ocne/nginx/image"
fi

podman image exists ${IMAGE} || crictl pull ${IMAGE}
exit 0
`

	nginxStart = `#!/bin/bash

IMAGE=container-registry.oracle.com/olcne/nginx:1.17.7-1

if [ -f "/etc/ocne/nginx/image" ]; then
	. "/etc/ocne/nginx/image"
fi

exec podman run --name ocne-nginx --replace --rm --network=host --volume=/etc/ocne/nginx:/etc/nginx ${IMAGE}
`

	nginxImage = "IMAGE=container-registry.oracle.com/olcne/nginx:1.17.7-1"
)

type keepalivedConfigArguments struct {
	Iface     string
	Priority  string
	VirtualIP string
	Peers     string
}

type keepalivedCheckScriptArguments struct {
	BindPort         string
	AltPort          string
	VirtualIP        string
	KeepalivedConfig string
}

type nginxConfigArguments struct {
	BindPort string
	AltPort  string
	Peer     string
}

func GetKeepalivedRefreshUnit() string {
	return fmt.Sprintf(keepalivedRefreshUnit, KeepalivedServiceName)
}

func GetKeepalivedRefreshPathUnit() string {
	return fmt.Sprintf(keepalivedRefreshPathUnit, keepAlivedConfigPath, KeepalivedRefreshServiceName)
}

func GetNginxRefreshUnit() string {
	return fmt.Sprintf(nginxRefreshUnit, NginxServiceName)
}

func GetNginxRefreshPathUnit() string {
	return fmt.Sprintf(nginxRefreshPathUnit, nginxConfigPath, NginxRefreshServiceName)
}

// generateNginxConfig generates a configuration file for nginx to load balance between
// all the master nodes in a given cluster.
func generateNginxConfig(bindPort uint16, altPort uint16, peer string) (string, error) {
	return util.TemplateToString(nginxConfig, &nginxConfigArguments{
		BindPort: fmt.Sprintf("%d", bindPort),
		AltPort:  fmt.Sprintf("%d", altPort),
		Peer:     peer,
	})
}

// generateKeepalivedConfig creates a config file that managed a virtual ip between all kubernetes
// master nodes.
func generateKeepalivedConfig(iface string, priority string, virtualIP string, peers string) (string, error) {
	return util.TemplateToString(keepalivedConfigTemplate, &keepalivedConfigArguments{
		Iface:     iface,
		Priority:  priority,
		VirtualIP: virtualIP,
		Peers:     peers,
	})
}

// GenerateKeepalivedCheckScript creates a check script for keepalived that monitors the kubernetes master on the node
// on which keepalived is installed.  This script is configured by default in the text
// generated by generateKeepalivedConfig
func GenerateKeepalivedCheckScript(bindPort uint16, altPort uint16, virtualIP string) (string, error) {
	return util.TemplateToString(keepalivedCheckScript, &keepalivedCheckScriptArguments{
		BindPort:         fmt.Sprintf("%d", bindPort),
		AltPort:          fmt.Sprintf("%d", altPort),
		VirtualIP:        virtualIP,
		KeepalivedConfig: keepAlivedConfigPath,
	})
}

type IgnitionData struct {
	Files []*File
	Units []*igntypes.Unit
}

// GenerateAssetsForVirtualIp generates file and systemd unit contents for configuring control plane HA using a virtual IP
func GenerateAssetsForVirtualIp(bindPort uint16, altPort uint16, virtualIP string, proxy *types.Proxy, netInterface string) (*IgnitionData, error) {
	data := &IgnitionData{
		Files: []*File{},
		Units: []*igntypes.Unit{},
	}

	keepAlivedConfig, err := generateKeepalivedConfig(netInterface, "50", virtualIP, "")
	if err != nil {
		return nil, err
	}

	data.Files = append(data.Files,
		&File{
			Path: keepAlivedConfigPath,
			Mode: 0644,
			Contents: FileContents{
				Source: keepAlivedConfig,
			},
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
		})

	keepAlivedCheckScript, err := GenerateKeepalivedCheckScript(bindPort, altPort, virtualIP)
	if err != nil {
		return nil, err
	}
	data.Files = append(data.Files,
		&File{
			Path:  KeepAlivedCheckScriptPath,
			Mode:  0755,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: keepAlivedCheckScript,
			},
		},
		&File{
			Path:  KeepAlivedStateScriptPath,
			Mode:  0755,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: keepAlivedStateScript,
			},
		},
		&File{
			Path:  "/etc/keepalived/log",
			Mode:  0644,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: "",
			},
		})

	nginxConfigSource, err := generateNginxConfig(bindPort, altPort, fmt.Sprintf("    server localhost:%d;", altPort))
	if err != nil {
		return nil, err
	}

	data.Files = append(data.Files,
		&File{
			Path:  nginxConfigPath,
			Mode:  0644,
			User:  NginxUser,
			Group: NginxGroup,
			Contents: FileContents{
				Source: nginxConfigSource,
			},
		},
		&File{
			Path: nginxPullPath,
			Mode: 0755,
			Contents: FileContents{
				Source: nginxPull,
			},
		},
		&File{
			Path: nginxStartPath,
			Mode: 0755,
			Contents: FileContents{
				Source: nginxStart,
			},
		},
		&File{
			Path: nginxImagePath,
			Mode: 0644,
			Contents: FileContents{
				Source: nginxImage,
			},
		})

	// services don't start unless they are included in the units in ostree
	nginxUnit := &igntypes.Unit{
		Name:     NginxServiceName,
		Enabled:  util.BoolPtr(true),
		Contents: util.StrPtr(NginxService),
	}
	// add a proxy dropin to ocne-nginx if proxy is configured
	if proxy != nil && (len(proxy.HttpProxy) > 0 || len(proxy.HttpsProxy) > 0 || len(proxy.NoProxy) > 0) {
		proxyConf, err := util.TemplateToString(ProxyDropinPattern, proxy)
		if err != nil {
			return nil, err
		}
		nginxUnit.Dropins = []igntypes.Dropin{
			{
				Name:     "proxy.conf",
				Contents: util.StrPtr(proxyConf),
			},
		}
	}
	nginxRefreshUnit := &igntypes.Unit{
		Name:     NginxRefreshServiceName,
		Enabled:  util.BoolPtr(true),
		Contents: util.StrPtr(fmt.Sprintf(nginxRefreshUnit, NginxServiceName)),
	}
	nginxRefreshPathUnit := &igntypes.Unit{
		Name: NginxRefreshPathName,
		Enabled: util.BoolPtr(true),
		Contents: util.StrPtr(fmt.Sprintf(nginxRefreshPathUnit, nginxConfigPath, NginxRefreshServiceName)),
	}
	keepAlivedUnit := &igntypes.Unit{
		Name:    KeepalivedServiceName,
		Enabled: util.BoolPtr(true),
	}
	keepAlivedRefreshUnit := &igntypes.Unit{
		Name: KeepalivedRefreshServiceName,
		Enabled: util.BoolPtr(true),
		Contents: util.StrPtr(fmt.Sprintf(keepalivedRefreshUnit, KeepalivedServiceName)),
	}
	keepAlivedRefreshPathUnit := &igntypes.Unit{
		Name: KeepalivedRefreshPathName,
		Enabled: util.BoolPtr(true),
		Contents: util.StrPtr(fmt.Sprintf(keepalivedRefreshPathUnit, keepAlivedConfigPath, KeepalivedServiceName)),
	}

	data.Units = append(data.Units, nginxUnit, nginxRefreshUnit, nginxRefreshPathUnit, keepAlivedUnit, keepAlivedRefreshUnit, keepAlivedRefreshPathUnit)

	return data, nil
}

// IgnitionForVirtualIp add keepalived and nginx services and its files to ignition
func IgnitionForVirtualIp(ign *igntypes.Config, bindPort uint16, altPort uint16, virtualIP string, proxy *types.Proxy, netInterface string) (*igntypes.Config, error) {
	// Setup nginx_script user. This user may eventually be part of the base ock image;
	// however, it is being created here for compatibility with existing ock images.
	err := AddGroup(ign, &Group{
		Name:   "nginx_script",
		System: true,
	})
	if err != nil && !strings.Contains(err.Error(), "already defined") {
		return nil, err
	}
	err = AddUser(ign, &User{
		Name:         "nginx_script",
		PrimaryGroup: "nginx_script",
		Shell:        "/sbin/nologin",
		System:       true,
		NoCreateHome: true,
	})
	if err != nil && !strings.Contains(err.Error(), "already defined") {
		return nil, err
	}

	data, err := GenerateAssetsForVirtualIp(bindPort, altPort, virtualIP, proxy, netInterface)
	if err != nil {
		return nil, err
	}

	for _, file := range data.Files {
		if err := AddFile(ign, file); err != nil {
			return nil, err
		}
	}
	for _, unit := range data.Units {
		ign = AddUnit(ign, unit)
	}

	return ign, nil
}
