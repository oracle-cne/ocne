// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package driver

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

type ClusterDriver interface {
	Start() (bool, bool, error)
	PostStart() error
	Stop() error
	Join(string, int, int) error
	Delete() error
	Close() error
	GetKubeconfigPath() string
	GetKubeAPIServerAddress() string
	PostInstallHelpStanza() string
	DefaultCNIInterfaces() []string
}

type DriverCreateFunc func(*types.Config, *types.ClusterConfig) (ClusterDriver, *cache.ClusterCache, error)

var drivers = map[string]DriverCreateFunc{}

func RegisterDriver(name string, ftor DriverCreateFunc) {
	drivers[name] = ftor
}

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (ClusterDriver, *cache.ClusterCache, error) {
	ftor, ok := drivers[clusterConfig.Provider]
	if !ok {
		return nil, nil, fmt.Errorf("No implementation exists for the %s driver", clusterConfig.Provider)
	}

	return ftor(config, clusterConfig)
}
