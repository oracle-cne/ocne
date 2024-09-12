#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Showing a specific application that exists succeeds" {
	ocne application show --release ocne-catalog --namespace ocne-system
}

@test "Showing the difference from default for a specific application succeeds" {
	ocne application show --release ocne-catalog --namespace ocne-system --difference
}

@test "Showing the computed difference from default for a specific application succeeds" {
	ocne application show --release ocne-catalog --namespace ocne-system --computed
}

@test "Showing a specific application that does not exist fails" {
	run ocne application show --release foobar
	[ $status -ne 0 ]
}

