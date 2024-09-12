// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package search

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/ls"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"regexp"
	"strings"

	"github.com/oracle-cne/ocne/pkg/constants"
)

// Search searches for applications in the catalog and returns the local port number that is used for a port-forwarding
func Search(opt catalog.SearchOptions) (*catalog.Catalog, error) {
	err := validateOptions(opt)
	if err != nil {
		return nil, err
	}

	// If no catalog was provided, add one
	if opt.CatalogName == "" {
		opt.CatalogName = constants.DefaultCatalogName
	}

	catalogInfo, err := getCatalogService(opt)
	if err != nil {
		return nil, err
	}

	cc, err := catalog.NewConnection(opt.KubeConfigPath, catalogInfo)
	if err != nil {
		return nil, err
	}

	catalog, err := cc.GetCharts(opt.Pattern)
	if err != nil {
		return nil, err
	}

	catalog.Connection = cc

	err = filterCharts(catalog, opt.Pattern)
	return catalog, err
}

// filterCharts removes charts from the catalog that don't match the regular expression
func filterCharts(chartMeta *catalog.Catalog, regx string) error {
	if regx == "" {
		return nil
	}

	// remove the charts that don't match
	re := regexp.MustCompile(regx)
	for key := range chartMeta.ChartEntries {
		if !re.MatchString(key) {
			delete(chartMeta.ChartEntries, key)
		}
	}

	return nil
}

// validate the search options
func validateOptions(opt catalog.SearchOptions) error {
	if opt.Pattern == "" {
		return nil
	}
	// Validate that the regular expression is ok
	_, err := regexp.Compile(opt.Pattern)
	return err
}

// getCatalogService returns the NamespacedName of the catalog service
func getCatalogService(opt catalog.SearchOptions) (*catalog.CatalogInfo, error) {
	// If looking for the internal catalog, return a reasonable
	// value.  This is done before the Ls rather than as part
	// of it because the internal catalog should not end
	// up in the list when searching via the CLI.
	if opt.CatalogName == catalog.InternalCatalog {
		return &catalog.CatalogInfo{
			CatalogName: opt.CatalogName,
			Protocol:    "internal",
		}, nil
	}

	catInfos, err := ls.Ls(opt.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	for _, c := range catInfos {
		if c.CatalogName == opt.CatalogName {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("catalog '%s' not found", opt.CatalogName)
}

func GetChartObjects(kubeConfigPath string, chartCatalogName string, name string, version string, overrides []helm.HelmOverrides, k8sVersion *semver.Version) ([]unstructured.Unstructured, error) {
	kubeInfo, err := client.CreateKubeInfo(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	foundCatalog, err := Search(catalog.SearchOptions{
		KubeConfigPath: kubeConfigPath,
		CatalogName:    chartCatalogName,
		Pattern:        name,
	})
	if err != nil {
		tmp := fmt.Sprintf("Could not search catalog %s: %s", name, err.Error())
		return nil, errors.New(tmp)
	}
	chart, err := foundCatalog.Connection.GetChart(name, version)
	if err != nil {
		tmp := fmt.Sprintf("Error getting helm chart %s, error getting chart from catalog: %s", name, err.Error())
		return nil, errors.New(tmp)
	}
	in := bytes.NewReader(chart)
	archive, err := loader.LoadArchive(in)
	if err != nil {
		tmp := fmt.Sprintf("Error getting helm chart %s, error reading chart bytes: %s", name, err.Error())
		return nil, errors.New(tmp)
	}
	if overrides == nil {
		overrides = make([]helm.HelmOverrides, 0)
	}
	manifest, err := helm.Template(kubeInfo, archive, overrides, k8sVersion)
	if err != nil {
		tmp := fmt.Sprintf("Error getting helm chart %s, error rendering chart: %s", name, err.Error())
		return nil, errors.New(tmp)
	}
	stringReader := strings.NewReader(manifest)
	bufReader := bufio.NewReader(stringReader)
	objects, err := k8s.Unmarshall(bufReader)
	if err != nil {
		tmp := fmt.Sprintf("Error getting helm chart %s, could not unmarshall chart with error: %s", name, err.Error())
		return nil, errors.New(tmp)
	} else if len(objects) == 0 {
		tmp := fmt.Sprintf("Error getting helm chart %s, could not unmarshall chart: no objects found", name)
		return nil, errors.New(tmp)
	}
	return objects, nil
}
