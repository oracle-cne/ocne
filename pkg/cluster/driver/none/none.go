// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package none

import (
	"fmt"
	"net"
	"strings"

	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
)

const (
	DriverName = "none"
)

type NoneDriver struct {
	KubeConfig string
	KubeAPIServerIP string
}

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
	if clusterConfig.Provider == constants.ProviderTypeNone && config.KubeConfig == "" {
		return nil, fmt.Errorf("When the none provider is used, an existing kubeconfig file must be specified")
	}

	restConfig, _, err := client.GetKubeClient(config.KubeConfig)
	if err != nil {
		return nil, err
	}

	controlPlaneEndpoint, err := k8s.GetControlPlaneEndpoint(restConfig)
	if err != nil {
		return nil, err
	}

	kubeAPIServerIP, _, err := net.SplitHostPort(controlPlaneEndpoint)
	if err != nil {
		// Maybe there is no port.  If that is the
		// error, assume that the value is a valid
		// address.  If it's not, the cluster has
		// bigger problems.
		if !strings.Contains(err.Error(), "missing port in address") {
			return nil, err
		}
		kubeAPIServerIP = controlPlaneEndpoint
	} else {
		// net.SplitHostPort removes any "[]" from IPv6 addresses.
		// Add them back.
		kubeAPIServerIP = util.GetURIAddress(kubeAPIServerIP)
	}

	ret := &NoneDriver{
		KubeConfig:      config.KubeConfig,
		KubeAPIServerIP: kubeAPIServerIP,
	}
	return ret, nil
}

func (nd *NoneDriver) Start() (bool, bool, error) {
	return true, false, nil
}

func (nd *NoneDriver) PostStart() error {
	return nil
}

func (nd *NoneDriver) Stop() error {
	return nil
}

func (nd *NoneDriver) Join(string, int, int) error {
	return nil
}

func (nd *NoneDriver) Delete() error {
	return nil
}

func (nd *NoneDriver) Close() error {
	return nil
}

func (nd *NoneDriver) GetKubeconfigPath() string {
	return nd.KubeConfig
}

func (nd *NoneDriver) GetKubeAPIServerAddress() string {
	return nd.KubeAPIServerIP
}

func (nd *NoneDriver) PostInstallHelpStanza() string {
	return ""
}

func (nd *NoneDriver) DefaultCNIInterfaces() []string {
	return []string{""}
}

// Stage is a no-op
func (nd *NoneDriver) Stage(version string) (string, string, bool, error) {
	return "", "", true, nil
}
