// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add adds a catalog to the cluster.  It is assumed that all
// catalogs added this way are external services rather than
// in-cluster catalogs.  In-cluster catalogs should be added
// by manually adding the appropriate Service to the cluster
// as part of the deployment itself.

// CatalogList defines the structure of catalogs.yaml
type CatalogList struct {
	Catalogs []catalog.CatalogInfo `yaml:"catalogs"`
}

// Add adds a catalog to the cluster or local system based on URI scheme
func Add(kubeconfig string, name string, namespace string, externalUri string, protocol string, friendlyName string) error {
	// Parse the URI
	exUrl, err := url.Parse(externalUri)
	if err != nil {
		return err
	}

	switch exUrl.Scheme {
	case "file":
		return addLocalCatalog(name, namespace, externalUri, protocol)
	case "http", "https":
		return addClusterCatalog(kubeconfig, name, namespace, exUrl, protocol, friendlyName)
	default:
		return fmt.Errorf("URI scheme %s is not supported", exUrl.Scheme)
	}
}

// addLocalCatalog adds a local catalog to ~/.ocne/catalogs.yaml
func addLocalCatalog(name string, namespace string, uri string, protocol string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	catalogFilePath := filepath.Join(homeDir, ".ocne", "catalogs.yaml")

	// ~/.ocne directory check
	catalogDir := filepath.Dir(catalogFilePath)
	if _, err := os.Stat(catalogDir); os.IsNotExist(err) {
		if err := os.MkdirAll(catalogDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", catalogDir, err)
		}
	}

	// Initialize or read existing catalogs.yaml
	var catalogList CatalogList
	if _, err := os.Stat(catalogFilePath); os.IsNotExist(err) {
		// If file doesn't exist, create an empty catalog list
		catalogList = CatalogList{}
	} else {
		// Read and unmarshal existing catalogs.yaml
		data, err := os.ReadFile(catalogFilePath)
		if err != nil {
			return fmt.Errorf("failed to read catalog file: %v", err)
		}
		if err := yaml.Unmarshal(data, &catalogList); err != nil {
			return fmt.Errorf("failed to parse catalog file: %v", err)
		}
	}

	catalogEntry := catalog.CatalogInfo{
		CatalogName: name,
		Namespace:   namespace,
		Uri:         uri,
		Protocol:    protocol,
		Scheme:      "file",
	}
	catalogList.Catalogs = append(catalogList.Catalogs, catalogEntry)

	// Write updated catalog list back to file
	data, err := yaml.Marshal(&catalogList)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog list: %v", err)
	}
	if err := os.WriteFile(catalogFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write catalog file: %v", err)
	}
	return nil
}

// addClusterCatalog creates a cluster service to represent the catalog
func addClusterCatalog(kubeconfig string, name string, namespace string, exUrl *url.URL, protocol string, friendlyName string) error {
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}

	hostname := exUrl.Hostname()
	scheme := exUrl.Scheme
	portStr := exUrl.Port()

	if portStr == "" {
		if scheme == "http" {
			portStr = "80"
		} else if scheme == "https" {
			portStr = "443"
		} else {
			return fmt.Errorf("URI scheme %s is not supported", scheme)
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %v", err)
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				constants.OCNECatalogLabelKey: "",
			},
			Annotations: map[string]string{
				constants.OCNECatalogAnnotationKey: friendlyName,
				constants.OCNECatalogURIKey:        exUrl.String(),
				constants.OCNECatalogProtoKey:      protocol,
			},
		},
		Spec: v1.ServiceSpec{
			Type:         "ExternalName",
			ExternalName: hostname,
			Ports: []v1.ServicePort{
				{
					Name:     scheme,
					Protocol: v1.ProtocolTCP,
					Port:     int32(port),
				},
			},
		},
	}

	return k8s.CreateService(kubeClient, svc)
}
