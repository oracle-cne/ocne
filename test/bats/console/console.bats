#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CONSOLE

setup_file() {
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

@test "Console can execute commands on a node" {
	run kubectl get node -o=jsonpath='{.items[0].metadata.name}'
	[ "$status" -eq 0 ]
	NODE="$output"

	run bash -c "echo true | timeout 1m ocne cluster console --node $NODE"
	[ "$status" -eq 0 ]

	run bash -c "echo false | timeout 1m ocne cluster console --node $NODE"
	[ "$status" -ne 0 ]

	run timeout 1m ocne cluster console --node $NODE -- true
	[ "$status" -eq 0 ]

	run timeout 1m ocne cluster console --node $NODE -- false
	[ "$status" -ne 0 ]
}

@test "Console can execute commands on a node using --direct" {
	run kubectl get node -o=jsonpath='{.items[0].metadata.name}'
	[ "$status" -eq 0 ]
	NODE="$output"

	run bash -c "echo true | timeout 1m ocne cluster console --node $NODE --direct"
	[ "$status" -eq 0 ]

	run bash -c "echo false | timeout 1m ocne cluster console --node $NODE --direct"
	[ "$status" -ne 0 ]

	run timeout 1m ocne cluster console --node $NODE --direct -- true
	[ "$status" -eq 0 ]

	run timeout 1m ocne cluster console --node $NODE --direct -- false
	[ "$status" -ne 0 ]
}

@test "Console without --direct runs in the container" {
	run kubectl get node -o=jsonpath='{.items[0].metadata.name}'
	[ "$status" -eq 0 ]
	NODE="$output"

	run timeout 1m ocne cluster console --node $NODE --direct -- ls /
	[ "$status" -eq 0 ]
	echo "$output" | grep -qv ostree
}

@test "Console with --direct runs in the host" {
	run kubectl get node -o=jsonpath='{.items[0].metadata.name}'
	[ "$status" -eq 0 ]
	NODE="$output"

	run timeout 1m ocne cluster console --node $NODE --direct -- ls /
	[ "$status" -eq 0 ]
	echo "$output" | grep -q ostree
}
