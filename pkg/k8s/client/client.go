// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package client

import (
	apiextv1Client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type TunnelInfo struct {
	HostIP         string
	KubernetesIP   string
	KubernetesPort string
	SshUser        string
}

type KubeInfo struct {
	KubeApiServerIP string
	KubeconfigPath  string
	RestConfig      *rest.Config
	Client          kubernetes.Interface
}

// GetKubeClient - return a Kubernetes clientset for use with the go-client
func GetKubeClient(kubeconfigPath string) (*rest.Config, kubernetes.Interface, error) {
	path, _, err := GetKubeConfigLocation(kubeconfigPath)
	if err != nil {
		return nil, nil, err
	}

	restConfig, err := GetKubeConfig(path)
	if err != nil {
		return nil, nil, err
	}

	cs, err := kubernetes.NewForConfig(restConfig)
	return restConfig, cs, err
}

// GetKubernetesClientset returns the Kubernetes clientset for the cluster set in the environment
func GetKubernetesClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	// use the current context in the kubeconfig
	var clientset *kubernetes.Clientset
	config, err := GetKubeConfig(kubeconfigPath)
	if err != nil {
		return clientset, err
	}
	return GetKubernetesClientsetWithConfig(config)
}

// GetKubernetesClientsetWithConfig returns the Kubernetes clientset for the given configuration
func GetKubernetesClientsetWithConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	var clientset *kubernetes.Clientset
	clientset, err := kubernetes.NewForConfig(config)
	return clientset, err
}

// GetCoreV1Func is the function to return the CoreV1Interface
var GetCoreV1Func = GetCoreV1Client

// GetCoreV1Client returns the CoreV1Interface
func GetCoreV1Client(config *rest.Config) (corev1.CoreV1Interface, error) {
	goClient, err := GetGoClient(config)
	if err != nil {
		return nil, err
	}
	return goClient.CoreV1(), nil
}

func ResetCoreV1Client() {
	GetCoreV1Func = GetCoreV1Client
}

// GetAPIExtV1ClientFunc is the function to return the ApiextensionsV1Interface
var GetAPIExtV1ClientFunc = GetAPIExtV1Client

// ResetGetAPIExtV1ClientFunc for unit testing, to reset any overrides to GetAPIExtV1ClientFunc
func ResetGetAPIExtV1ClientFunc() {
	GetAPIExtV1ClientFunc = GetAPIExtV1Client
}

// GetAPIExtV1Client returns the ApiextensionsV1Interface
func GetAPIExtV1Client(config *rest.Config) (apiextv1.ApiextensionsV1Interface, error) {
	goClient, err := GetAPIExtGoClient(config)
	if err != nil {
		return nil, err
	}
	return goClient.ApiextensionsV1(), nil
}

// GetAppsV1Func is the function the AppsV1Interface
var GetAppsV1Func = GetAppsV1Client

// GetAppsV1Client returns the AppsV1Interface
func GetAppsV1Client(config *rest.Config) (appsv1.AppsV1Interface, error) {
	goClient, err := GetGoClient(config)
	if err != nil {
		return nil, err
	}
	return goClient.AppsV1(), nil
}

// GetGoClient returns a go-client
func GetGoClient(config *rest.Config) (kubernetes.Interface, error) {
	if fakeClient != nil {
		return fakeClient, nil
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return kubeClient, err
}

// GetAPIExtGoClient returns an API Extensions go-client
func GetAPIExtGoClient(config *rest.Config) (apiextv1Client.Interface, error) {
	apiextClient, err := apiextv1Client.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return apiextClient, err
}

// GetDynamicClient returns a dynamic client
func GetDynamicClient(config *rest.Config) (dynamic.Interface, error) {
	cli, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return cli, err
}
