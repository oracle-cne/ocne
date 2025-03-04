// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"

	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

const (
	HelmProtocol        = "helm"
	ArtifacthubProtocol = "artifacthub"
	InternalCatalog     = "embedded"
)

// NewConnection creates a connection to the desired catalog.
func NewConnection(kubeconfig string, ci *CatalogInfo) (CatalogConnection, error) {
	// An internal catalog is baked into the application.  It has no
	// protocol, but it does have a magic name.
	if ci.CatalogName == InternalCatalog {
		return NewInternalConnection(kubeconfig, ci)
	}

	switch ci.Protocol {
	case HelmProtocol:
		return NewHelmConnection(kubeconfig, ci)
	case ArtifacthubProtocol:
		return NewArtifacthubConnection(kubeconfig, ci)
	}

	return nil, fmt.Errorf("protocol %s is not implemented", ci.Protocol)
}

// getCatalogURI builds a connection URI from the catalog info.  In the
// case where the catalog requires a tunnel into the cluster, this function
// opens that tunnel as well.
func getCatalogURI(kubeconfig string, ci *CatalogInfo) (string, error) {
	// If the service is an ExternalName, then there
	// is no need to open a tunnel.  It's not strictly
	// necessary to open a tunnel to LoadBalancer or NodePort
	// services either, but that adds a bunch of complexity
	// that is not necessary
	if ci.Type == v1.ServiceTypeExternalName {
		return ci.Uri, nil
	}

	// Open a tunnel
	if ci.Port == 0 {
		return "", fmt.Errorf("%s does not have a port assigned", ci.CatalogName)
	}

	localPort, err := k8s.PortForwardToService(kubeconfig, ci.ServiceNsn, strconv.Itoa(int(ci.Port)))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://127.0.0.1:%d%s", ci.Scheme, localPort, ci.Uri), nil
}

// getCanonicalURI takes a two URIs and figures out what the "real URI" is.
// If the second URI is a relative path, it is appended to the first URI.
// If it is an absolute path, then it is simply returned.
func getCanonicalURI(absolute string, relative string) string {
	if strings.HasPrefix(relative, "http") {
		return relative
	}
	return fmt.Sprintf("%s%s", absolute, relative)
}

// findChartAtVersion looks at a Catalog and finds the named chart at the
// desired version.  If the version string is empty, then the latest version
// is returned.
func findChartAtVersion(cat *Catalog, chart string, version string) (*ChartMeta, error) {
	cms, ok := cat.ChartEntries[chart]
	if !ok {
		return nil, fmt.Errorf("application %s not found in catalog", chart)
	}

	var cm *ChartMeta
	if version != "" {
		for _, c := range cms {
			log.Debugf("Checking version %s against %s", c.Version, version)
			if c.Version == version {
				cm = &c
				break
			}
		}
	} else {
		cm = &cms[0]
	}
	if cm == nil {
		return nil, fmt.Errorf("application %s does not have version %s", chart, version)
	}

	return cm, nil
}

// getKubeVersion gets the server version for a Kubernetes cluster
func getKubeVersion(kubeconfig string) (string, error) {
	rc, _, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return "", err
	}

	vi, err := k8s.GetServerVersion(rc)
	if err != nil {
		return "", err
	}

	return k8s.VersionInfoToString(vi), nil
}
