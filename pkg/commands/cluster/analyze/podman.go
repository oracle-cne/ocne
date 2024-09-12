// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import "time"

// PodmanImageData has the summary of all images in the cluster
// Each map is the nodename as the key
type PodmanImageData struct {
	PodmanInfoMap       map[string]*PodmanInfo
	ImageDetailMap      map[string][]*PodmanImageDetail
	ImageDetailErrorMap map[string]string
	ImageMap            map[string][]*PodmanImage
}

type PodmanImage struct {
	Id          string      `json:"Id"`
	ParentId    string      `json:"ParentId"`
	RepoTags    interface{} `json:"RepoTags"`
	RepoDigests []string    `json:"RepoDigests"`
	Size        int         `json:"Size"`
	SharedSize  int         `json:"SharedSize"`
	VirtualSize int         `json:"VirtualSize"`
	Labels      struct {
		IoBuildahVersion string `json:"io.buildah.version"`
	} `json:"Labels"`
	Containers int       `json:"Containers"`
	ReadOnly   bool      `json:"ReadOnly"`
	Names      []string  `json:"Names"`
	Digest     string    `json:"Digest"`
	History    []string  `json:"History"`
	Created    int       `json:"Created"`
	CreatedAt  time.Time `json:"CreatedAt"`
}

