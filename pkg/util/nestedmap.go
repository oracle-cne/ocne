package util

import "strings"

// EnsureNestedMap ensures that a nested map exists.  The map path is specified using dot notation.
// For example "a.b.c" will ensure the following exists.  The inner-most map is returned
//
//	map[string]interface{}{
//		"a": map[string]interface{}{
//		   "b": map[string]interface{}{
//		       "c": map[string]interface{}
func EnsureNestedMap(m map[string]interface{}, dotPath string) map[string]interface{} {
	segs := strings.Split(dotPath, ".")
	var inner map[string]interface{}

	for _, seg := range segs {
		entry := m[seg]
		if entry == nil {
			inner = make(map[string]interface{})
			m[seg] = inner
		} else {
			inner = entry.(map[string]interface{})
		}
	}
	return inner
}
