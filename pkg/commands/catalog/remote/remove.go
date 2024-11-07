// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package remote

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"

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
		if c.CatalogName == name && c.ServiceNsn.Namespace == namespace {
			ci = &c
			break
		}
	}
	if ci == nil {
		return fmt.Errorf("Could not find catalog %s/%s", namespace, name)
	}

	// Check if the catalog is local (file:// URI)
	if ci.Uri != "" && filepath.IsAbs(ci.Uri[7:]) {
		return removeLocalCatalog(ci)
	}

	// If not local, delete from the cluster as a service
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}
	return k8s.DeleteService(kubeClient, ci.ServiceNsn.Namespace, ci.ServiceNsn.Name)
}

// removeLocalCatalog removes a local catalog entry from catalogs.yaml
func removeLocalCatalog(ci *catalog.CatalogInfo) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to retrieve home directory: %v", err)
	}
	catalogFilePath := filepath.Join(homeDir, ".ocne", "catalogs.yaml")

	// Read the existing local catalogs from catalogs.yaml
	data, err := ioutil.ReadFile(catalogFilePath)
	if err != nil {
		return fmt.Errorf("failed to read local catalogs file: %v", err)
	}

	var catalogList CatalogList
	if err := yaml.Unmarshal(data, &catalogList); err != nil {
		return fmt.Errorf("failed to parse local catalogs file: %v", err)
	}

	// Filter out the catalog that matches the name and URI
	var updatedCatalogs []catalog.CatalogInfo
	for _, c := range catalogList.Catalogs {
		if c.CatalogName != ci.CatalogName || c.Uri != ci.Uri {
			updatedCatalogs = append(updatedCatalogs, c)
		}
	}
	catalogList.Catalogs = updatedCatalogs

	// Write the updated list back to catalogs.yaml
	updatedData, err := yaml.Marshal(&catalogList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated catalogs: %v", err)
	}
	if err := ioutil.WriteFile(catalogFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated catalogs file: %v", err)
	}

	fmt.Printf("Removed local catalog %s from %s\n", ci.CatalogName, catalogFilePath)
	return nil
}
