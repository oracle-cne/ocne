// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import "time"

const (
	FlagTimeout      = "timeout"
	FlagTimeoutShort = "t"
	FlagTimeoutHelp  = "time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)"

	FlagWait      = "wait"
	FlagWaitShort = "w"
	FlagWaitHelp  = "if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a " +
		"ready state before marking the release as successful. It will wait for as long as --timeout"

	FlagWaitForJobs      = "wait-for-jobs"
	FlagWaitForJobsShort = "j"
	FlagWaitForJobsHelp  = "if set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout"

	DefaultTimeout time.Duration = 300 * time.Second
)
