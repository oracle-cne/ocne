// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
)

const (
	justbytes = 1
	kibibytes = justbytes * 1024
	mebibytes = kibibytes * 1024
	gibibytes = mebibytes * 1024
	tebibytes = gibibytes * 1024
)

var suffixMap map[uint64]string = map[uint64]string{
	justbytes: "B",
	kibibytes: "KiB",
	mebibytes: "MiB",
	gibibytes: "GiB",
	tebibytes: "TiB",
}

func HumanReadableSize(size uint64) string {
	var base uint64
	switch {
	case size >= tebibytes:
		base = tebibytes
	case size >= gibibytes:
		base = gibibytes
	case size >= mebibytes:
		base = mebibytes
	case size >= kibibytes:
		base = kibibytes
	default:
		base = justbytes
	}

	return fmt.Sprintf("%.2f %s", float64(size) / float64(base), suffixMap[base])
}
