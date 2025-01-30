// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/commands/application/install"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// getApplications gets the applications that are needed for the OLVM provider.
// These applications will be installed in the bootstrap cluster
func (cad *OlvmDriver) getApplications() ([]install.ApplicationDescription, error) {
	proxyValues := map[string]interface{}{
		"httpsProxy": *cad.ClusterConfig.Providers.Olvm.Proxy.HttpsProxy,
		"httpProxy":  *cad.ClusterConfig.Providers.Oci.Proxy.HttpProxy,
		"noProxy":    *cad.ClusterConfig.Providers.Oci.Proxy.NoProxy,
	}
	certManagerChart := constants.CertManagerChart
	certManagerNamespace := constants.CertManagerNamespace
	certManagerRelease := constants.CertManagerRelease
	certManagerVersion := constants.CertManagerVersion
	internalCatalog := catalog.InternalCatalog

	coreCAPIChart := constants.CoreCAPIChart
	coreCAPINamespace := constants.CoreCAPINamespace
	coreCAPIRelease := constants.CoreCAPIRelease
	coreCAPIVersion := constants.CoreCAPIVersion

	olvmChart := constants.OLVMCAPIChart
	olvmOperatorNamesapce := constants.OLVMCAPIOperatorNamespace
	olvmRelease := constants.OLVMCAPIRelease
	olvmVersion := constants.OLVMCAPIVersion

	kubeadmBootstrapChart := constants.KubeadmBootstrapCAPIChart
	kubeadmBootstrapNamespace := constants.KubeadmBootstrapCAPINamespace
	kubeadmBootstrapRelease := constants.KubeadmBootstrapCAPIRelease
	kubeadmBootstrapVersion := constants.KubeadmBootstrapCAPIVersion

	kubeadmControlPlaneChart := constants.KubeadmControlPlaneCAPIChart
	kubeadmControlPlaneNamespace := constants.KubeadmControlPlaneCAPINamespace
	kubeadmControlPlaneRelease := constants.KubeadmControlPlaneCAPIRelease
	kubeadmControlPlaneVersion := constants.KubeadmBootstrapCAPIVersion

	return []install.ApplicationDescription{
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      &certManagerChart,
				Namespace: &certManagerNamespace,
				Release:   &certManagerRelease,
				Version:   &certManagerVersion,
				Catalog:   &internalCatalog,
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      &coreCAPIChart,
				Namespace: &coreCAPINamespace,
				Release:   &coreCAPIRelease,
				Version:   &coreCAPIVersion,
				Catalog:   &internalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      &olvmChart,
				Namespace: &olvmOperatorNamesapce,
				Release:   &olvmRelease,
				Version:   &olvmVersion,
				Catalog:   &internalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      &kubeadmBootstrapChart,
				Namespace: &kubeadmBootstrapNamespace,
				Release:   &kubeadmBootstrapRelease,
				Version:   &kubeadmBootstrapVersion,
				Catalog:   &internalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
		install.ApplicationDescription{
			Application: &types.Application{
				Name:      &kubeadmControlPlaneChart,
				Namespace: &kubeadmControlPlaneNamespace,
				Release:   &kubeadmControlPlaneRelease,
				Version:   &kubeadmControlPlaneVersion,
				Catalog:   &internalCatalog,
				Config: map[string]interface{}{
					"proxy": proxyValues,
				},
			},
		},
	}, nil
}

// getWorkloadClusterApplications gets the applications that need to be installed into the new CAPI cluster
func (cad *OlvmDriver) getWorkloadClusterApplications(restConfig *rest.Config, kubeClient kubernetes.Interface) ([]install.ApplicationDescription, error) {
	return nil, nil
}
