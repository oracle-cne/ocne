// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package garbage

type GarbageCleanupFunc func(interface{})

type garbageEntry struct {
	cb GarbageCleanupFunc
	arg interface{}
}

var garbage []*garbageEntry

func Add(cb GarbageCleanupFunc, arg interface{}) {
	garbage = append(garbage, &garbageEntry{
		cb: cb,
		arg: arg,
	})
}

func Cleanup() {
	for _, g := range garbage {
		g.cb(g.arg)
	}
}
