// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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
	if !cad.ClusterConfig.Providers.Olvm.InstallCsiDriver {
		return nil, nil
	}

	compartmentId, err := oci.GetCompartmentId(cad.ClusterConfig.Providers.Oci.Compartment, cad.ClusterConfig.Providers.Oci.Profile)
	if err != nil {
		return nil, err
	}

	authCreds := map[string]interface{}{
		"auth": map[string]interface{}{
			"region":                ociConfig.Region,
			"tenancy":               ociConfig.Tenancy,
			"user":                  ociConfig.User,
			"key":                   ociConfig.Key,
			"passphrase":            ociConfig.Passphrase,
			"fingerprint":           ociConfig.Fingerprint,
			"useInstancePrincipals": ociConfig.UseInstancePrincipal,
		},
		"compartment": compartmentId,
		"vcn":         cad.ClusterConfig.Providers.Oci.Vcn,
		"loadBalancer": map[string]interface{}{
			"subnet1":                    cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet1,
			"subnet2":                    cad.ClusterConfig.Providers.Oci.LoadBalancer.Subnet2,
			"securityListManagementMode": "None",
		},
	}
	authCredBytes, err := yaml.Marshal(authCreds)
	if err != nil {
		return nil, err
	}

	ociCcmCreds := map[string][]byte{
		"cloud-provider.yaml": authCredBytes,
	}
	ociCsiCreds := map[string][]byte{
		"config.yaml": authCredBytes,
	}

	ret := []install.ApplicationDescription{
		install.ApplicationDescription{
			PreInstall: func() error {

				secretName := cad.credSecretName()
				k8s.DeleteSecret(kubeClient, cad.ClusterConfig.Providers.Olvm.Namespace, secretName)
				err = k8s.CreateSecret(kubeClient, cad.ClusterConfig.Providers.Olvm.Namespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: cad.ClusterConfig.Providers.Olvm.Namespace,
					},
					Data: credmap,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				// get the CA
				ca, err := GetCA(&cad.ClusterConfig.Providers.Olvm)
				if err != nil {
					return err
				}

				err := k8s.CreateSecret(kubeClient, OciCcmNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: OciCcmSecretName,
					},
					Data: ociCcmCreds,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				err = k8s.CreateSecret(kubeClient, OciCcmNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: OciCcmCsiSecretName,
					},
					Data: ociCsiCreds,
					Type: "Opaque",
				})
				return err
			},

			//type OvirtCsiDriver struct {
			//	CsiDriverName        string `yaml:"csiDriverName"`
			//	CaProvided           bool   `yaml:"caProvidedFake"`
			//	CaProvidedPtr        *bool  `yaml:"caProvided,omitempty"`
			//	SecretName           string `yaml:"credsSecretName"`
			//	ConfigMapName        string `yaml:"caConfigmapName"`
			//	NodePluginName       string `yaml:"nodePluginName"`
			//	ControllerPluginName string `yaml:"controllerPluginName"`
			//}
			Application: &types.Application{
				Name:      OciCcmChart,
				Namespace: OciCcmNamespace,
				Release:   OciCcmRelease,
				Version:   OciCcmVersion,
				Catalog:   catalog.InternalCatalog,
			},
		},
	}

	return ret, nil
}
