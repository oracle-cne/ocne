// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"


	"github.com/oracle-cne/ocne/pkg/constants"
)

func fromConfigMap(kubeIface kubernetes.Interface) (string, error) {
	cm, err := GetConfigmap(kubeIface, constants.KubeNamespace, constants.KubeCMName)
	if err != nil {
		return "", err
	}
	config := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(cm.Data["ClusterConfiguration"]), config)
	if err != nil {
		return "", err
	}
	hostIface, ok := config["controlPlaneEndpoint"]
	if !ok {
		return "", fmt.Errorf("Kubeadm configuration does not have a controlPlaneEndpoint")
	}

	host, ok := hostIface.(string)
	if !ok {
		return "", fmt.Errorf("Kubeadm configuration field controlPlaneEndpoint is not a string")
	}
	return host, nil
}

func GetControlPlaneEndpoint(restConfig *rest.Config) (string, error) {
	kubeIface, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", err
	}

	// Prefer the Kubeadm configmap because the rest client may
	// have been futzed with to account for network conditions.
	cpEndpoint, err := fromConfigMap(kubeIface)
	if cpEndpoint != "" {
		log.Debugf("Got endpoint %s from ConfigMap", cpEndpoint)
		return cpEndpoint, nil
	} else if err != nil {
		log.Debugf("Error finding KubeadmConfig from configmap: %+v", err)
	}

	// If that can't be used, look at the rest config directly.
	log.Debugf("Getting endpoint from rest config host field %s", restConfig.Host)
	u, err := url.Parse(restConfig.Host)
	if err != nil {
		return "", err
	}
	log.Debugf("Got endpoint %s from kubeconfig", u.Host)

	return u.Host, nil
}
