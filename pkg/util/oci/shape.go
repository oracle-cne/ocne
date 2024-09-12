// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"github.com/oracle-cne/ocne/pkg/constants"
)

func ArchitectureFromShape(shape string) string {
	for _, a := range constants.OciArmCompatibleShapes {
		if shape == a {
			return "arm64"
		}
	}
	return "amd64"
}
