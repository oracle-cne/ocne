// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"fmt"
	httphelpers "github.com/oracle-cne/ocne/pkg/http"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// HelmConnection implements the "helm" protocol.  This protocol
// interacts directly with raw Helm repositories.
type HelmConnection struct {
	Kubeconfig  string
	CatalogInfo *CatalogInfo
	Uri         string
	LastSearch  *Catalog
}

// NewHelmConnection opens a connection to a vanilla Helm repo
func NewHelmConnection(kubeconfig string, ci *CatalogInfo) (CatalogConnection, error) {
	uri, err := getCatalogURI(kubeconfig, ci)
	if err != nil {
		return nil, err
	}

	return &HelmConnection{
		Kubeconfig:  kubeconfig,
		CatalogInfo: ci,
		Uri:         uri,
	}, nil
}

// GetCharts returns a Catalog populated with the charts from a particular
// Helm repository.  Most Helm repos are of a reasonable size, and a
// query pattern is not required.  The query parameter is ignored.
func (hc *HelmConnection) GetCharts(query string) (*Catalog, error) {
	uri := fmt.Sprintf("%s/%s", hc.Uri, "charts/index.yaml")
	log.Debugf("Fetching %s\n", uri)
	body, err := httphelpers.HTTPGet(uri)
	if err != nil {
		return nil, err
	}

	cat, err := fromHelmYAML(body)
	if err != nil {
		return nil, err
	}

	cat.RawJSON = body
	hc.LastSearch = cat

	return cat, nil
}

// GetChart returns the bytes of the tarball for the desired chart/version pair.
func (hc *HelmConnection) GetChart(chart string, version string) ([]byte, error) {
	// If there has been no search, make one to populate the cache
	if hc.LastSearch == nil {
		_, err := hc.GetCharts(chart)
		if err != nil {
			return nil, err
		}
	}

	cm, err := findChartAtVersion(hc.LastSearch, chart, version)
	if err != nil {
		return nil, err
	}

	if len(cm.Urls) == 0 {
		return nil, fmt.Errorf("application %s has no downloadable artifacts", chart)
	}

	uri := getCanonicalURI(fmt.Sprintf("%s/%s", hc.Uri, "charts/"), cm.Urls[0])
	log.Debugf("Fetching %s\n", uri)
	return httphelpers.HTTPGet(uri)
}

// fromJSON un-marshals the catalog
func fromHelmYAML(data []byte) (*Catalog, error) {
	cat := Catalog{}
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}
