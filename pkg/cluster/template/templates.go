// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"embed"
	"fmt"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
)

//go:embed all:templates
var templates embed.FS

func getTemplate(name string) ([]byte, error) {
	return templates.ReadFile(fmt.Sprintf("templates/%s", name))
}

func GetTemplate(config *types.Config, clusterConfig *types.ClusterConfig) (string, error) {
	var tmpl string
	var err error
	switch clusterConfig.Provider {
	case constants.ProviderTypeOCI:
		tmpl, err = GetOciTemplate(config, clusterConfig)
	default:
		return "", fmt.Errorf("templates not implemented for provider %s", clusterConfig.Provider)
	}
	if err != nil {
		return "", err
	}
	return tmpl, nil
}
