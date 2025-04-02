// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

type KubeletConfig struct {
	APIVersion string `yaml:"apiVersion"`
	ClusterDNS []string `yaml:"clusterDNS"`
}

var kubeletData = "kubelet"

func WaitForKubeletConfig(client kubernetes.Interface) (*KubeletConfig, error) {
	var ret *KubeletConfig
	var err error

	// Check once before waiting to avoid log spew
	ret, err = GetKubeletConfig(client)
	if err == nil {
		return ret, nil
	}

	waitors := []*logutils.Waiter{
		&logutils.Waiter{
			Message: "Waiting for ConfigMap kubelet-config",
			WaitFunction: func(ignored interface{}) error {
				// None of these values matter
				util.LinearRetryImpl(func(unused interface{})(interface{},bool,error){
					ret, err = GetKubeletConfig(client)
					return nil, false, err
				}, nil, 1*time.Second, 5*time.Minute)
				return err
			},
		},
	}
	haveError := logutils.WaitFor(logutils.Info, waitors)
	if haveError {
		return nil, fmt.Errorf("Failed to get ConfigMap kubelet-config")
	}
	return ret, nil
}

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
