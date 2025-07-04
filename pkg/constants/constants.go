// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	DarwinLibvirtSocketPath               = ".cache/libvirt/libvirt-sock"
	UserConfigDir                         = ".ocne"
	UserConfigDefaults                    = ".ocne/defaults.yaml"
	UserConfigDefaultsEnvironmentVariable = "OCNE_DEFAULTS"
	UserImageCacheDir                     = "images"
	UserContainerConfigDir                = "config"
	UserIPData                            = "ips.yaml"

	BootVolumeContainerImage = "docker://container-registry.oracle.com/olcne/ock"

	// Cluster Defaults
	ControlPlaneNodes = 1
	WorkerNodes       = 0

	// Libvirt Defaults
	StoragePool                  = "images"
	SessionURI                   = "qemu:///session"
	Network                      = "default"
	StoragePoolPath              = "/var/lib/libvirt/images"
	UserStoragePoolPath          = ".local/share/libvirt/images"
	BootVolumeName               = "boot.qcow2"
	BootVolumeContainerImagePath = "disk/boot.qcow2"
	NodeCPUs                     = 2
	NodeMemory                   = "4194304Ki"
	NodeStorage                  = "20Gi"
	EphemeralNodeStorage         = "30Gi"
	EphemeralClusterName         = "ocne-ephemeral"
	EphemeralClusterPreserve     = true
	SlirpSubnet                  = "192.18.255.0/24"

	// Kubernetes defaults`
	KubeAPIServerBindPort    = uint16(6443)
	KubeAPIServerBindPortAlt = uint16(6444)
	PodSubnet                = "10.244.0.0/16"
	ServiceSubnet            = "10.96.0.0/12"
	InstanceMetadata         = "169.254.169.254"
	ContainerRegistry        = "container-registry.oracle.com"

	CertKey = "tls.crt"
	PrivKey = "tls.key"

	KubeVersion = "1.32"

	OciControlPlaneOcpus = 2
	OciWorkerOcpus       = 4
	OciBootVolumeSize    = "50"
	OciImageName         = "ock"
	OciBucket            = "ocne-images"

	OciVmStandardE4Flex = "VM.Standard.E4.Flex"

	// OCI shapes compatible with ARM images
	OciVmStandardA1Flex = "VM.Standard.A1.Flex"
	OciVmStandardA2Flex = "VM.Standard.A2.Flex"
	OciBmStandardA1160  = "BM.Standard.A1.160"

	// OCI Image Identifier Tags
	OCIArchitectureTag      = "ocne/architecture"
	OCIKubernetesVersionTag = "ocne/kubernetes"

	// OCI Configuration Options
	OciDefaultProfile   = "DEFAULT"

	// OCNE annotations
	OCNEAnnoUpdateAvailable = "ocne.oracle.com/update-available"

	// Kubernetes Labels
	K8sLabelControlPlane = "node-role.kubernetes.io/control-plane"

	// Provider Types
	ProviderTypeLibvirt = "libvirt"
	ProviderTypeOCI     = "oci"
	ProviderTypeOlvm    = "olvm"
	ProviderTypeNone    = "none"

	CatalogMirror = "ocne.oracle.com/mirror"
)

var OciArmCompatibleShapes = [...]string{OciVmStandardA1Flex, OciBmStandardA1160, OciVmStandardA2Flex}
