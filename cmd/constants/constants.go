// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	FlagKubeconfig      = "kubeconfig"
	FlagKubeconfigShort = "k"
	FlagKubeconfigHelp  = "the kubeconfig filepath"

	FlagSshURI      = "session"
	FlagSshURIShort = "s"
	FlagSshURIHelp  = "The URI for the libvirt session to connect to"

	FlagSshKey      = "key"
	FlagSshKeyShort = "i"
	FlagSshKeyHelp  = "the ssh public key of the remote system. Default is ~/.ssh/id_rsa.pub"

	FlagBootVolumeContainerImage      = "boot-volume-container-image"
	FlagBootVolumeContainerImageShort = "o"
	FlagBootVolumeContainerImageHelp  = "the URI of a container image that contains the OCK boot volume"

	FlagClusterName      = "cluster-name"
	FlagClusterNameShort = "C"
	FlagClusterNameHelp  = "A name for the created cluster"

	FlagProviderName      = "provider"
	FlagProviderNameShort = "P"
	FlagProviderNameHelp  = "The provider to use when interacting with the cluster"

	FlagAutoStartUIName      = "auto-start-ui"
	FlagAutoStartUINameShort = "u"
	FlagAutoStartUIHelp      = "Auto start a browser window to access the UI"

	FlagConfig      = "config"
	FlagConfigShort = "c"
	FlagConfigHelp  = "The path to a configuration file that contains the definition of the cluster to create. If this value is not provided, a small cluster is created using the default hypervisor for the system where the command is executed"
	FlagConfigHelp2 = "The path to a configuration file that contains the definition of the cluster"

	FlagVersionName  = "version"
	FlagVersionShort = "v"

	FlagKubernetesVersionHelp = "The version of Kubernetes"
)

const (
	OsRegistry = "ostree-unverified-registry:container-registry.oracle.com/olcne/ock-ostree"
)
