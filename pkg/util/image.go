// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"strings"

	"github.com/oracle-cne/ocne/pkg/constants"
)

func GetFullImage(registry, image string) string {
	if strings.TrimSpace(registry) == "" {
		return constants.ContainerRegistry + "/" + image
	}
	return registry + "/" + image
}
