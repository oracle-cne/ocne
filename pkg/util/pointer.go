// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"reflect"
)

func Int64Ptr(v int64) *int64 {
	return &v
}

func Int32Ptr(v int32) *int32 {
	return &v
}

func BoolPtr(v bool) *bool {
	return &v
}

func IntPtr(v int) *int {
	return &v
}

func StrPtr(v string) *string {
	return &v
}

// IsNil is a mostly complete way of checking if a
// value is nil.  It still misses some esoteric types
// of values.
//
// See: https://go.dev/doc/faq#nil_error
func IsNil(v interface{}) bool {
	return v == nil || reflect.ValueOf(v).IsNil()
}
