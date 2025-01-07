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
	return nil, nil
}
