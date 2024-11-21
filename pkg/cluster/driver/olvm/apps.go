package olvm

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

func (cad *ClusterApiDriver) getApplications(kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	proxyValues := map[string]interface{}{
		"httpsProxy": cad.ClusterConfig.Providers.Olvm.Proxy.HttpsProxy,
		"httpProxy":  cad.ClusterConfig.Providers.Oci.Proxy.HttpProxy,
		"noProxy":    cad.ClusterConfig.Providers.Oci.Proxy.NoProxy,
	}

	// Get the creds
	credmap, err := getCreds()
	if err != nil {
		return nil, err
	}
	// get the CA
	ca, err := getCA(&cad.ClusterConfig.Providers.Olvm)
	if err != nil {
		return nil, err
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
			PreInstall: func() error {
				err := k8s.CreateSecret(kubeClient, constants.OLVMCAPIOperatorNamespace, &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", cad.ClusterConfig.Name, constants.OLVMOVirtCredSecretSuffix),
						Namespace: constants.OLVMCAPIOperatorNamespace,
					},
					Data: credmap,
					Type: "Opaque",
				})
				if err != nil {
					return err
				}

				err = k8s.CreateConfigmap(kubeClient, &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", cad.ClusterConfig.Name, constants.OLVMOVirtCAConfigMapSuffix),
						Namespace: constants.OLVMCAPIOperatorNamespace,
					},
					Data: map[string]string{
						"ca.crt": ca,
					},
				})
				return err
			},
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

func (cad *ClusterApiDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	return nil, nil
}

func getCA(prov *types.OlvmProvider) (string, error) {
	if prov.OlvmCluster.OVirtApiCA != "" && prov.OlvmCluster.OVirtApiCAPath != "" {
		return "", fmt.Errorf("The OLVM Provider cannot specify both ovirtApiCA and ovirtApiCAPath")
	}
	if prov.OlvmCluster.OVirtApiCA != "" {
		return prov.OlvmCluster.OVirtApiCA, nil
	}

	if prov.OlvmCluster.OVirtApiCAPath != "" {
		by, err := os.ReadFile(prov.OlvmCluster.OVirtApiCAPath)
		if err != nil {
			return "", fmt.Errorf("Error reading OLVM Provider oVirt CA from %s: %v", prov.OlvmCluster.OVirtApiCAPath, err)
		}
		return string(by), nil
	}
	return "", fmt.Errorf("The OLVM Provider must specify ovirtApiCA or ovirtApiCAPath")
}

func getCreds() (map[string][]byte, error) {
	username := os.Getenv(EnvUsername)
	if username == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM username")
	}
	password := os.Getenv(EnvPassword)
	if password == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM password")
	}
	scope := os.Getenv(EnvScope)
	if scope == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM username")
	}

	return map[string][]byte{
		"username": []byte(username),
		"password": []byte(password),
		"scope":    []byte(scope),
	}, nil

}
