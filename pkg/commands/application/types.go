// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package application

import (
	"github.com/oracle-cne/ocne/pkg/catalog"
	"github.com/oracle-cne/ocne/pkg/helm"
)

// LsOptions are the options for the application ls command
type LsOptions struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// Namespace is the namespace that should be searched
	Namespace string

	// All indicates whether all namespaces should be searched
	All bool
}

// UninstallOptions are the options for the application uninstall command
type UninstallOptions struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// Namespace is the namespace that should be searched
	Namespace string

	// ReleaseName is the release name that should be uninstalled
	ReleaseName string
}

// ShowOptions are the options for the application show command
type ShowOptions struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// Namespace is the namespace that should be searched
	Namespace string

	// ReleaseName is the name of the release that you want to gather information about
	ReleaseName string

	// Computed indicates whether the user wants the computed (overrides + default) values.yaml file to be given as output
	Computed bool

	// Difference indicates whether the user wants the default (override only) output and the generated output to both
	// be outputted for comparison by the user
	Difference bool
}

// InstallOptions are the options for the application install command
type InstallOptions struct {

	// Catalog is the catalog object used to gather information about the app-catalog
	Catalog *catalog.Catalog

	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// AppName is the name of the application
	AppName string

	// Namespace is the namespace to install the application
	Namespace string

	// Version is the version of the application to install
	Version string

	// ReleaseName is the release name of the instance of the application
	ReleaseName string

	// Values is the path to a file containing helm values that will be used for overrides
	Values string

	// Overrides is a list of overrides that get munged together.
	// Later values take precedence over earlier ones.
	Overrides []helm.HelmOverrides

	// ResetValues is used to reset the values to the ones built into the chart.
	ResetValues bool

	// Force causes the application to overwrite and take ownership
	// of existing resources.
	Force bool
}
type UpdateOptions struct {

	// Catalog is the name of the catalog used to gather information about the app-catalog
	CatalogName string

	// ReleaseName is the release name of the instance of the application
	ReleaseName string

	// Version is the version of the application to update to
	Version string

	// Namespace is the namespace that contains the application
	Namespace string

	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// Values is the path to a file containing helm values that will be used for overrides
	Values string

	// ResetValues is used to reset the values to the ones built into the chart.
	ResetValues bool
}

// TemplateOptions are the options for the application template command
type TemplateOptions struct {

	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// AppName is the name of the application
	AppName string

	// Version is the version of the application to install
	Version string

	// Interactive indicates whether the rendered template should appear in the text editor defined by the EDITOR environment variable
	Interactive bool

	// Catalog is the name of the catalog to search for template values
	Catalog string
}
