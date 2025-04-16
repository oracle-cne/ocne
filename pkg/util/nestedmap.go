// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import "strings"

// EnsureNestedMap ensures that a nested map exists.  The map path is specified using dot notation.
// For example "a.b.c" will ensure the following exists.  The inner-most map is returned
//
//	map[string]interface{}{
//		"a": map[string]interface{}{
//		   "b": map[string]interface{}{
//		       "c": map[string]interface{}
func EnsureNestedMap(mapRoot map[string]interface{}, dotPath string) map[string]interface{} {
	segs := strings.Split(dotPath, ".")
	var inner map[string]interface{}

	m := mapRoot
	for _, seg := range segs {
		entry := m[seg]
		if entry == nil {
			inner = make(map[string]interface{})
			m[seg] = inner
		} else {
			inner = entry.(map[string]interface{})
		}
		m = inner
	}
	return inner
}
