// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"bytes"

	"github.com/oracle-cne/ocne/pkg/catalog/embedded"
)

// InternalConnection implements the CatalogConnection interface
// for the compiled-in catalog.
type InternalConnection struct {
	KubeVersion string
}

// NewInternalConnection opens a connection to the internal catalog
func NewInternalConnection(kubeconfig string, ci *CatalogInfo) (CatalogConnection, error) {
	// Ignore any errors getting the Kubernetes version.  This catalog can
	// be searched without the need for a working cluster.  Due to the fact
	// that Kubernetes clients try very hard to connect to something, it's
	// nearly impossible to determine if a connection error is legitimate
	// or not.  Thankfully it either doesn't matter if it is possible to
	// connect to the cluster or doing anything with the applications in the
	// catalog will fail when the it is actually necessary to interact with
	// said cluster.
	ver, _ := getKubeVersion(kubeconfig)
	return &InternalConnection{
		KubeVersion: ver,
	}, nil
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
	return fromHelmYAML(buf.Bytes(), ic.KubeVersion)
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
