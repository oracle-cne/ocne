// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

const KubeApiServerPort = "6443"

// EnvVarKubeConfig Name of Environment Variable for KUBECONFIG
const EnvVarKubeConfig = "KUBECONFIG"

// EnvVarTestKubeConfig Name of Environment Variable for test KUBECONFIG
const EnvVarTestKubeConfig = "TEST_KUBECONFIG"

const APIServerBurst = 150
const APIServerQPS = 100

type ClientConfigFunc func() (*rest.Config, kubernetes.Interface, error)

// fakeClient is for unit testing
var fakeClient kubernetes.Interface

// SetFakeClient for unit tests
func SetFakeClient(client kubernetes.Interface) {
	fakeClient = client
}

// ClearFakeClient for unit tests
func ClearFakeClient() {
	fakeClient = nil
}

func GetKubeconfigPath(filename string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", filename), nil
}

// sanitizePath converts the input path to an absolute path
// and check if the file exists.  If it does not exist, an error
// is returned.  The second boolean argument is returned unchanged
// and exists only to match the return signature of GetKubeConfigLocation.
func sanitizePath(path string, echo bool) (string, bool, error) {
	log.Debugf("Sanitizing %s", path)
	path, err := filepath.Abs(path)
	if err != nil {
		return path, echo, err
	}

	_, err = os.Stat(path)
	if err != nil {
		return path, echo, err
	}

	return path, echo, nil
}

// GetKubeConfigLocation Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation(kubeconfigPath string) (string, bool, error) {
	if kubeconfigPath != "" {
		return sanitizePath(kubeconfigPath, false)
	}

	if testKubeConfig := os.Getenv(EnvVarTestKubeConfig); len(testKubeConfig) > 0 {
		path, echo, err := sanitizePath(testKubeConfig, false)
		if err != nil {
			err = fmt.Errorf("Failed to access the kubeconfig set by the environment variable %s: ", EnvVarTestKubeConfig)
		}
		return path, echo, err
	}

	if kubeConfig := os.Getenv(EnvVarKubeConfig); len(kubeConfig) > 0 {
		path, echo, err := sanitizePath(kubeConfig, false)
		if err != nil {
			err = fmt.Errorf("Failed to access the kubeconfig set by the environment variable %s: ", EnvVarKubeConfig)
		}
		return path, echo, err
	}

	if home := homedir.HomeDir(); home != "" {
		return sanitizePath(filepath.Join(home, ".kube", "config"), true)
	}

	return "", true, errors.New("unable to find kubeconfig")

}

// GetKubeConfigGivenPath GetKubeConfig will get the kubeconfig from the given kubeconfigPath
func GetKubeConfigGivenPath(kubeconfigPath string) (*rest.Config, error) {
	return BuildKubeConfig(kubeconfigPath)
}

func BuildKubeConfig(kubeconfig string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	setConfigQPSBurst(config)
	return config, nil
}

// GetKubeConfig Returns kubeconfig from KUBECONFIG env var if set
// Else from default location ~/.kube/config
func GetKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	var config *rest.Config

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	config.TLSClientConfig.Insecure = true
	config.CAFile = ""
	config.CAData = []byte{}

	setConfigQPSBurst(config)
	return config, nil
}

// GetKubeConfigGivenPathAndContext returns a rest.Config given a kubeConfig and kubeContext.
func GetKubeConfigGivenPathAndContext(kubeConfigPath string, kubeContext string) (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext}).ClientConfig()
	if err != nil {
		return nil, err
	}
	setConfigQPSBurst(config)
	return config, nil
}

// GetKubeConfigFromString returns a kubeconfig from an in-memory string
func GetKubeConfigFromString(kcfg string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromKubeconfigGetter("", func()(*clientcmdapi.Config, error){
		return clientcmd.Load([]byte(kcfg))
	})
}

func setConfigQPSBurst(config *rest.Config) {
	config.Burst = APIServerBurst
	config.QPS = APIServerQPS
}

// CreateKubeInfo returns a kubeInfo struct from a path to a kubeConfig
func CreateKubeInfo(kubeConfigPath string) (*KubeInfo, error) {
	restConfig, _, err := GetKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	path, _, err := GetKubeConfigLocation(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	kubeInfo := KubeInfo{KubeconfigPath: path, RestConfig: restConfig}
	return &kubeInfo, nil
}

// GetNamespaceFromConfig gets the namespace that is set in the current context of the kubeConfig file
func GetNamespaceFromConfig(kubeConfigPath string) (string, error) {
	path, _, err := GetKubeConfigLocation(kubeConfigPath)
	if err != nil {
		return "", err
	}
	bytesOfKubeConfig, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	config, err := clientcmd.Load(bytesOfKubeConfig)
	if err != nil {
		return "", err
	}
	currentContextName := config.CurrentContext
	contexts := config.Contexts
	currentContext := contexts[currentContextName]
	if currentContext == nil || currentContext.Namespace == "" {
		return "default", nil
	}
	return currentContext.Namespace, nil
}
