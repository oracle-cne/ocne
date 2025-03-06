// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func GetServerVersion(restConf *rest.Config) (*version.Info, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(restConf)
	if err != nil {
		return nil, err
	}

	inf, err := dc.ServerVersion()
	if err != nil {
		return nil, err
	}
	return inf, err
}

func VersionInfoToString(v *version.Info) string {
	return fmt.Sprintf("%s.%s.%d", v.Major, v.Minor, 0)
}
