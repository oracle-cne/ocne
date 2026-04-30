// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"os"

	"github.com/moby/term"
)

// FileIsTTY reports whether a given file is attached to a terminal.
// Use the same check as kubectl so both commands agree when stdin/stdout
// is a character device that is not actually a TTY.
func FileIsTTY(f *os.File) (bool, error) {
	_, isTTY := term.GetFdInfo(f)
	return isTTY, nil
}
