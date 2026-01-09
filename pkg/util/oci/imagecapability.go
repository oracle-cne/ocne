// Copyright (c) 2025, 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

type ImageCapability struct {
	Version                int                   `json:"version"`
	ExternalLaunchOptions  ExternalLaunchOptions `json:"externalLaunchOptions"`
	ImageCapabilityData    map[string]Descriptor `json:"imageCapabilityData"`
	ImageCapsFormatVersion string                `json:"imageCapsFormatVersion"`
	OperatingSystem        string                `json:"operatingSystem"`
	OperatingSystemVersion string                `json:"operatingSystemVersion"`
	AdditionalMetadata     AdditionalMetadata    `json:"additionalMetadata"`
}

type ExternalLaunchOptions struct {
	Firmware                      string `json:"firmware"`
	NetworkType                   string `json:"networkType"`
	BootVolumeType                string `json:"bootVolumeType"`
	RemoteDataVolumeType          string `json:"remoteDataVolumeType"`
	LocalDataVolumeType           string `json:"localDataVolumeType"`
	LaunchOptionsSource           string `json:"launchOptionsSource"`
	PvAttachmentVersion           int    `json:"pvAttachmentVersion"`
	PvEncryptionInTransitEnabled  bool   `json:"pvEncryptionInTransitEnabled"`
	ConsistentVolumeNamingEnabled bool   `json:"consistentVolumeNamingEnabled"`
}

type AdditionalMetadata struct {
	SourcePublicImageId  string               `json:"sourcePublicImageId,omitempty"`
	ShapeCompatibilities []ShapeCompatibility `json:"shapeCompatibilities"`
}

type ShapeCompatibility struct {
	InternalShapeName string `json:"internalShapeName"`
	OcpuConstraints   string `json:"ocpuConstraints,omitempty"`
	MemoryConstraints string `json:"memoryConstraints,omitempty"`
}
type ImageArch string

const (
	AMD64 ImageArch = "amd64"
	ARM64 ImageArch = "arm64"
)

type Descriptor struct {
	DefaultValue   interface{} `json:"defaultValue"`
	DescriptorType string      `json:"descriptorType"`
	Values         []string    `json:"values,omitempty"` // omit if not present
}

var amd64ImageShapesOCI = []string{"BM.Standard2.52", "BM.DenseIO.E4.128", "BM.Standard.E4.128", "BM.Standard.E3.128", "BM.Standard.E2.64", "BM.DenseIO2.52",
	"BM.GPU2.2", "BM.Standard1.36", "BM.Standard.B1.44", "VM.Standard.E4.Flex", "VM.Standard.E3.Flex", "VM.Standard2.1", "VM.Standard2.2", "VM.Standard2.4",
	"VM.Standard2.8", "VM.Standard2.16", "VM.Standard2.24", "VM.Standard.E2.1", "VM.Standard.E2.2", "VM.Standard.E2.4", "VM.Standard.E2.8", "VM.Standard.E2.1.Micro",
	"VM.Standard3.Flex", "VM.DenseIO2.8", "VM.DenseIO2.16", "VM.DenseIO2.24", "VM.GPU2.1", "VM.Standard1.1", "VM.Standard1.2", "VM.Standard1.4", "VM.Standard1.8",
	"VM.Standard1.16", "VM.Standard.B1.1", "VM.Standard.B1.2", "VM.Standard.B1.4", "VM.Standard.B1.8", "VM.Standard.B1.16", "VM.Standard.E5.Flex",
	"VM.Standard.E6.Flex", "BM.GPU3.8", "VM.GPU3.1", "VM.GPU3.2", "VM.GPU3.4", "BM.HPC2.36", "BM.GPU4.8"}

var arm64ImageShapesOCI = []string{"VM.Standard.A1.Flex", "VM.Standard.A2.Flex", "a1-2c.160.1024"}

var amd64ImageShapesPCA = []string{"VM.PCAStandard.E5.Flex"}

var arm64ImageShapesPCA []string

// NewImageCapability - generate image capabilities structure based on architecture and whether PCA or not
func NewImageCapability(imageArch ImageArch, isPCA bool) *ImageCapability {
	switch imageArch {
	case AMD64:
		return amd64Capabilities(isPCA)
	case ARM64:
		return arm64Capabilities(isPCA)
	}
	return &ImageCapability{}
}

