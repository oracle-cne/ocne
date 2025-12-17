package oci

type ImageCapability struct {
	Version                int                   `json:"version"`
	ExternalLaunchOptions  ExternalLaunchOptions `json:"externalLaunchOptions"`
	ImageCapabilityData    string                `json:"imageCapabilityData"`
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

var amd64ImageShapes = []string{"VM.DenseIO1.16", "VM.DenseIO1.4", "VM.DenseIO1.8", "VM.DenseIO2.16", "VM.DenseIO2.24", "VM.DenseIO2.8",
	"VM.GPU2.1", "VM.GPU3.1", "VM.GPU3.2", "VM.GPU3.4", "VM.Standard.B1.1", "VM.Standard.B1.16", "VM.Standard.B1.2", "VM.Standard.B1.4", "VM.Standard.B1.8",
	"VM.Standard.E2.1", "VM.Standard.E2.1.Micro", "VM.Standard.E2.2", "VM.Standard.E2.4", "VM.Standard.E2.8", "VM.Standard.E3.Flex"}

func NewImageCapability(imageArch ImageArch) *ImageCapability {
	switch imageArch {
	case AMD64:
		return amd64Capabilities()
	case ARM64:
		return arm64Capabilities()
	}
	return &ImageCapability{}
}

func amd64Capabilities() *ImageCapability {
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
			PvEncryptionInTransitEnabled:  true,
			ConsistentVolumeNamingEnabled: true,
		},
		ImageCapabilityData:    "",
		ImageCapsFormatVersion: "34dd2cea-aff2-4f45-b8f6-1cb5290bfab2",
		OperatingSystem:        "Oracle Linux",
		OperatingSystemVersion: "8",
		AdditionalMetadata: AdditionalMetadata{
			ShapeCompatibilities: []ShapeCompatibility{},
		},
	}

	shapeCapabilities := imageCapability.AdditionalMetadata.ShapeCompatibilities
	for _, shape := range amd64ImageShapes {
		shapeCapabilities = append(shapeCapabilities, ShapeCompatibility{InternalShapeName: shape})
	}

	return imageCapability
}

func arm64Capabilities() *ImageCapability {
	return &ImageCapability{}
}
