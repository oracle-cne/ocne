// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package util

import (
	"encoding/json"
	"strings"
)

type JsonPatch struct {
	Op string `json:"op"`
	Path string `json:"path"`
	Value string `json:"value"`
}
type JsonPatches struct {
	Patches []*JsonPatch
}

func (jp *JsonPatches) AddPatch(op string, path []string, value string) *JsonPatches {
	jp.Patches = append(jp.Patches, &JsonPatch{
		Op: op,
		Path: strings.Join(path, "/"),
		Value: value,
	})
	return jp
}

func (jp *JsonPatches) Replace(path []string, value string) *JsonPatches {
	return jp.AddPatch("replace", path, value)
}

func (jp *JsonPatches) String() string {
	ret, _ := json.Marshal(jp.Patches)
	return string(ret)
}
