// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ls

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
)

// Ls gets a logs of catalogs
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
	// Add local catalogs from ~/.ocne/catalogs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to access home directory: %v", err)
	}

	localCatalogPath := filepath.Join(homeDir, ".ocne", "catalogs")
	files, err := os.ReadDir(localCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read local catalogs: %v", err)
	}

	// Parse each file as a local catalog
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(localCatalogPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Failed to read catalog file %s: %v\n", filePath, err)
			continue
		}

		var localCatalog catalog.CatalogInfo
		if err := yaml.Unmarshal(content, &localCatalog); err != nil {
			fmt.Printf("Failed to parse catalog file %s: %v\n", filePath, err)
			continue
		}

		// Add local catalog to the list
		catalogs = append(catalogs, localCatalog)
	}

	return catalogs, nil

}
