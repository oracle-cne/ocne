// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	httphelpers "github.com/oracle-cne/ocne/pkg/http"
)

// HelmConnection implements the "helm" protocol.  This protocol
// interacts directly with raw Helm repositories.
type HelmConnection struct {
	Kubeconfig  string
	KubeVersion string
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

	ver, err := getKubeVersion(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &HelmConnection{
		Kubeconfig:  kubeconfig,
		KubeVersion: ver,
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

	cat, err := fromHelmYAML(body, hc.KubeVersion)
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

// fromJSON un-marshals the catalog, and filters by Kuberenetes version
func fromHelmYAML(data []byte, kubeVersion string) (*Catalog, error) {
	cat := Catalog{}
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, err
	}

	// Sort the entries for easy searching and viewing.  For display
	// and search purposes, the entries are sorted in reverse order.
	// That way, newer versions trivially print first and the latest
	// version is always at index zero.
	for _, metas := range cat.ChartEntries {
		slices.SortFunc(metas, func(a ChartMeta, b ChartMeta)int{
			aVer := a.AppVersion
			bVer := b.AppVersion

			log.Debugf("Comparing %s-%s to %s-%s", a.Name, aVer, b.Name, bVer)

			// If the strings are the same, then so
			// are the versions.  This skips some expensive
			// checks while also giving an amount of sorting
			// stability to nonsense values.
			if aVer == bVer {
				return 0
			}

			// If a is empty, then it is less.  Empty strings
			// are less than non-empty strings
			if aVer == "" {
				return 1
			}

			// If b is empty, then it is less.
			if bVer == "" {
				return -1
			}

			aSv, aErr := semver.NewVersion(aVer)
			bSv, bErr := semver.NewVersion(bVer)

			// If both values are nonsense, string compare them
			if aErr != nil && bErr != nil {
				return strings.Compare(bVer, aVer)
			}

			// If only one is, that one is less
			if aErr != nil {
				return 1
			}
			if bErr != nil {
				return -1
			}

			// We did it!  Both values are semantic versions!
			return bSv.Compare(aSv)
		})
	}

	if kubeVersion == "" {
		return &cat, nil
	}

	kv, err := semver.NewVersion(kubeVersion)
	if err != nil {
		log.Errorf("%s: %s", kubeVersion, err)
		return nil, err
	}

	newCEs := map[string][]ChartMeta{}
	for name, metas := range cat.ChartEntries {
		// Delete anything that is not supported in the
		// target cluster.
		metas = slices.DeleteFunc(metas, func(cm ChartMeta)bool{
			// Assume unspecified constraint means all versions
			if cm.KubeVersion == "" {
				return false
			}

			// Assume bad constraint means no constraint
			c, err := semver.NewConstraint(cm.KubeVersion)
			if err != nil {
				return true
			}

			return !c.Check(kv)
		})

		// Sort them
		newCEs[name] = metas
	}
	cat.ChartEntries = newCEs

	return &cat, nil
}
