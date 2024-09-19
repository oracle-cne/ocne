// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/commands/application/ls"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
)

// Update takes in a set of update options and returns an error that indicates whether the update was successful
func Update(opt application.UpdateOptions) error {
	// If release is not installed, don't update it
	kubeInfo, err := client.CreateKubeInfo(opt.KubeConfigPath)
	if err != nil {
		return err
	}
	log.Infof("Updating release %s", opt.ReleaseName)
	if opt.Namespace == "" {
		opt.Namespace, err = client.GetNamespaceFromConfig(opt.KubeConfigPath)
		if err != nil {
			return err
		}
	}
	releaseInstalled, err := helm.IsReleaseInstalled(kubeInfo, opt.ReleaseName, opt.Namespace)
	if err != nil {
		return err
	}
	if !releaseInstalled {
		return fmt.Errorf("%s cannot be updated because it is not found in the %s namespace", opt.ReleaseName, opt.Namespace)
	}

	// Determine Application Name of Release
	appName, err := GetAppNameFromRelease(opt.ReleaseName, opt.KubeConfigPath, opt.Namespace)
	if err != nil {
		return err
	}

	// Get the catalog information along with setting the port-forward
	cat, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: opt.KubeConfigPath,
		CatalogName:    opt.CatalogName,
		Pattern:        appName,
	})
	if err != nil {
		return err
	}

	// If the release is installed, update to the specified version or the most stable version
	err = install.Install(application.InstallOptions{
		Namespace:      opt.Namespace,
		Catalog:        cat,
		KubeConfigPath: opt.KubeConfigPath,
		AppName:        appName,
		Version:        opt.Version,
		ReleaseName:    opt.ReleaseName,
		Values:         opt.Values,
	})
	return err
}

// GetAppNameFromRelease gets the application name for a given release in a namespace
func GetAppNameFromRelease(releaseName string, kubeConfigPath string, namespace string) (string, error) {
	releases, err := ls.List(application.LsOptions{
		KubeConfigPath: kubeConfigPath,
		Namespace:      namespace,
		All:            false,
	})
	if err != nil {
		return "", err
	}
	for _, release := range releases {
		if releaseName == release.Name {
			return release.Chart.Name(), nil
		}
	}
	return "", fmt.Errorf("an application name for that release was not found in the cluster")

}

// UpdateInternalCatalog updates the internal OCNE catalog built into the client binary.
func UpdateInternalCatalog(kubeConfigPath string, namespace string) error {
	releaseExists, err := install.DoesReleaseExist(constants.CatalogRelease, kubeConfigPath, constants.CatalogNamespace)
	if err != nil {
		return err
	}

	if !releaseExists {
		return fmt.Errorf("The built-in catalog is not installed in the cluster.  Use the ocne application install command.")
	}
	log.Infof("Updating %s catalog", constants.CatalogName)

	// get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(kubeConfigPath)
	if err != nil {
		return err
	}

	// install the OCNE catalog application
	apps := []install.ApplicationDescription{}
	apps = append(apps, install.NewInternalCatalogApplication(namespace))
	if err := install.UpdateApplications(apps, kubeConfigPath, false); err != nil {
		return err
	}

	// wait for the catalog to be installed
	if err := catalog.WaitForInternalCatalogInstall(kubeClient, logutils.Info); err != nil {
		return err
	}

	log.Info("Catalog updated successfully")
	return nil
}
