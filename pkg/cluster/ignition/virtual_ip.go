// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"fmt"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

const (
	keepAlivedServiceName = "keepalived.service"
	keepAlivedUser        = "keepalived_script"
	keepAlivedGroup       = "keepalived_script"

	nginxServiceName = "ocne-nginx.service"

	keepAlivedConfigPath         = "/etc/keepalived/keepalived.conf"
	KeepAlivedConfigTemplatePath = "/etc/ocne/keepalived.conf.tmpl"
	keepAlivedCheckScriptPath    = "/etc/keepalived/check_apiserver.sh"
	keepAlivedStateScriptPath    = "/etc/keepalived/keepalived_state.sh"

	nginxConfigPath         = "/etc/ocne/nginx/nginx.conf"
	nginxConfigTemplatePath = "/etc/ocne/nginx/nginx.conf.tmpl"
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

	keepalivedCheckScript = `#!/bin/bash
errorExit() {
  echo "*** $*" 2>&2
  exit 1
}

# update keepalived.conf/nginx.conf and restart keepalived.service/ocne-nginx.service if necessary
refreshServices() {
  NODES=$(KUBECONFIG=/etc/keepalived/kubeconfig kubectl --server=https://localhost:{{ .AltPort }} --tls-server-name=$(hostname) get nodes --request-timeout 1m --no-headers --selector 'node-role.kubernetes.io/control-plane' -o wide | awk -v OFS='\t\t' '{print $6}')
  if [ $? -ne 0 ]; then
    return 0
  fi

  if [ -z "${NODES}" ]; then
	NODES="localhost"
  fi

  # check if the existing peers is the same as NODES
  if [[ "${NODES}" != "$(cat /etc/keepalived/peers)" ]]; then
    echo $(date): keepalived peers have been changed to: $NODES >> /etc/keepalived/log
    echo "$NODES" > /etc/keepalived/peers

    ADDRS=$(/usr/sbin/ip addr)
    ADDRS="$ADDRS localhost"
    ESCAPED_PEERS=""
    for node in $NODES; do
      echo "$ADDRS" | grep -q "$node"
      if [ $? -eq 0 ]; then
        continue
      fi
      ESCAPED_PEERS="$ESCAPED_PEERS\n    $node"
    done

    sed -e 's/PEERS/'"$ESCAPED_PEERS"'/g' /etc/ocne/keepalived.conf.tmpl > /etc/keepalived/keepalived.conf

    systemctl reload keepalived.service &
  fi

  # check if the existing servers is the same as NODES
  if [[ "${NODES}" != "$(cat /etc/ocne/nginx/servers)" ]]; then
    echo $(date): ocne-nginx servers have been changed to: $NODES >> /etc/keepalived/log
    echo "$NODES" > /etc/ocne/nginx/servers

    ESCAPED_SERVERS=""
    for node in $NODES; do
      if echo "$node" | grep ':'; then
        node="[$node]"
      fi
      ESCAPED_SERVERS="$ESCAPED_SERVERS\n    server $node:{{ .AltPort }};"
    done

    sed -e 's/SERVERS/'"$ESCAPED_SERVERS"'/g' /etc/ocne/nginx/nginx.conf.tmpl > /etc/ocne/nginx/nginx.conf

    systemctl restart ocne-nginx.service &
  fi
}

refreshServices
curl --silent --max-time 2 --insecure https://localhost:{{ .BindPort }}/ -o /dev/null || errorExit "Error GET https://localhost:{{ .BindPort }}/"
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
{{ .Peer }}
    least_conn;
  }
  server {
    listen {{ .BindPort }};
    listen [::]:{{ .BindPort }};
    proxy_pass backend1;
  }
}
`

	nginxService = `
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

	polkitRules = `
// Allow keepalived_script user to restart services
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.manage-units" &&
        (action.lookup("unit") == "keepalived.service" || action.lookup("unit") == "ocne-nginx.service") &&
        subject.user == "keepalived_script") {
        return polkit.Result.YES;
    }
});
`
)

type keepalivedConfigArguments struct {
	Iface     string
	Priority  string
	VirtualIP string
	Peers     string
}

type keepalivedCheckScriptArguments struct {
	BindPort string
	AltPort  string
}

type nginxConfigArguments struct {
	BindPort string
	Peer     string
}

// generateNginxConfig generates a configuration file for nginx to load balance between
// all the master nodes in a given cluster.
func generateNginxConfig(bindPort uint16, peer string) (string, error) {
	return util.TemplateToString(nginxConfig, &nginxConfigArguments{
		BindPort: fmt.Sprintf("%d", bindPort),
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

// generateKeepalivedCheckScript creates a check script for keepalived that monitors the kubernetes master on the node
// on which keepalived is installed.  This script is configured by default in the text
// generated by generateKeepalivedConfig
func generateKeepalivedCheckScript(bindPort uint16, altPort uint16) (string, error) {
	return util.TemplateToString(keepalivedCheckScript, &keepalivedCheckScriptArguments{
		BindPort: fmt.Sprintf("%d", bindPort),
		AltPort:  fmt.Sprintf("%d", altPort),
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

	keepAlivedConfigTemplate, err := generateKeepalivedConfig(netInterface, "50", virtualIP, "PEERS")
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
		},
		&File{
			Path: KeepAlivedConfigTemplatePath,
			Mode: 0644,
			Contents: FileContents{
				Source: keepAlivedConfigTemplate,
			},
		})

	keepAlivedCheckScript, err := generateKeepalivedCheckScript(bindPort, altPort)
	if err != nil {
		return nil, err
	}
	data.Files = append(data.Files,
		&File{
			Path:  keepAlivedCheckScriptPath,
			Mode:  0755,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: keepAlivedCheckScript,
			},
		},
		&File{
			Path:  keepAlivedStateScriptPath,
			Mode:  0755,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: keepAlivedStateScript,
			},
		},
		&File{
			Path:  "/etc/keepalived/peers",
			Mode:  0644,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: "",
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

	nginxConfig, err := generateNginxConfig(bindPort, fmt.Sprintf("    server localhost:%d;", altPort))
	if err != nil {
		return nil, err
	}

	nginxConfigTemplate, err := generateNginxConfig(bindPort, "SERVERS")
	if err != nil {
		return nil, err
	}

	data.Files = append(data.Files,
		&File{
			Path:  nginxConfigPath,
			Mode:  0644,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: nginxConfig,
			},
		},
		&File{
			Path: nginxConfigTemplatePath,
			Mode: 0644,
			Contents: FileContents{
				Source: nginxConfigTemplate,
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
		},
		&File{
			Path:  "/etc/ocne/nginx/servers",
			Mode:  0644,
			User:  keepAlivedUser,
			Group: keepAlivedGroup,
			Contents: FileContents{
				Source: "",
			},
		},
		&File{
			Path: "/etc/polkit-1/rules.d/51-keepalived.rules",
			Mode: 0644,
			Contents: FileContents{
				Source: polkitRules,
			},
		})

	// services don't start unless they are included in the units in ostree
	nginxUnit := &igntypes.Unit{
		Name:     nginxServiceName,
		Enabled:  util.BoolPtr(true),
		Contents: util.StrPtr(nginxService),
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
	keepAlivedUnit := &igntypes.Unit{
		Name:    keepAlivedServiceName,
		Enabled: util.BoolPtr(true),
	}

	data.Units = append(data.Units, nginxUnit, keepAlivedUnit)

	return data, nil
}

// IgnitionForVirtualIp add keepalived and nginx services and its files to ignition
func IgnitionForVirtualIp(ign *igntypes.Config, bindPort uint16, altPort uint16, virtualIP string, proxy *types.Proxy, netInterface string) (*igntypes.Config, error) {
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
