// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)


// This map ensures that the File object remains referenced.  If the File
// gets cleaned up, there is a hidden implicit close of the file that the
// runtime does.  The problem with that is the file created by MemfdCreate
// gets closed as well, which destroys the file and nobody else can use
// it.
var holds map[string]*os.File = map[string]*os.File{}

// InMemoryFile returns the filename of a file that exists only in memory.
// It can be used with all typical file operations except for deletion.
func InMemoryFile(name string) (string, error) {
	fd, err := unix.MemfdCreate(name, 0)
	if err != nil {
		return "", err
	}

	fnamePattern := "/proc/self/fd/%d"
	if runtime.GOOS == "darwin" {
		fnamePattern = "/dev/fd/%d"
	}
	fname := fmt.Sprintf(fnamePattern, fd)
	f := os.NewFile(uintptr(fd), name)
	holds[fname] = f
	return fname, nil
}