// amd64Capabilities - create image capabilities for amd64
func amd64Capabilities(isPCA bool) *ImageCapability {
	if isPCA {
		return amd64CapabilitiesPCA()
	}
	return amd64CapabilitiesOCI()
}

// amd64CapabilitiesOCI - create image capabilities for amd64 OCI
func amd64CapabilitiesOCI() *ImageCapability {
	imageCapability := newOCICommonImageCapability()

	var shapeCapabilities []ShapeCompatibility
	for _, shape := range amd64ImageShapesOCI {
		shapeCapabilities = append(shapeCapabilities, ShapeCompatibility{InternalShapeName: shape})
	}
	imageCapability.AdditionalMetadata.ShapeCompatibilities = shapeCapabilities

	return imageCapability
}

// amd64CapabilitiesPCA - create image capabilities for amd64 PCA
func amd64CapabilitiesPCA() *ImageCapability {
	imageCapability := newPCACommonImageCapability()
	var shapeCapabilities []ShapeCompatibility
	for _, shape := range amd64ImageShapesPCA {
		shapeCapabilities = append(shapeCapabilities, ShapeCompatibility{InternalShapeName: shape})
	}
	return imageCapability
}

// arm64Capabilities - create image capabilities for arm64
func arm64Capabilities(isPCA bool) *ImageCapability {
	if isPCA {
		return arm64CapabilitiesPCA()
	}
	return arm64CapabilitiesOCI()

}

// arm64CapabilitiesOCI - create image capabilities for arm64 OCI
func arm64CapabilitiesOCI() *ImageCapability {
	imageCapability := newOCICommonImageCapability()

	var shapeCapabilities []ShapeCompatibility
	for _, shape := range arm64ImageShapesOCI {
		shapeCapabilities = append(shapeCapabilities, ShapeCompatibility{InternalShapeName: shape})
	}
	imageCapability.AdditionalMetadata.ShapeCompatibilities = shapeCapabilities

	return imageCapability
}

// arm64CapabilitiesPCA - create image capabilities for arm64 PCA
func arm64CapabilitiesPCA() *ImageCapability {
	imageCapability := newPCACommonImageCapability()
	var shapeCapabilities []ShapeCompatibility
	for _, shape := range arm64ImageShapesPCA {
		shapeCapabilities = append(shapeCapabilities, ShapeCompatibility{InternalShapeName: shape})
	}
	return imageCapability
}

func newOCICommonImageCapability() *ImageCapability {
	imageCapability := &ImageCapability{
		Version: 2,
		ExternalLaunchOptions: ExternalLaunchOptions{
			Firmware:                      "UEFI_64",
			NetworkType:                   "VFIO",
			BootVolumeType:                "ISCSI",
			RemoteDataVolumeType:          "PARAVIRTUALIZED",
			LocalDataVolumeType:           "VFIO",
			LaunchOptionsSource:           "NATIVE",
			PvAttachmentVersion:           1,
			PvEncryptionInTransitEnabled:  false,
			ConsistentVolumeNamingEnabled: false,
		},
		ImageCapsFormatVersion: "34dd2cea-aff2-4f45-b8f6-1cb5290bfab2",
		OperatingSystem:        "Oracle Linux",
		OperatingSystemVersion: "8",
		AdditionalMetadata:     AdditionalMetadata{},
	}

	data := map[string]Descriptor{
		"Compute.LaunchMode": {
			DescriptorType: "enumstring",
			Values:         []string{"PARAVIRTUALIZED"},
			DefaultValue:   "PARAVIRTUALIZED",
		},
		"Compute.Firmware": {
			DescriptorType: "enumstring",
			Values:         []string{"UEFI_64"},
			DefaultValue:   "UEFI_64",
		},
	}

	imageCapability.ImageCapabilityData = data

	return imageCapability
}

func newPCACommonImageCapability() *ImageCapability {
	return &ImageCapability{
		Version: 2,
		ExternalLaunchOptions: ExternalLaunchOptions{
			Firmware:                      "UEFI_64",
			NetworkType:                   "PARAVIRTUALIZED",
			BootVolumeType:                "PARAVIRTUALIZED",
			RemoteDataVolumeType:          "PARAVIRTUALIZED",
			LocalDataVolumeType:           "PARAVIRTUALIZED",
			LaunchOptionsSource:           "CUSTOM",
			PvEncryptionInTransitEnabled:  false,
			ConsistentVolumeNamingEnabled: false,
		},
		OperatingSystem:        "Oracle Linux",
		OperatingSystemVersion: "8",
	}
}
