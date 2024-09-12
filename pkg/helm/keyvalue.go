// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

// KeyValue defines the Key, Value pair used to override a single helm Value
type KeyValue struct {
	Key       string
	Value     string
	SetString bool // for --set-string
	SetFile   bool // for --set-file
	IsFile    bool // for -f
}
