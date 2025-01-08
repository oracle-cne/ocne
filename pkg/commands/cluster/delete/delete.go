// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delete

import (
	"github.com/oracle-cne/ocne/pkg/cluster/cache"
	"github.com/oracle-cne/ocne/pkg/cluster/driver"
	"github.com/oracle-cne/ocne/pkg/config/types"
	log "github.com/sirupsen/logrus"
)

type DeleteOptions struct {
	Config         *types.Config
	ClusterConfig  *types.ClusterConfig
	KubeConfigPath string
	Provider       string
	Name           string
	SessionURI     string
}

func Delete(clusterConfig *types.ClusterConfig) error {

	drv, err := driver.CreateDriver(clusterConfig)
	if err != nil {
		return err
	}

	log.Debugf("Deleting cluster %s", clusterConfig.Name)
	err = drv.Delete()
	if err != nil {
		return err
	}

	clusterCache, err := cache.GetCache()
	if err != nil {
		return err
	}

	err = clusterCache.Delete(*clusterConfig.Name)
	if err != nil {
		return err
	}

	return nil
}
