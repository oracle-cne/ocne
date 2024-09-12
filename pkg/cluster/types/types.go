// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

type NodeRole string

const (
	ControlPlaneRole NodeRole = "control-plane"
	WorkerRole       NodeRole = "worker"
)
