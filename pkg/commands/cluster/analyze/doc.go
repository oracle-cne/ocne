// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

// Package analyze contains the code to help triage problems. There are distinctions between
// symptoms, problems and causes.  A problem is that a pod won't start. The symptoms might include
// pod log error messages, Kubernetes events, pod resource condition messages, etc.  The cause
// might be resource starvation, low memory, permission issues, etc.  Sometimes there is a blurring
// between the problem and the symptom, such as "the pod won't start" might be considered to be both.
// However, in the context of this package, the symptoms are data points (like error messages) that
// indicate some problem might be happening or about to happen.
//
// The 'ocne cluster analyze' functionality has two phases: 1) collecting and report symptoms
// 2) analyzing the problem(s) based on those symptoms.
