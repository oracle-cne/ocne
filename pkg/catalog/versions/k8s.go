// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package versions

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

type KubernetesVersions struct {
	Kubernetes string
	Pause      string
	Etcd       string
	CoreDNS    string
}

var kubernetesVersions = map[string]KubernetesVersions{
	"1.26.0": {
		Kubernetes: "1.26.0",
		Pause:      "3.9",
		Etcd:       "3.5.6",
		CoreDNS:    "current",
	},
	"1.27.0": {
		Kubernetes: "1.27.0",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "current",
	},
	"1.28.0": {
		Kubernetes: "1.28.0",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "current",
	},
	"1.29.0": {
		Kubernetes: "1.29.0",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "current",
	},
	"1.30.0": {
		Kubernetes: "1.30.0",
		Pause:      "3.9",
		Etcd:       "3.5.12",
		CoreDNS:    "current",
	},
}

func init() {
	// Add keys to map from the major.minor versions to major.minor.patch versions
	kubernetesVersions["1.26"] = kubernetesVersions["1.26.0"]
	kubernetesVersions["1.27"] = kubernetesVersions["1.27.0"]
	kubernetesVersions["1.28"] = kubernetesVersions["1.28.0"]
	kubernetesVersions["1.29"] = kubernetesVersions["1.29.0"]
	kubernetesVersions["1.30"] = kubernetesVersions["1.30.0"]
}

func GetKubernetesVersions(ver string) (KubernetesVersions, error) {
	ret, ok := kubernetesVersions[ver]
	if !ok {
		return ret, fmt.Errorf("No Kubernetes version available for %s", ver)
	}
	return ret, nil
}

func CompareKubernetesVersions(v1 string, v2 string) (int, error) {
	// Make sure these are valid versions
	_, ok := kubernetesVersions[v1]
	if !ok {
		return 0, fmt.Errorf("%s is not a supported Kubernetes version", v1)
	}

	_, ok = kubernetesVersions[v2]
	if !ok {
		return 0, fmt.Errorf("%s is not a supported Kubernetes version", v2)
	}

	ver1, err := semver.NewVersion(v1)
	if err != nil {
		return 0, err
	}

	ver2, err := semver.NewVersion(v2)
	if err != nil {
		return 0, err
	}

	return ver1.Compare(ver2), nil
}
