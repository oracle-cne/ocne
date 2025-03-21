// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	app "github.com/oracle-cne/ocne/pkg/application"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

func uninstallCRDs(kubeInfo *client.KubeInfo, name string, namespace string) error {
	crds, err := app.CRDsForApplication(name, namespace, kubeInfo.KubeconfigPath)
	if err != nil {
		return err
	}

	// Check if the CRDs are have any associated CRs.  If so, log them and fail
	haveCRs := false
	for _, crd := range crds {
		for _, ver := range crd.Spec.Versions {
			apiVer := fmt.Sprintf("%s/%s", crd.Spec.Group, ver.Name)
			crs, err := k8s.GetResources(kubeInfo.RestConfig, "", apiVer, crd.Spec.Names.Kind)
			if err != nil {
				return err
			}

			if len(crs.Items) > 0 {
				haveCRs = true
				log.Warnf("%s/%s has resources defined", apiVer, crd.Spec.Names.Kind)
				for _, cr := range crs.Items {
					log.Warnf("  %s/%s", cr.GetNamespace(), cr.GetName())
				}
			}
	 }
	}

	if haveCRs {
		return fmt.Errorf("resources exist for some of the CustomResourceDefinitions from this application")
	}

	for _, crd := range crds {
		err = k8s.DeleteCRD(kubeInfo.RestConfig, crd)
	}
	return nil
}

// Uninstall takes in a set of uninstall options and returns an error that indicates whether the uninstallation was successful
func Uninstall(opt application.UninstallOptions) error {
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
	log.Infof("Uninstalling release %s", opt.ReleaseName)
	releaseInstalled, err := helm.IsReleaseInstalled(kubeInfo, opt.ReleaseName, opt.Namespace)
	if err != nil {
		return err
	}

	if releaseInstalled {
		err = helm.Uninstall(kubeInfo, opt.ReleaseName, opt.Namespace, false)
		if err != nil {
			return err
		}
	} else if !opt.UninstallCRDs {
		return fmt.Errorf("%s cannot be uninstalled because it is not found in the %s namespace", opt.ReleaseName, opt.Namespace)
	}

	if opt.UninstallCRDs {
		return uninstallCRDs(kubeInfo, opt.ReleaseName, opt.Namespace)
	}
	return nil
}
