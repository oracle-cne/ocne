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
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
)

type CatalogList struct {
	Catalogs []catalog.CatalogInfo `yaml:"catalogs"`
}

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
	// Get local catalogs from ~/.ocne/catalogs.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve home directory: %v", err)
	}
	catalogFilePath := filepath.Join(homeDir, ".ocne", "catalogs.yaml")

	// Check if local catalogs file exists
	if _, err := os.Stat(catalogFilePath); err == nil {
		// Read and parse catalogs.yaml
		data, err := ioutil.ReadFile(catalogFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read local catalogs file: %v", err)
		}

		var localCatalogs CatalogList
		if err := yaml.Unmarshal(data, &localCatalogs); err != nil {
			return nil, fmt.Errorf("failed to parse local catalogs file: %v", err)
		}

		// Append local catalogs to the catalog list
		catalogs = append(catalogs, localCatalogs.Catalogs...)
	}

	return catalogs, nil

}
