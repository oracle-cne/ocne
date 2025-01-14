// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"strconv"
	"strings"
)


// IncrementCount adds one to a string that has a number at the end
// or appends a 1 to a string that does not.  Empty strings are
// returned unmodified.
func IncrementCount(in string, delim string) string {
	fields := strings.Split(in, delim)
	if len(fields) == 0 {
		return ""
	}
	idx := len(fields) - 1

	count, err := strconv.ParseUint(fields[idx], 10, 64)
	if err != nil {
		fields = append(fields, "1")
	} else {
		fields[idx] = strconv.FormatUint(count+1, 10)
	}
	return strings.Join(fields, delim)
}
