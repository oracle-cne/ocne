// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
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
func Add(kubeconfig string, name string, namespace string, externalUri string, protocol string, friendlyName string) error {
	// Get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return err
	}

	exUrl, err := url.Parse(externalUri)
	if err != nil {
		return err
	}

	hostname := exUrl.Hostname()
	scheme := exUrl.Scheme
	portStr := exUrl.Port()

	if scheme == "file" {
		return addLocalCatalog(name, externalUri, friendlyName)
	}

	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URI scheme %s is not supported", scheme)
	}

	if portStr == "" {
		if scheme == "http" {
			portStr = "80"
		} else if scheme == "https" {
			portStr = "443"
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil
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
				constants.OCNECatalogURIKey:        externalUri,
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

func addLocalCatalog(name string, externalUri string, friendlyName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to get user home diirectory: %v", err)
	}

	catalogDir := filepath.Join(homeDir, ".ocne")
	catalogFile := filepath.Join(catalogDir, name+".yaml")
	catalogData := fmt.Sprintf("name: %s\nuri: %s\nfriendlyName: %s\n", name, externalUri, friendlyName)

	// Write the catalog data to the file
	err = os.WriteFile(catalogFile, []byte(catalogData), 0644)
	if err != nil {
		return fmt.Errorf("unable to write catalog file: %v", err)
	}

	fmt.Printf("Local catalog saved at %s\n", catalogFile)
	return nil

}
