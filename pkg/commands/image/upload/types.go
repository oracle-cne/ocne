// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"github.com/oracle-cne/ocne/pkg/config/types"
	"io"
)

// UploadOptions are the options for the upload image command
type UploadOptions struct {
	// KubeConfigPath is the optional path to the kubeconfig file
	KubeConfigPath string

	// ClusterConfig is the cluster config.
	// This is optional and only needed by OLVM for now
	ClusterConfig *types.ClusterConfig

	// ProviderConfigPath is the path for the provider config (e.g ~/.oci/config)
	ProviderConfigPath string

	// ProviderType is the provider type (e.g. oci)
	ProviderType string

	// ImagePath is the path of the local boot image
	ImagePath string

	// BucketName is the bucket where the image will get uploaded
	BucketName string

	// CompartmentName is the compartment where the image will get upload
	CompartmentName string

	// ImageName is the name of the custom image to create
	ImageName string

	// KubernetesVersion is the version of Kubernetes embedded
	// in the image to upload
	KubernetesVersion string

	// ImageArchitecture is the architecture of the image to upload
	ImageArchitecture string

	// Destination is the place to upload the image
	Destination string

	compartmentId string
	filename      string
	size          int64
	file          io.ReadCloser
}

// OlvmUploadOptions are the options for the upload image command
type OlvmUploadOptions struct {
	// ProviderConfigPath is the path for the provider config (e.g ~/.oci/config)
	ProviderConfigPath string

	// ProviderType is the provider type (e.g. oci)
	ProviderType string

	// ImagePath is the path of the local boot image
	ImagePath string

	// BucketName is the bucket where the image will get uploaded
	BucketName string

	// CompartmentName is the compartment where the image will get upload
	CompartmentName string

	// ImageName is the name of the custom image to create
	ImageName string

	// KubernetesVersion is the version of Kubernetes embedded
	// in the image to upload
	KubernetesVersion string

	// ImageArchitecture is the architecture of the image to upload
	ImageArchitecture string

	// Destination is the place to upload the image
	Destination string

	compartmentId string
	filename      string
	size          int64
	file          io.ReadCloser
}
