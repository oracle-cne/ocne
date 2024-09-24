// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bytes"
	"fmt"
	"io"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/ls"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
)

// Install parses a catalog for an entry with a given application name and version
// It downloads the helm chart for that entry locally and uploads it to a cluster
func Install(opt application.InstallOptions) error {
	// Download the helm chart that you want to install into the cluster
	chartReader, err := DownloadApplication(opt.Catalog, opt.AppName, opt.Version)
	if err != nil {
		return err
	}

	kubeInfo, err := client.CreateKubeInfo(opt.KubeConfigPath)
	if err != nil {
		return err
	}

	if opt.Namespace == "" {
		opt.Namespace, err = client.GetNamespaceFromConfig(opt.KubeConfigPath)
		if err != nil {
			return err
		}
	}

	if opt.ReleaseName == "" {
		releaseExists, err := DoesReleaseExist(opt.AppName, opt.KubeConfigPath, opt.Namespace)
		if err != nil {
			return err
		}
		if releaseExists {
			return fmt.Errorf("A release already exists with the name %s in the cluster. Please specify a different release name ", opt.AppName)
		} else {
			opt.ReleaseName = opt.AppName
		}
	}
	overrides := append([]helm.HelmOverrides{}, opt.Overrides...)
	if opt.Values != "" {
		overrides = append(overrides, helm.HelmOverrides{
			FileOverride: opt.Values,
		})
	}
	// Upload the helm chart stored at the temporary directory
	_, err = helm.UpgradeChartFromArchive(kubeInfo, opt.ReleaseName, opt.Namespace, true, chartReader, false, false, overrides, opt.ResetValues)
	return err
}

// DownloadApplication downloads a single application from the catalog endpoint
func DownloadApplication(cat *catalog.Catalog, chart string, version string) (io.Reader, error) {
	log.Debugf("Downloading Helm Chart %s at %s", chart, version)
	bytesOfTgzFile, err := cat.Connection.GetChart(chart, version)
	log.Debugf("Finished downloading %d bytes", len(bytesOfTgzFile))
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(bytesOfTgzFile), nil
}

// DoesReleaseExist checks to see if a given release is already installed.
func DoesReleaseExist(releaseName string, kubeConfig string, namespace string) (bool, error) {
	releases, err := ls.List(application.LsOptions{
		KubeConfigPath: kubeConfig,
		Namespace:      namespace,
		All:            true,
	})
	if err != nil {
		return false, err
	}
	for _, release := range releases {
		if releaseName == release.Name {
			return true, nil
		}
	}
	return false, nil

}

// InstallInternalCatalog installs the internal OCNE catalog built into the client binary.
func InstallInternalCatalog(kubeConfigPath string, namespace string) error {
	releaseExists, err := DoesReleaseExist(constants.CatalogRelease, kubeConfigPath, constants.CatalogNamespace)
	if err != nil {
		return err
	}
	if releaseExists {
		return fmt.Errorf("The built-in catalog is already installed in the cluster.  Use the ocne application update command.")
	}
	log.Infof("Installing %s catalog", constants.CatalogName)

	// get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(kubeConfigPath)
	if err != nil {
		return err
	}

	// install the OCNE catalog application
	var apps []ApplicationDescription
	apps = append(apps, NewInternalCatalogApplication(namespace))
	if err := InstallApplications(apps, kubeConfigPath, false); err != nil {
		return err
	}

	// wait for the catalog to be installed
	if err := catalog.WaitForInternalCatalogInstall(kubeClient, logutils.Info); err != nil {
		return err
	}

	log.Info("Catalog installed successfully")
	return nil
}
