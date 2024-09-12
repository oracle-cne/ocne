// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/oracle-cne/ocne/pkg/commands/application"
	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

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
	if !releaseInstalled {
		return fmt.Errorf("%s cannot be uninstalled because it is not found in the %s namespace", opt.ReleaseName, opt.Namespace)
	}
	return helm.Uninstall(kubeInfo, opt.ReleaseName, opt.Namespace, false)
}
