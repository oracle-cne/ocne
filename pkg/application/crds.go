// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package application

import (
	apiexv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

func CRDsForApplication(name string, namespace string, kubeConfigPath string) ([]*apiexv1.CustomResourceDefinition, error) {
	// This function can look for CRDs that were left behind after an
	// uninstall.  It works even if the application is not installed.
	kubeInfo, err := client.CreateKubeInfo(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	crds, err := k8s.GetCRDsWithAnnotations(kubeInfo.RestConfig, GetAnnotations(name, namespace))
	if err != nil {
		return nil, err
	}

	return crds, nil
}
