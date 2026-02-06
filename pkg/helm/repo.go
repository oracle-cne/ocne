// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"strings"

	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/getter"
	repov1 "helm.sh/helm/v4/pkg/repo/v1"
)

type HelmReleaseOpts struct {
	RepoURL      string
	ReleaseName  string
	Namespace    string
	ChartPath    string
	ChartVersion string
	Overrides    []HelmOverrides

	Username string
	Password string
}

// GetReleaseChartVersion extracts the chart version from a deployed helm release
func GetReleaseChartVersion(kubeInfo *client.KubeInfo, releaseName string, namespace string) (string, error) {
	releases, err := GetReleases(kubeInfo, namespace)
	if err != nil {
		if err.Error() == ReleaseNotFound {
			return ReleaseNotFound, nil
		}
		return "", err
	}

	var version string
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			version = info.Chart.Metadata.Version
			break
		}
	}
	return strings.TrimSpace(version), nil
}

// FindLatestChartVersion Finds the most recent ChartVersion
func FindLatestChartVersion(chartName, repoName, repoURI string) (string, error) {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	if err != nil {
		return "", err
	}
	version, err := findMostRecentChartVersion(indexFile, chartName)
	if err != nil {
		return "", err
	}
	return version.Version, nil
}

// findMostRecentChartVersion Finds the most recent ChartVersion that
func findMostRecentChartVersion(indexFile *repov1.IndexFile, chartName string) (*repov1.ChartVersion, error) {
	// The indexFile is already sorted in descending order for each chart
	chartVersions := findChartEntry(indexFile, chartName)
	if len(chartVersions) == 0 {
		return nil, fmt.Errorf("no entries found for chart %s in repo", chartName)
	}
	return chartVersions[0], nil
}

func findChartEntry(index *repov1.IndexFile, chartName string) repov1.ChartVersions {
	var selectedVersion repov1.ChartVersions
	for name, chartVersions := range index.Entries {
		if name == chartName {
			selectedVersion = chartVersions
		}
	}
	return selectedVersion
}

func loadAndSortRepoIndexFile(repoName string, repoURL string) (*repov1.IndexFile, error) {
	// NOTES:
	// - we'll need to allow defining credentials etc in the source lists for protected repos
	// - also we'll likely need better scaffolding around local repo management
	cfg := &repov1.Entry{
		Name: repoName,
		URL:  repoURL,
	}
	chartRepository, err := repov1.NewChartRepository(cfg, getter.All(cli.New()))
	if err != nil {
		return nil, err
	}
	indexFilePath, err := chartRepository.DownloadIndexFile()
	if err != nil {
		return nil, err
	}
	indexFile, err := repov1.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, err
	}
	indexFile.SortEntries()
	return indexFile, nil
}
