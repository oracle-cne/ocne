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
	return &ImageCapability{
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
			ShapeCompatibilities: []ShapeCompatibility{
				{
					InternalShapeName: "VM.DenseIO1.16",
				},
				{
					InternalShapeName: "VM.DenseIO1.4",
				},
				{
					InternalShapeName: "VM.DenseIO1.8",
				},
				{
					InternalShapeName: "VM.DenseIO2.16",
				},
				{
					InternalShapeName: "VM.DenseIO2.24",
				},
				{
					InternalShapeName: "VM.DenseIO2.8",
				},
				{
					InternalShapeName: "VM.GPU2.1",
				},
				{
					InternalShapeName: "VM.GPU3.1",
				},
				{
					InternalShapeName: "VM.GPU3.2",
				},
				{
					InternalShapeName: "VM.GPU3.4",
				},
				{
					InternalShapeName: "VM.Standard.B1.1",
				},
				{
					InternalShapeName: "VM.Standard.B1.16",
				},
				{
					InternalShapeName: "VM.Standard.B1.2",
				},
				{
					InternalShapeName: "VM.Standard.B1.4",
				},
				{
					InternalShapeName: "VM.Standard.B1.8",
				},
				{
					InternalShapeName: "VM.Standard.E2.1",
				},
			},
		},
	}
}

func arm64Capabilities() *ImageCapability {
	return &ImageCapability{}
}
