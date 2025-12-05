// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
type JsonPatches struct {
	Patches []*JsonPatch
}

func (jp *JsonPatches) AddPatch(op string, path []string, value interface{}) *JsonPatches {
	jp.Patches = append(jp.Patches, &JsonPatch{
		Op:    op,
		Path:  fmt.Sprintf("/%s", strings.Join(path, "/")),
		Value: value,
	})
	return jp
}

func (jp *JsonPatches) Replace(path []string, value interface{}) *JsonPatches {
	return jp.AddPatch("replace", path, value)
}

func (jp *JsonPatches) Add(path []string, value interface{}) *JsonPatches {
	return jp.AddPatch("add", path, value)
}

func (jp *JsonPatches) String() string {
	ret, _ := json.Marshal(jp.Patches)
	//escaped := strings.ReplaceAll(string(ret), `\`, `\\`)
	//escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	escaped := strings.ReplaceAll(string(ret), `'`, `'\''`)
	return escaped
}

func (jp *JsonPatches) Merge(toMerge *JsonPatches) *JsonPatches {
	for _, p := range toMerge.Patches {
		jp.Patches = append(jp.Patches, &JsonPatch{
			Op:    p.Op,
			Path:  p.Path,
			Value: p.Value,
		})
	}

	return jp
}
