// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package driver

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/config/types"
	log "github.com/sirupsen/logrus"
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
	Stage(string) (string, string, bool, error)
}

type DriverCreateFunc func(*types.Config, *types.ClusterConfig) (ClusterDriver, error)

var drivers = map[string]DriverCreateFunc{}

func RegisterDriver(name string, ftor DriverCreateFunc) {
	drivers[name] = ftor
}

func CreateDriver(config *types.Config, clusterConfig *types.ClusterConfig) (ClusterDriver, error) {
	log.Debugf("Validating customCertificates support for provider %s", clusterConfig.Provider)
	if clusterConfig.Provider != "byo" && (clusterConfig.CustomCertificates != nil || config.CustomCertificates != nil) {
		return nil, fmt.Errorf("customCertificates is supported only by the byo provider; it must be omitted from cluster configuration and defaults for provider %q", clusterConfig.Provider)
	}
	ftor, ok := drivers[clusterConfig.Provider]
	if !ok {
		return nil, fmt.Errorf("No implementation exists for the %s driver", clusterConfig.Provider)
	}

	return ftor(config, clusterConfig)
}
