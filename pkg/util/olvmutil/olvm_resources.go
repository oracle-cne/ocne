// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvmutil

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
)

func CredSecretNsn(cc *types.ClusterConfig) *types.NamespacedName {
	defName := fmt.Sprintf("%s-%s", cc.Name, constants.OLVMOVirtCredSecretSuffix)
	nn := cc.Providers.Olvm.OlvmAPIServer.CredentialsSecret
	if nn.Name == "" {
		nn.Name = defName
	}
	if nn.Namespace == "" {
		nn.Namespace = cc.Providers.Olvm.Namespace
	}
	return &nn
}

func CaConfigMapNsn(cc *types.ClusterConfig) *types.NamespacedName {
	defName := fmt.Sprintf("%s-%s", cc.Name, constants.OLVMOVirtCAConfigMapSuffix)
	nn := cc.Providers.Olvm.OlvmAPIServer.CAConfigMap
	if nn.Name == "" {
		nn.Name = defName
	}
	if nn.Namespace == "" {
		nn.Namespace = cc.Providers.Olvm.Namespace
	}
	return &nn
}
