// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/cluster/template/oci"
	"github.com/oracle-cne/ocne/pkg/cluster/template/olvm"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
)

func GetTemplate(clusterConfig *types.ClusterConfig) (string, error) {
	var tmpl string
	var err error
	switch *clusterConfig.Provider {
	case constants.ProviderTypeOCI:
		tmpl, err = oci.GetOciTemplate(clusterConfig)
	case constants.ProviderTypeOlvm:
		tmpl, err = olvm.GetOlvmTemplate(clusterConfig)

	default:
		return "", fmt.Errorf("templates not implemented for provider %s", clusterConfig.Provider)
	}
	if err != nil {
		return "", err
	}
	return tmpl, nil
}
