// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/commands/catalog/search"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s/client"

	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

type ApplicationDescription struct {
	PreInstall     func() error
	Force          bool
	Application    *types.Application
	kubeConfigPath string
}

func updateApplication(appIface interface{}) error {
	return installOrUpdateApplication(appIface, true)
}

func installApplication(appIface interface{}) error {
	return installOrUpdateApplication(appIface, false)
}

func installOrUpdateApplication(appIface interface{}, update bool) error {
	app, ok := appIface.(*ApplicationDescription)
	if !ok {
		return fmt.Errorf("Tried to install something that was not an application: %+v", appIface)
	}

	catalog, err := search.Search(catalog.SearchOptions{
		KubeConfigPath: app.kubeConfigPath,
		CatalogName:    app.Application.Catalog,
		Pattern:        app.Application.Name,
	})
	if err != nil {
		log.Debugf("Could not search catalog: %v", err)
		return err
	}

	config := ""
	if app.Application.Config != nil {
		yamlValues, err := yaml.Marshal(app.Application.Config)
		if err != nil {
			return err
		}
		config = string(yamlValues)
	}
	opt := application.InstallOptions{
		Catalog:        catalog,
		KubeConfigPath: app.kubeConfigPath,
		AppName:        app.Application.Name,
		Namespace:      app.Application.Namespace,
		ReleaseName:    app.Application.Release,
		Version:        app.Application.Version,
		Values:         app.Application.ConfigFrom,
		Force:          app.Force,
		Overrides: []helm.HelmOverrides{
			{
				LiteralOverride: config,
			},
		},
	}

	isInstalled, err := DoesReleaseExist(opt.ReleaseName, opt.KubeConfigPath, opt.Namespace)
	if err != nil {
		log.Debugf("Failed to check if release is installed: %v", err)
		return err
	}

	if isInstalled && !update {
		return nil
	}

	_, kubeClient, err := client.GetKubeClient(opt.KubeConfigPath)
	if err != nil {
		return err
	}

	// Pre-create the namespace
	if err = k8s.CreateNamespaceIfNotExists(kubeClient, opt.Namespace); err != nil {
	}

	if app.PreInstall != nil {
		err = app.PreInstall()
		if err != nil {
			log.Debugf("Application pre-install procedure failed: %v", err)
			return err
		}
	}

	// Try a couple of times, just in case an odd failure occurs.
	_, _, err = util.LinearRetry(func(optIface interface{}) (interface{}, bool, error) {
		opt, _ := optIface.(*application.InstallOptions)
		err := Install(*opt)

		if err != nil {
			log.Debugf("Application installation failed: %v", err)
		}
		return nil, false, err
	}, &opt)

	return err
}

func makeWaiter(a *ApplicationDescription) *logutils.Waiter {
	// Release and Namespace are optional, so do our best to construct a reasonable message
	msg := "Installing "
	if len(a.Application.Release) > 0 {
		msg = msg + a.Application.Release
	} else {
		msg = msg + a.Application.Name
	}
	if len(a.Application.Namespace) > 0 {
		msg = msg + " into " + a.Application.Namespace
	}

	return &logutils.Waiter{
		WaitFunction: installApplication,
		Args:         a,
		Message:      msg,
	}
}

func makeUpdateWaiter(a *ApplicationDescription) *logutils.Waiter {
	// Release and Namespace are optional, so do our best to construct a reasonable message
	msg := "Updating "
	if len(a.Application.Release) > 0 {
		msg = msg + a.Application.Release
	} else {
		msg = msg + a.Application.Name
	}
	if len(a.Application.Namespace) > 0 {
		msg = msg + " in " + a.Application.Namespace
	}

	return &logutils.Waiter{
		WaitFunction: updateApplication,
		Args:         a,
		Message:      msg,
	}
}

// InstallApplications installs a sequence of applications.  Applications must be
// given in dependency order.  Applications must not have dependency cycles.  The
// behavior of this function if there is a dependency cycle between applications
// is undefined.
func InstallApplications(apps []ApplicationDescription, kubeConfigPath string, quiet bool) error {
	return installOrUpdateApplications(apps, kubeConfigPath, quiet, false)
}

// UpdateApplications updates a sequence of applications.  Applications must be
// given in dependency order.  Applications must not have dependency cycles.  The
// behavior of this function if there is a dependency cycle between applications
// is undefined.
func UpdateApplications(apps []ApplicationDescription, kubeConfigPath string, quiet bool) error {
	return installOrUpdateApplications(apps, kubeConfigPath, quiet, true)
}

// installOrUpdateApplications installs or updates a sequence of applications.  Applications must be
// given in dependency order.  Applications must not have dependency cycles.  The
// behavior of this function if there is a dependency cycle between applications
// is undefined.
func installOrUpdateApplications(apps []ApplicationDescription, kubeConfigPath string, quiet bool, update bool) error {
	// Convert the application to an installable thing
	// and create waiter list.
	info := logutils.Info
	if quiet {
		info = func(s string) {}
	}
	var waiters []*logutils.Waiter
	for i := range apps {
		apps[i].kubeConfigPath = kubeConfigPath
		if update {
			waiters = append(waiters, makeUpdateWaiter(&apps[i]))
		} else {
			waiters = append(waiters, makeWaiter(&apps[i]))
		}
	}

	haveErrors := logutils.WaitForSerial(info, waiters)
	if haveErrors {
		return fmt.Errorf("Could not install all applications")
	}

	return nil
}

// NewInternalCatalogApplication returns an ApplicationDescription for the internal catalog
func NewInternalCatalogApplication(namespace string) ApplicationDescription {
	return ApplicationDescription{
		Application: &types.Application{
			Name:      constants.CatalogChart,
			Namespace: namespace,
			Release:   constants.CatalogRelease,
			Version:   constants.CatalogVersion,
			Catalog:   catalog.InternalCatalog,
		},
	}
}
