// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"bytes"

	"github.com/oracle-cne/ocne/pkg/catalog/embedded"
)

// InternalConnection implements the CatalogConnection interface
// for the compiled-in catalog.
type InternalConnection struct {
}

// NewInternalConnection opens a connection to the internal catalog
func NewInternalConnection(kubeconfig string, ci *CatalogInfo) (CatalogConnection, error) {
	return &InternalConnection{}, nil
}

// GetCharts returns a Catalog populated with the contents of
// the embedded catalog
func (ic *InternalConnection) GetCharts(query string) (*Catalog, error) {
	reader, err := embedded.GetIndex()
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(reader)
	return fromHelmYAML(buf.Bytes())
}

// GetChart returns the bytes of the tarball for the desired chart/version pair
func (ic *InternalConnection) GetChart(chart string, version string) ([]byte, error) {
	if version == "" {
		cat, err := ic.GetCharts(chart)
		if err != nil {
			return nil, err
		}

		cm, err := findChartAtVersion(cat, chart, version)
		if err != nil {
			return nil, err
		}

		chart = cm.Name
		version = cm.Version
	}

	reader, err := embedded.GetChartAtVersion(chart, version)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(reader)
	return buf.Bytes(), err
}
