// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package remote

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
	"github.com/oracle-cne/ocne/pkg/k8s/client"

	"github.com/oracle-cne/ocne/pkg/k8s"
)

func Remove(kubeconfig string, name string, namespace string) error {
	catInfos, err := ls.Ls(kubeconfig)
	if err != nil {
		return err
	}

	var ci *catalog.CatalogInfo
	for _, c := range catInfos {
		if c.CatalogName == name && c.ServiceNsn.Namespace == namespace {
			ci = &c
			break
		}
	}
	if ci == nil {
		return fmt.Errorf("Could not find catalog %s/%s", namespace, name)
	}

	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	return k8s.DeleteService(kubeClient, ci.ServiceNsn.Namespace, ci.ServiceNsn.Name)
}
