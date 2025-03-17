// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"github.com/oracle-cne/ocne/pkg/helm"
	"time"
)

// CatalogConnection represents a connection to a catalog
type CatalogConnection interface {
	// GetCharts fetches charts in a catalog.  A query
	// string is given to limit the search results for
	// some protocols.  Implementations are not required
	// to respect the query string.
	GetCharts(string) (*Catalog, error)

	// GetChart downloads a chart at a specific version
	GetChart(string, string) ([]byte, error)
}

// ChartMeta represents a single helm chart
// NOTE: This structure is derived from the JSON return from the catalog service /charts/ endpoint
type ChartMeta struct {
	Name        string             `yaml:"name"`
	Version     string             `yaml:"version"`
	Description string             `yaml:"description"`
	ApiVersion  string             `yaml:"apiVersion"`
	AppVersion  string             `yaml:"appVersion"`
	Type        string             `yaml:"type"`
	Urls        []string           `yaml:"urls"`
	Created     time.Time          `yaml:"created"`
	Digest      string             `yaml:"digest"`
	Annotations map[string]string  `yaml:"annotations"`
}

// Catalog contains the helm chart index of all the charts in the catalog
// NOTE: This structure is derived from the JSON return from the catalog service /charts/ endpoint
type Catalog struct {
	ApiVersion   string                 `yaml:"apiVersion"`
	Generated    time.Time              `yaml:"generated"`
	ChartEntries map[string][]ChartMeta `yaml:"entries"`
	RawJSON      []byte
	Connection   CatalogConnection
}

// CatalogInfo specifies the catalog information
type CatalogInfo struct {
	// ServiceNsn is the namespaced name of the Kubernetes service hosting the catalog
	ServiceNsn types.NamespacedName

	// CatalogName is the user visible name that identifies the catalog
	CatalogName string

	// Protocol is the catalog protocol for the catalog
	Protocol string

	// Uri is the URI of the of the catalog
	Uri string

	// Port is the port for the service, if one is defined
	Port int32

	// Scheme is the scheme for the service, if one is defined.
	// This value is taken from the name of the port.
	Scheme string

	// Hostname is the hostname of the catalog
	// for ExternalName services
	Hostname string

	// Type is the type of service
	Type v1.ServiceType
}

// SearchOptions are the options for search
type SearchOptions struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// CatalogName is the name of the catalog to search
	CatalogName string

	// Pattern is a regular expression search pattern
	Pattern string
}

// CopyOptions are the options for copy
type CopyOptions struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// FilePath is the name of the source file
	FilePath string

	// Destination is the name of the destination domain
	Destination string

	// DestinationFilePath is the name of the destination file
	DestinationFilePath string

	// Images is an optional slice of images, if provided, FilePath is ignored
	Images []string
}

type ChartWithOverrides struct {
	// Chart represents versions of a helm chart. One entry per version.
	Chart ChartMeta

	// Overrides is a list of overrides that get munged together. Later values take precedence over earlier ones.
	Overrides []helm.HelmOverrides
}
