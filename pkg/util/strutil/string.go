// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package strutil

import "strings"

func TrimArray(a []string) []string {
	var out []string
	for _, s := range a {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}
