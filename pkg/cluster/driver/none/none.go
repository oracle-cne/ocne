// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package none

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

const (
	DriverName = "none"
)

type NoneDriver struct {
	KubeConfig string
}

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (driver.ClusterDriver, error) {
	if clusterConfig.Provider == "none" && config.KubeConfig == "" {
		return nil, fmt.Errorf("When the none provider is used, an existing kubeconfig file must be specified")
	}
	ret := &NoneDriver{
		KubeConfig: config.KubeConfig,
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
	return ""

}

func (nd *NoneDriver) PostInstallHelpStanza() string {
	return ""

}

func (nd *NoneDriver) DefaultCNIInterfaces() []string {
	return []string{""}
}
