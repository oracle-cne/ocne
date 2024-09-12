// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package template

import (
	"github.com/oracle-cne/ocne/pkg/cluster/template"
	"github.com/oracle-cne/ocne/pkg/config/types"
)

// TemplateOptions are the options for the cluster template command
type TemplateOptions struct {
	// Config is the OCNE configuration
	Config types.Config

	// ClusterConfig is the cluster configuration
	ClusterConfig types.ClusterConfig
}

func Template(opt TemplateOptions) (string, error) {
	return template.GetTemplate(&opt.Config, &opt.ClusterConfig)
}
