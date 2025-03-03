// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/constants"
)

type KubeletConfig struct {
	APIVersion string `yaml:"apiVersion"`
	ClusterDNS []string `yaml:"clusterDNS"`
}

var kubeletData = "kubelet"

func GetKubeletConfig(client kubernetes.Interface) (*KubeletConfig, error) {
	kcm, err:= GetConfigmap(client, constants.KubeNamespace, constants.KubeletCMName)
	if err != nil {
		return nil, err
	}

	kubeletConf, ok := kcm.Data[kubeletData]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s in %s does not have key %s", constants.KubeletCMName, constants.KubeNamespace, kubeletData)
	}

	ret := KubeletConfig{}
	err = yaml.Unmarshal([]byte(kubeletConf), &ret)
	if err != nil {
		return nil, err
	}

	return &ret, err
}
