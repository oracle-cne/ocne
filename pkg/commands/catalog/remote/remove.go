// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package remote

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle-cne/ocne/pkg/k8s"
)

type CatalogList struct {
	Catalogs []catalog.CatalogInfo `yaml:"catalogs"`
}

func Remove(kubeconfig string, name string, namespace string) error {
	catInfos, err := ls.Ls(kubeconfig)
	if err != nil {
		return err
	}

	var ci *catalog.CatalogInfo
	for _, c := range catInfos {
		// Handle local catalogs by checking the URI scheme (file://)
		if c.CatalogName == name {
			// Check if it's a local catalog by URI scheme (file://)
			if strings.HasPrefix(c.Uri, "file://") {
				// For local catalogs, we will remove it differently, based on file path
				return removeLocalCatalog(name)
			}

			// For cluster-based catalogs, ensure the namespace matches
			if c.ServiceNsn.Namespace == namespace {
				ci = &c
				break
			}
		}
	}

	if ci == nil {
		return fmt.Errorf("Could not find catalog %s/%s", namespace, name)
	}

	// If not local, delete from the cluster as a service
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	return k8s.DeleteService(kubeClient, ci.ServiceNsn.Namespace, ci.ServiceNsn.Name)
}

// removeLocalCatalog removes a local catalog entry from catalogs.yaml
func removeLocalCatalog(name string) error {
	// Define the path to the catalog file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	catalogFilePath := filepath.Join(homeDir, ".ocne", "catalogs.yaml")

	// Read and parse the catalogs.yaml file
	data, err := os.ReadFile(catalogFilePath)
	if err != nil {
		return fmt.Errorf("failed to read catalog file: %v", err)
	}

	var catalogList CatalogList
	if err := yaml.Unmarshal(data, &catalogList); err != nil {
		return fmt.Errorf("failed to parse catalog file: %v", err)
	}

	// Filter out the catalog to remove
	newCatalogs := []catalog.CatalogInfo{}
	for _, c := range catalogList.Catalogs {
		if c.CatalogName != name {
			newCatalogs = append(newCatalogs, c)
		}
	}
	catalogList.Catalogs = newCatalogs

	// Marshal and write the updated catalog list back to the file
	data, err = yaml.Marshal(&catalogList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated catalog list: %v", err)
	}
	if err := os.WriteFile(catalogFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write updated catalog file: %v", err)
	}

	fmt.Printf("Successfully removed local catalog: %s\n", name)
	return nil

}
