// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package versions

import (
	"fmt"
	"strings"

	"github.com/oracle-cne/ocne/pkg/util"

	"github.com/Masterminds/semver/v3"
)

type KubernetesVersions struct {
	Kubernetes string
	Pause      string
	Etcd       string
	CoreDNS    string
}

var kubernetesVersions = map[string]KubernetesVersions{
	"1.26.6": {
		Kubernetes: "1.26.6",
		Pause:      "3.9",
		Etcd:       "3.5.6",
		CoreDNS:    "v1.9.3-4",
	},
	"1.27.12": {
		Kubernetes: "1.27.12",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "v1.10.1",
	},
	"1.28.8": {
		Kubernetes: "1.28.8",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "v1.10.1-1",
	},
	"1.29.3": {
		Kubernetes: "1.29.3",
		Pause:      "3.9",
		Etcd:       "3.5.10",
		CoreDNS:    "v1.11.1",
	},
	"1.30.3": {
		Kubernetes: "1.30.3",
		Pause:      "3.9",
		Etcd:       "3.5.12",
		CoreDNS:    "v1.11.1",
	},
	"1.31.0": {
		Kubernetes: "1.31.0",
		Pause: "3.10",
		Etcd: "3.5.15",
		CoreDNS: "current",
	},
	"1.32.0": {
		Kubernetes: "1.32.0",
		Pause: "3.10",
		Etcd: "3.5.15",
		CoreDNS: "current",
	},
}

func init() {
	// Add keys to map from the major.minor versions to major.minor.patch versions
	kubernetesVersions["1.26"] = kubernetesVersions["1.26.6"]
	kubernetesVersions["1.27"] = kubernetesVersions["1.27.12"]
	kubernetesVersions["1.28"] = kubernetesVersions["1.28.8"]
	kubernetesVersions["1.29"] = kubernetesVersions["1.29.3"]
	kubernetesVersions["1.30"] = kubernetesVersions["1.30.3"]
	kubernetesVersions["1.31"] = kubernetesVersions["1.31.0"]
	kubernetesVersions["1.32"] = kubernetesVersions["1.32.0"]
}

func GetKubernetesVersions(ver string) (KubernetesVersions, error) {
	ret, ok := kubernetesVersions[ver]
	if !ok {
		return ret, fmt.Errorf("No Kubernetes version available for %s", ver)
	}
	return ret, nil
}

// IsSupportedKubernetesVersion takes a semantic version and checks
// if it is a supported version or not.  Unlike a direct lookup,
// this check normalizes the full version string to major.minor.patch.
func IsSupportedKubernetesVersion(v string) (bool, error) {
	// First do a direct lookup into the map.  If the version is
	// there then it is supported.  If it's not, then coerce the
	// version into a major.minor.patch semantic version and try again.
	v = strings.TrimPrefix(v, "v")
	_, ok := kubernetesVersions[v]
	if ok {
		return true, nil
	}

	sv, err := semver.NewVersion(v)
	if err != nil {
		return false, err
	}
	kv := semver.New(sv.Major(), sv.Minor(), sv.Patch(), "", "")
	_, ok = kubernetesVersions[kv.String()]
	return ok, nil
}

// EnsureSupportedKubernetesVersion returns an error if the input
// version is not a supported Kubernetes version.
func EnsureSupportedKubernetesVersion(v string) error {
	ok, err := IsSupportedKubernetesVersion(v)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s is not a supported Kubernetes version", v)
	}
	return nil
}

// CompareKubernetesVersions compares two supported Kubernetes versions.
func CompareKubernetesVersions(v1 string, v2 string) (int, error) {
	// Make sure these are valid versions
	err := EnsureSupportedKubernetesVersion(v1)
	if err != nil {
		return 0, err
	}

	err = EnsureSupportedKubernetesVersion(v2)
	if err != nil {
		return 0, err
	}

	return util.CompareVersions(v1, v2)
}
