// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"os"
)

// FileIsTTY gives a best guess that a given file represents
// a TTY or PTY.  The current best guess is that the file is
// a character device.
//
// While it is possible for character devices not to be TTYs,
// it is unlikely to be true in the context of this code.  A
// common counterexample is something like a disk.  If someone
// is trying to do something like pipe a disk to stdin, then
// they are being silly and some odd behavior can be tolerated.
func FileIsTTY(f *os.File) (bool, error) {
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}

	isCharDevice := fi.Mode()&os.ModeCharDevice != 0
	return isCharDevice, nil
}
