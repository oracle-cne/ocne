// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"github.com/oracle-cne/ocne/pkg/cluster/template/oci"
	"github.com/oracle-cne/ocne/pkg/cluster/template/olvm"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
)

const (
	// Kubernetes Gateway API yaml sourced from https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml
	kubernetesGatewayAPIYAML = "gateway-api/v1.2.1/standard-install.yaml"
)

func GetTemplate(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
	var tmpl string
	var err error
	switch clusterConfig.Provider {
	case constants.ProviderTypeOCI:
		tmpl, err = oci.GetOciTemplate(config, clusterConfig)
	case constants.ProviderTypeOlvm:
		tmpl, err = olvm.GetOlvmTemplate(config, clusterConfig)

	default:
		return "", fmt.Errorf("templates not implemented for provider %s", clusterConfig.Provider)
	}
	if err != nil {
		return "", err
	}
	return tmpl, nil
}

func GetKubernetesGatewayApiTemplate() (string, error) {
	var err error

	tmplBytes, err := template.ReadTemplate(kubernetesGatewayAPIYAML)
	if err != nil {
		return "", err
	}

	return string(tmplBytes), err
}
