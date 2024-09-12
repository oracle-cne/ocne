// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"gopkg.in/yaml.v3"
)

// Why all the structs?  Can't you just use the structs from the Kubernetes
// Go libraries and avoid having to redefine all this stuff?
//
// Yes, techically.  However, when you marshal those structures into JSON
// the emit thousands and thousands of characters.  The resulting string
// is too large to fit into an ignition file for even the simplest
// configuration.  It would require that users always reference an external
// URL to configure things.

// https://kubernetes.io/docs/reference/config-api/kube-proxy-config.v1alpha1/
type KubeProxy struct {
	ApiVersion         string `yaml:"apiVersion"`
	Kind               string `yaml:"kind"`
	Mode               string `yaml:"mode"`
	MetricsBindAddress string `yaml:"metricsBindAddress"`
}

func GenerateKubeProxyConfiguration(proxyMode string) *KubeProxy {
	return &KubeProxy{
		ApiVersion:         "kubeproxy.config.k8s.io/v1alpha1",
		Kind:               "KubeProxyConfiguration",
		Mode:               proxyMode,
		MetricsBindAddress: "0.0.0.0:10249",
	}
}

func GenerateKubeProxyConfigurationYaml(proxyMode string) (string, error) {
	kpc := GenerateKubeProxyConfiguration(proxyMode)
	outBytes, err := yaml.Marshal(kpc)
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}
