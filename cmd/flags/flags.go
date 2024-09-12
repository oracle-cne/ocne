// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package flags

import (
	"errors"
	"slices"
	"strings"
)

const (
	FlagArchitecture      = "arch"
	FlagArchitectureShort = "a"
)

var ValidArchs = []string{"amd64", "arm64"}

// ValidateArchitecture validates that the passed in architecture is one of the valid values
func ValidateArchitecture(arch string) error {
	if !slices.Contains(ValidArchs, arch) {
		return errors.New("Architecture must be one of: " + strings.Join(ValidArchs, ", "))
	}
	return nil
}
