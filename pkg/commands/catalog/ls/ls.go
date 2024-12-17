// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ls

import (
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/types"
)

// Ls - return the list of catalogs
func Ls(kubeconfig string) ([]catalog.CatalogInfo, error) {
	// Get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	// search by key only, the value is the name of the catalog
	serviceList, err := k8s.FindServicesByLabelKey(kubeClient, "", constants.OCNECatalogLabelKey)
	if err != nil {
		return nil, err
	}

	// load the catalogInfo list
	var catalogs []catalog.CatalogInfo
	for _, service := range serviceList.Items {
		annot := service.ObjectMeta.Annotations
		scheme := ""
		port := int32(0)

		if len(service.Spec.Ports) > 0 {
			scheme = service.Spec.Ports[0].Name
			port = service.Spec.Ports[0].Port
		}

		catalogs = append(catalogs, catalog.CatalogInfo{
			ServiceNsn: types.NamespacedName{
				Namespace: service.Namespace,
				Name:      service.Name,
			},
			CatalogName: annot[constants.OCNECatalogAnnotationKey],
			Uri:         annot[constants.OCNECatalogURIKey],
			Protocol:    annot[constants.OCNECatalogProtoKey],
			Port:        port,
			Scheme:      scheme,
			Hostname:    service.Spec.ExternalName,
			Type:        service.Spec.Type,
		})

	}
	// Always add the catalog embedded in the CLI
	catalogs = append(catalogs, catalog.CatalogInfo{
		CatalogName: catalog.InternalCatalog,
		Protocol:    catalog.HelmProtocol,
	})

	return catalogs, nil
}
