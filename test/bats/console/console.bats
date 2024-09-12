#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CONSOLE

@test "Console can execute commands on a node" {
	run kubectl get node -o=jsonpath='{.items[0].metadata.name}'
	[ "$status" -eq 0 ]
	NODE="$output"

	run bash -c "echo true | timeout 1m ocne cluster console --node $NODE"
	[ "$status" -eq 0 ]

	run bash -c "echo false | timeout 1m ocne cluster console --node $NODE"
	[ "$status" -ne 0 ]
}