type PodmanInfo struct {
	Host struct {
		Arch              string   `json:"arch"`
		BuildahVersion    string   `json:"buildahVersion"`
		CgroupManager     string   `json:"cgroupManager"`
		CgroupVersion     string   `json:"cgroupVersion"`
		CgroupControllers []string `json:"cgroupControllers"`
		Conmon            struct {
			Package string `json:"package"`
			Path    string `json:"path"`
			Version string `json:"version"`
		} `json:"conmon"`
		Cpus           int `json:"cpus"`
		CpuUtilization struct {
			UserPercent   float64 `json:"userPercent"`
			SystemPercent float64 `json:"systemPercent"`
			IdlePercent   float64 `json:"idlePercent"`
		} `json:"cpuUtilization"`
		DatabaseBackend string `json:"databaseBackend"`
		Distribution    struct {
			Distribution string `json:"distribution"`
			Variant      string `json:"variant"`
			Version      string `json:"version"`
		} `json:"distribution"`
		EventLogger string `json:"eventLogger"`
		FreeLocks   int    `json:"freeLocks"`
		Hostname    string `json:"hostname"`
		IdMappings  struct {
			Gidmap interface{} `json:"gidmap"`
			Uidmap interface{} `json:"uidmap"`
		} `json:"idMappings"`
		Kernel             string `json:"kernel"`
		LogDriver          string `json:"logDriver"`
		MemFree            int    `json:"memFree"`
		MemTotal           int64  `json:"memTotal"`
		NetworkBackend     string `json:"networkBackend"`
		NetworkBackendInfo struct {
			Backend string `json:"backend"`
			Package string `json:"package"`
			Path    string `json:"path"`
			Dns     struct {
				Version string `json:"version"`
				Package string `json:"package"`
				Path    string `json:"path"`
			} `json:"dns"`
		} `json:"networkBackendInfo"`
		OciRuntime struct {
			Name    string `json:"name"`
			Package string `json:"package"`
			Path    string `json:"path"`
			Version string `json:"version"`
		} `json:"ociRuntime"`
		Os           string `json:"os"`
		RemoteSocket struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"remoteSocket"`
		ServiceIsRemote bool `json:"serviceIsRemote"`
		Security        struct {
			ApparmorEnabled    bool   `json:"apparmorEnabled"`
			Capabilities       string `json:"capabilities"`
			Rootless           bool   `json:"rootless"`
			SeccompEnabled     bool   `json:"seccompEnabled"`
			SeccompProfilePath string `json:"seccompProfilePath"`
			SelinuxEnabled     bool   `json:"selinuxEnabled"`
		} `json:"security"`
		Slirp4Netns struct {
			Executable string `json:"executable"`
			Package    string `json:"package"`
			Version    string `json:"version"`
		} `json:"slirp4netns"`
		Pasta struct {
			Executable string `json:"executable"`
			Package    string `json:"package"`
			Version    string `json:"version"`
		} `json:"pasta"`
		SwapFree  int    `json:"swapFree"`
		SwapTotal int    `json:"swapTotal"`
		Uptime    string `json:"uptime"`
		Variant   string `json:"variant"`
		Linkmode  string `json:"linkmode"`
	} `json:"host"`
	Store struct {
		ConfigFile     string `json:"configFile"`
		ContainerStore struct {
			Number  int `json:"number"`
			Paused  int `json:"paused"`
			Running int `json:"running"`
			Stopped int `json:"stopped"`
		} `json:"containerStore"`
		GraphDriverName string `json:"graphDriverName"`
		GraphOptions    struct {
			OverlayImagestore string `json:"overlay.imagestore"`
			OverlayMountopt   string `json:"overlay.mountopt"`
		} `json:"graphOptions"`
		GraphRoot          string `json:"graphRoot"`
		GraphRootAllocated int64  `json:"graphRootAllocated"`
		GraphRootUsed      int64  `json:"graphRootUsed"`
		GraphStatus        struct {
			BackingFilesystem string `json:"Backing Filesystem"`
			NativeOverlayDiff string `json:"Native Overlay Diff"`
			SupportsDType     string `json:"Supports d_type"`
			SupportsShifting  string `json:"Supports shifting"`
			SupportsVolatile  string `json:"Supports volatile"`
			UsingMetacopy     string `json:"Using metacopy"`
		} `json:"graphStatus"`
		ImageCopyTmpDir string `json:"imageCopyTmpDir"`
		ImageStore      struct {
			Number int `json:"number"`
		} `json:"imageStore"`
		RunRoot        string `json:"runRoot"`
		VolumePath     string `json:"volumePath"`
		TransientStore bool   `json:"transientStore"`
	} `json:"store"`
	Registries struct {
		Search []string `json:"search"`
	} `json:"registries"`
	Plugins struct {
		Volume        []string    `json:"volume"`
		Network       []string    `json:"network"`
		Log           []string    `json:"log"`
		Authorization interface{} `json:"authorization"`
	} `json:"plugins"`
	Version struct {
		APIVersion string `json:"APIVersion"`
		Version    string `json:"Version"`
		GoVersion  string `json:"GoVersion"`
		GitCommit  string `json:"GitCommit"`
		BuiltTime  string `json:"BuiltTime"`
		Built      int    `json:"Built"`
		OsArch     string `json:"OsArch"`
		Os         string `json:"Os"`
	} `json:"version"`
}

type PodmanImageDetail struct {
	Id          string    `json:"Id"`
	Digest      string    `json:"Digest"`
	RepoTags    []string  `json:"RepoTags"`
	RepoDigests []string  `json:"RepoDigests"`
	Parent      string    `json:"Parent"`
	Comment     string    `json:"Comment"`
	Created     time.Time `json:"Created"`
	Config      struct {
		Env    []string `json:"Env"`
		Cmd    []string `json:"Cmd"`
		Labels struct {
			IoBuildahVersion string `json:"io.buildah.version"`
		} `json:"Labels"`
	} `json:"Config"`
	Version      string `json:"Version"`
	Author       string `json:"Author"`
	Architecture string `json:"Architecture"`
	Os           string `json:"Os"`
	Size         int    `json:"Size"`
	VirtualSize  int    `json:"VirtualSize"`
	GraphDriver  struct {
		Name string `json:"Name"`
		Data struct {
			LowerDir string `json:"LowerDir"`
			UpperDir string `json:"UpperDir"`
			WorkDir  string `json:"WorkDir"`
		} `json:"Data"`
	} `json:"GraphDriver"`
	RootFS struct {
		Type   string   `json:"Type"`
		Layers []string `json:"Layers"`
	} `json:"RootFS"`
	Labels struct {
		IoBuildahVersion string `json:"io.buildah.version"`
	} `json:"Labels"`
	Annotations struct {
	} `json:"Annotations"`
	ManifestType string `json:"ManifestType"`
	User         string `json:"User"`
	History      []struct {
		Created    time.Time `json:"created"`
		CreatedBy  string    `json:"created_by"`
		EmptyLayer bool      `json:"empty_layer,omitempty"`
	} `json:"History"`
	NamesHistory []string `json:"NamesHistory"`
}
