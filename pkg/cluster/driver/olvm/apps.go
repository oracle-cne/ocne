// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"sigs.k8s.io/yaml"
)

const (
	// keys for csi driver secret
	csiDriverUsernameKey = "ovirt_username"
	csiDriverPasswordKey = "ovirt_password"
	csiDriverURLKey      = "ovirt_url"

	// keys for csi driver ca.crt
	csiDriverCaKey = "ca.crt"

	// keys for chart overrides

)

// this is used to marshal only the config that was overridden in the config file
type csiChartOverrides struct {
	CaProvided           bool   `yaml:"caProvided,omitempty"`
	ConfigMapName        string `yaml:"caConfigmapName,omitempty"`
	ControllerPluginName string `yaml:"controllerPluginName,omitempty"`
	CsiDriverName        string `yaml:"csiDriverName,omitempty"`
	NodePluginName       string `yaml:"nodePluginName,omitempty"`
	SecretName           string `yaml:"credsSecretName,omitempty"`
}

// getApplications gets the applications that are needed for the OLVM provider.
// These applications will be installed in the bootstrap cluster
func (cad *OlvmDriver) getApplications() ([]install.ApplicationDescription, error) {
	proxyValues := map[string]interface{}{
		"httpsProxy": cad.ClusterConfig.Providers.Olvm.Proxy.HttpsProxy,
		"httpProxy":  cad.ClusterConfig.Providers.Oci.Proxy.HttpProxy,
		"noProxy":    cad.ClusterConfig.Providers.Oci.Proxy.NoProxy,
	}

	return []install.ApplicationDescription{
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CertManagerChart,
				Namespace: constants.CertManagerNamespace,
				Release:   constants.CertManagerRelease,
				Version:   constants.CertManagerVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.CoreCAPIChart,
				Namespace: constants.CoreCAPINamespace,
				Release:   constants.CoreCAPIRelease,
				Version:   constants.CoreCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.OLVMCAPIChart,
				Namespace: constants.OLVMCAPIOperatorNamespace,
				Release:   constants.OLVMCAPIRelease,
				Version:   constants.OLVMCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.KubeadmBootstrapCAPIChart,
				Namespace: constants.KubeadmBootstrapCAPINamespace,
				Release:   constants.KubeadmBootstrapCAPIRelease,
				Version:   constants.KubeadmBootstrapCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      constants.KubeadmControlPlaneCAPIChart,
				Namespace: constants.KubeadmControlPlaneCAPINamespace,
				Release:   constants.KubeadmControlPlaneCAPIRelease,
				Version:   constants.KubeadmControlPlaneCAPIVersion,
				Catalog:   catalog.InternalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
	}, nil
}

// getWorkloadClusterApplications gets the applications that need to be installed into the new CAPI cluster
func (cad *OlvmDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	if !cad.ClusterConfig.Providers.Olvm.CSIDriver.Install {
		log.Debugf("OLVM installCsiDriver flag is false, skipping driver installation")
		return nil, nil
	}

	olvm := &cad.ClusterConfig.Providers.Olvm

	// load user overriders

	// get the creds
	ovirtCreds, err := getCreds()
	if err != nil {
		return nil, err
	}

	// Append /api to ovirt URL
	parsedURL, err := url.Parse(olvm.OlvmCluster.OVirtAPI.ServerURL)
	if err != nil {
		return nil, err
	}
	ovirtURL := parsedURL.JoinPath("/api")

	// set the creds needed by the ovirt csi driver
	credmap := map[string][]byte{
		csiDriverUsernameKey: []byte(ovirtCreds[credsUsernameKey]),
		csiDriverPasswordKey: []byte(ovirtCreds[credsPasswordKey]),
		csiDriverURLKey:      []byte(ovirtURL.String()),
	}

	// create chart overrides
	chartOverrides, err := getOverrides(olvm)
	if err != nil {
		return nil, err
	}

	// Specify pre-install function to create secret and configmap, then
	// Also specify function to install the csi driver chart
	namespace := olvm.CSIDriver.Namespace
	ret := []install.ApplicationDescription{
		install.ApplicationDescription{
			PreInstall: func() error {
				// Create the oVirt creds secret
				k8s.DeleteSecret(kubeClient, namespace, olvm.CSIDriver.SecretName)
				err = k8s.CreateSecret(kubeClient, namespace, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      olvm.CSIDriver.SecretName,
						Namespace: namespace,
					},
					Data: credmap,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				// create the CA.CRT configmap
				ca, err := GetCA(&cad.ClusterConfig.Providers.Olvm)
				if err != nil {
					return err
				}
				k8s.DeleteConfigmap(kubeClient, namespace, olvm.CSIDriver.ConfigMapName)
				err = k8s.CreateConfigmap(kubeClient, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      olvm.CSIDriver.ConfigMapName,
						Namespace: namespace,
					},
					Data: map[string]string{
						csiDriverCaKey: ca,
					},
				})
				return err
			},
			// install ovirt-csi-driver chart
			Application: &types.Application{
				Name:      constants.OvirtCsiChart,
				Namespace: namespace,
				Release:   constants.OvirtCsiRelease,
				Version:   constants.OvirtCsiVersion,
				Catalog:   catalog.InternalCatalog,
				Config:    chartOverrides,
			},
		},
	}

	return ret, nil
}

// return the user overrides as a map
func getOverrides(olvm *types.OlvmProvider) (map[string]interface{}, error) {
	ov := csiChartOverrides{}
	ov.CaProvided = olvm.CSIDriver.CaProvided
	if olvm.CSIDriver.ConfigMapName != "" {
		ov.ConfigMapName = olvm.CSIDriver.ConfigMapName
	}
	if olvm.CSIDriver.ControllerPluginName != "" {
		ov.ControllerPluginName = olvm.CSIDriver.ControllerPluginName
	}
	if olvm.CSIDriver.CsiDriverName != "" {
		ov.SecretName = olvm.CSIDriver.CsiDriverName
	}
	if olvm.CSIDriver.NodePluginName != "" {
		ov.NodePluginName = olvm.CSIDriver.NodePluginName
	}
	if olvm.CSIDriver.SecretName != "" {
		ov.SecretName = olvm.CSIDriver.SecretName
	}

	yamlValues, err := yaml.Marshal(ov)
	if err != nil {
		return nil, err
	}
	overrides := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(yamlValues), overrides)
	return overrides, err
}
