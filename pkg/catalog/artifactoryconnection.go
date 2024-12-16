// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"encoding/json"
	"fmt"
	httphelpers "github.com/oracle-cne/ocne/pkg/http"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	SearchPath = "api/v1/packages/search"
)

// ArtifacthubPackageRepository contains select fields from the
// "repository" field in an ArtifactHub searchPackages API call.
//
// Please see: https://artifacthub.io/docs/api/#/Packages/searchPackages
type ArtifacthubPackageRepository struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

// ArtifacthubPackage contains select fields from a call to
// the ArtifactHub searchPackages API.
//
// Please see: https://artifacthub.io/docs/api/#/Packages/searchPackages
type ArtifacthubPackage struct {
	Name       string                       `json:"name"`
	Version    string                       `json:"version"`
	AppVersion string                       `json:"app_version"`
	Repository ArtifacthubPackageRepository `json:"repository"`
}

// ArtifacthubQueryResults is the top level structure of the response
// from a call to the ArtifactHub searchPackages API.
//
// Please see: https://artifacthub.io/docs/api/#/Packages/searchPackages
type ArtifacthubQueryResults struct {
	Packages []ArtifacthubPackage
}

// ArtifacthubConnection implements the "artifacthub" protocol.
// This protocol interacts with ArtifactHub style repositories.
type ArtifacthubConnection struct {
	Kubeconfig            string
	CatalogInfo           *CatalogInfo
	Uri                   *url.URL
	LastSearch            *Catalog
	LastArtifacthubSearch *ArtifacthubQueryResults
}

// NewArtifacthubConnection returns a Connection that implements the
// ArtifactHub protocol.
func NewArtifacthubConnection(kubeconfig string, ci *CatalogInfo) (CatalogConnection, error) {
	uriStr, err := getCatalogURI(kubeconfig, ci)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, err
	}

	return &ArtifacthubConnection{
		Kubeconfig:  kubeconfig,
		CatalogInfo: ci,
		Uri:         uri,
	}, nil
}

// GetCharts returns a Catalog containing the results of a query for Helm charts.
// artifacthub.io has a huge number of packages, over 10,000, so a query term
// is required.
func (ac *ArtifacthubConnection) GetCharts(query string) (*Catalog, error) {
	if query == "" {
		return nil, fmt.Errorf("ArtifactHub powered catalogs do not support unqualified searches")
	}

	uri := *ac.Uri
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, SearchPath)

	// Get all packages that match the query terms.  Since the ArtifactHub API
	// is paged, it is necessary to make multiple queries until the full list
	// has been read.
	results := &ArtifacthubQueryResults{}
	offset := 0
	limit := 50
	limitStr := strconv.Itoa(limit)
	for {
		// Set the query parameters
		params := url.Values{}
		params.Add("kind", "0")
		params.Add("facets", "true")
		params.Add("limit", limitStr)
		params.Add("offset", strconv.Itoa(offset))
		params.Add("ts_query_web", query)

		uri.RawQuery = params.Encode()
		log.Debugf("Querying %s", uri.String())
		body, err := httphelpers.HTTPGet(uri.String())
		if err != nil {
			return nil, err
		}

		tmpRes := &ArtifacthubQueryResults{}
		err = json.Unmarshal(body, tmpRes)
		if err != nil {
			return nil, err
		}
		log.Debugf("Received %d applications", len(tmpRes.Packages))

		results.Packages = append(results.Packages, tmpRes.Packages...)
		offset = offset + len(tmpRes.Packages)
		if len(tmpRes.Packages) < limit {
			break
		}
	}

	ret := &Catalog{
		ApiVersion:   "v1",
		ChartEntries: map[string][]ChartMeta{},
		Connection:   ac,
	}

	ac.LastSearch = ret
	ac.LastArtifacthubSearch = results

	// Massage the results into a Catalog structure to
	// allow for catalog readers to use a consistent format.
	for _, chart := range results.Packages {
		cm := ChartMeta{
			Name:       chart.Name,
			Version:    chart.Version,
			AppVersion: chart.AppVersion,
		}

		metas, ok := ret.ChartEntries[chart.Name]
		if !ok {
			ret.ChartEntries[chart.Name] = []ChartMeta{cm}
			continue
		}
		ret.ChartEntries[chart.Name] = append(metas, cm)
	}

	return ret, nil
}

// GetChart returns the bytes of the tarball for a given chart/version pair.  In
// general, ArtifactHub does not serve any Helm charts itself.  It is primarily
// a means of aggregating charts into a single, searchable location.  As such,
// this function will usually reach out to some other host to actually fetch
// the chart data.
func (ac *ArtifacthubConnection) GetChart(chart string, version string) ([]byte, error) {
	log.Debugf("Searching for chart %s", chart)
	// If a search has not been made, populate the cache.
	if ac.LastSearch == nil {
		_, err := ac.GetCharts(chart)
		if err != nil {
			return nil, err
		}
	}

	// Find the chart metadata from the ArtifactHub query
	var ahRes *ArtifacthubPackage
	for _, ahr := range ac.LastArtifacthubSearch.Packages {
		log.Debugf("Inspecting %s", ahr.Name)
		if ahr.Name == chart {
			log.Debugf("Found chart")
			ahRes = &ahr
			break
		}
	}

	// In theory this shouldn't happen.
	if ahRes == nil {
		return nil, fmt.Errorf("application %s not found in catalog", chart)
	}

	// Reach out to the main repository and get the chart
	uri := fmt.Sprintf("%s/%s", ahRes.Repository.Url, "index.yaml")
	log.Debugf("Fetching %s", uri)
	body, err := httphelpers.HTTPGet(uri)
	if err != nil {
		return nil, err
	}

	cat, err := fromHelmYAML(body)
	if err != nil {
		return nil, err
	}

	cm, err := findChartAtVersion(cat, chart, version)
	if err != nil {
		return nil, err
	}

	if len(cm.Urls) == 0 {
		return nil, fmt.Errorf("application %s has no downloadable artifacts", chart)
	}

	uri = getCanonicalURI(ahRes.Repository.Url, cm.Urls[0])

	log.Debugf("Fetching %s", uri)
	return httphelpers.HTTPGet(uri)
}
