// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package application

import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/oracle-cne/ocne/pkg/helm"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

// GetAnnotations returns a good set of Helm release annotations
func GetAnnotations(name string) map[string]string {
	return map[string]string{
		"meta.helm.sh/release-name": name,
	}
}

// DeploymentsForApplication returns a list of all deployments that
// are managed by a given Helm release.
func DeploymentsForApplication(name string, namespace string, kubeConfigPath string) ([]*appsv1.Deployment, error) {
	kubeInfo, err := client.CreateKubeInfo(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Make sure the release exists
	_, err = helm.GetRelease(kubeInfo, name, namespace)
	if err != nil {
		return nil, err
	}

	ret, err := k8s.GetDeploymentsWithAnnotations(kubeInfo.Client, "", GetAnnotations(name))
	if err != nil {
		return nil, err
	}

	log.Debugf("Found %d deployments", len(ret))

	return ret, nil
}

// DaemonSetsForApplication returns a list of all daemon sets that
// are managed by a given Helm release
func DaemonSetsForApplication(name string, namespace string, kubeConfigPath string) ([]*appsv1.DaemonSet, error) {
	kubeInfo, err := client.CreateKubeInfo(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// Make sure the release exists
	_, err = helm.GetRelease(kubeInfo, name, namespace)
	if err != nil {
		return nil, err
	}

	ret, err := k8s.GetDaemonSetsWithAnnotations(kubeInfo.Client, "", GetAnnotations(name))
	if err != nil {
		return nil, err
	}

	log.Debugf("Found %d daemonsets", len(ret))

	return ret, nil
}
