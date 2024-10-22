// Copyright (c) 2024, Oracle and/or its affiliates.
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
	EphemeralNodeStorage         = "20Gi"
	EphemeralClusterName         = "ocne-ephemeral"
	EphemeralClusterPreserve     = true

	// Kubernetes defaults`
	KubeAPIServerBindPort    = uint16(6443)
	KubeAPIServerBindPortAlt = uint16(6444)
	PodSubnet                = "10.244.0.0/16"
	ServiceSubnet            = "10.96.0.0/12"
	InstanceMetadata         = "169.254.169.254"
	ContainerRegistry        = "container-registry.oracle.com"

	CertKey = "tls.crt"
	PrivKey = "tls.key"

	KubeVersion = "1.30"

	OciControlPlaneOcpus = 2
	OciWorkerOcpus       = 4
	OciImageName         = "ock"
	OciBucket            = "ocne-images"

	OciVmStandardE4Flex = "VM.Standard.E4.Flex"

	// OCI shapes compatible with ARM images
	OciVmStandardA1Flex = "VM.Standard.A1.Flex"
	OciBmStandardA1160  = "BM.Standard.A1.160"

	// OCI Image Identifier Tags
	OCIArchitectureTag      = "ocne/architecture"
	OCIKubernetesVersionTag = "ocne/kubernetes"

	// OCNE annotations
	OCNEAnnoUpdateAvailable = "ocne.oracle.com/update-available"

	// Kubernetes Labels
	K8sLabelControlPlane = "node-role.kubernetes.io/control-plane"

	// Provider Types
	ProviderTypeLibVirt = "libvirt"
	ProviderTypeOCI     = "oci"
	ProviderTypeNone    = "none"
)

var OciArmCompatibleShapes = [...]string{OciVmStandardA1Flex, OciBmStandardA1160}
