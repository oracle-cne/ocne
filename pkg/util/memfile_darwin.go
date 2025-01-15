// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
//go:build darwin

package util

import (
	"os"

	"github.com/oracle-cne/ocne/pkg/util/garbage"
)


// InMemoryFile returns the filename of a file that registers its
// own deletion with the garbage collector.  It is automatically
// deleted when the collector cleanup is called.  This has a unique
// implementation for Mac because the Linux implementation uses
// MemfdCreate for a safer experience.
func InMemoryFile(name string) (string, error) {
	f, err := os.CreateTemp("", "kcfg")
	if err != nil {
		return "", err
	}
	garbage.Add(func(a interface{}){
		f, _ := a.(*os.File)
		f.Close()
		os.Remove(f.Name())
	}, f)
	return f.Name(), nil
}

