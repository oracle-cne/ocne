// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package application

import (
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
)

func PodsForApplication(name string, namespace string, kubeConfigPath string) ([]*v1.Pod, error) {
	daemonSets, err := DaemonSetsForApplication(name, namespace, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	deployments, err := DeploymentsForApplication(name, namespace, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	_, client, err := client.GetKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	ret := []*v1.Pod{}
	for _, ds := range daemonSets {
		dsPods, err := k8s.GetDaemonSetPods(client, ds)
		if err != nil {
			return nil, err
		}
		ret = append(ret, dsPods...)
	}

	for _, dep := range deployments {
		depPods, err := k8s.GetDeploymentPods(client, dep)
		if err != nil {
			return nil, err
		}
		ret = append(ret, depPods...)
	}

	log.Debugf("Found %d application pods", len(ret))

	return ret, nil
}
