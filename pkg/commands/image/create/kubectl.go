// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"io"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
)

// Logger used to filter some  kubectl output from appearing on the console
var myLogger klog.Logger

// kubectlConfig contains information used access a pod via kubectl
type kubectlConfig struct {
	configFlags *genericclioptions.ConfigFlags
	streams     genericiooptions.IOStreams
	podName     string
	namespace   string
}

// getKubectlConfig gets the configuration needed to use kubectl
func getKubectlConfig(config *otypes.Config, restConfig *rest.Config, namespace string) (*kubectlConfig, error) {
	// Create ConfigFlags so that kubectl cmd package can be used
	kubeConfigPath, _, err := client.GetKubeConfigLocation(*config.KubeConfig)
	if err != nil {
		return nil, err
	}
	kubeInfo := client.KubeInfo{KubeconfigPath: kubeConfigPath, RestConfig: restConfig}
	configFlags := &genericclioptions.ConfigFlags{
		KubeConfig:    &kubeInfo.KubeconfigPath,
		BearerToken:   &kubeInfo.RestConfig.BearerToken,
		Namespace:     &namespace,
		Insecure:      util.BoolPtr(true),
		TLSServerName: &kubeInfo.KubeApiServerIP,
	}

	kc := kubectlConfig{
		configFlags: configFlags,
		streams: genericiooptions.IOStreams{
			In:     os.Stdin,
			Out:    io.Discard,
			ErrOut: os.Stderr,
		},
		podName:   podName,
		namespace: namespace,
	}

	return &kc, nil
}
