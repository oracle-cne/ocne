// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package features

// OLVM indicates if developer features are enabled or disabled.  This is determined at compile time via the Makefile
// with go build conditional compile flag named "developer"
const OLVM = enabled
