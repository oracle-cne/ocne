// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"github.com/Masterminds/semver/v3"
)

// CompareVersions compares two version strings that roughly conform
// to the Semantic Version specification.
func CompareVersions(v1 string, v2 string) (int, error) {
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
