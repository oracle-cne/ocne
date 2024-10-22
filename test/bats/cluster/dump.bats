#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CLUSTER, CLUSTER_DUMP

setup_file() {
	# Turn off parallelization within this file.  Running two or more
	# dumps against the same cluster is not supported.
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

setup() {
	rm -rf /tmp/dump*
}

teardown() {
	rm -rf /tmp/dump*
}

base_dump_dir=/tmp/dump

@test "ocne cluster dump -d /tmp/dump -N node with redaction" {
	dump_dir=$base_dump_dir-$RANDOM
	target_node=$(kubectl --kubeconfig $KUBECONFIG get nodes --no-headers | grep Ready | awk '{print $1}' | head -1)
	ocne cluster dump -d $dump_dir -N $target_node
	if [ ! -d $dump_dir/nodes/REDACTED* ]; then
		exit 1
	fi
}

@test "ocne cluster dump --output-directory /tmp/dump" {
	dump_dir=$base_dump_dir-$RANDOM
	ocne cluster dump --output-directory $dump_dir -t
	for node in $(kubectl --kubeconfig $KUBECONFIG get nodes | grep Ready | awk '{print $1}'); do
		if [ ! -d $dump_dir/nodes/$node ]; then
			exit 1
		fi
	done
}

@test "ocne cluster dump -d /tmp/dump -N node" {
	dump_dir=$base_dump_dir-$RANDOM
	target_node=$(kubectl --kubeconfig $KUBECONFIG get nodes | grep Ready | awk '{print $1}' | tail -n1)
	ocne cluster dump --output-directory $dump_dir -N $target_node -t
	if [ ! -d $dump_dir/nodes/$target_node ]; then
		exit 1
	fi
}

@test "ocne cluster dump -d /tmp/dump --skip-nodes" {
	dump_dir=$base_dump_dir-$RANDOM
	ocne cluster dump -d $dump_dir --skip-nodes
	if [ -d $dump_dir/nodes ]; then
		exit 1
	fi
}

@test "ocne cluster dump -d /tmp/dump --skip-cluster" {
	dump_dir=$base_dump_dir-$RANDOM
	ocne cluster dump -d $dump_dir --skip-cluster
	if [ -d $dump_dir/cluster ]; then
		exit 1
	fi
}

@test "ocne cluster dump -d /tmp/dump --skip-pod-logs" {
	dump_dir=$base_dump_dir-$RANDOM
	ocne cluster dump -d $dump_dir --skip-pod-logs
	if [ -d $dump_dir/cluster/namespaces/*/podlogs ]; then
		exit 1
	fi
}

@test "ocne cluster dump -d /tmp/dump -n myns" {
	dump_dir=$base_dump_dir-$RANDOM
	# create myns namespace
	kubectl --kubeconfig $KUBECONFIG create ns myns
	ocne cluster dump -d $dump_dir -n myns
	if [ ! -d $dump_dir/cluster/namespaces/myns ]; then
		exit 1
	fi
}

@test "ocne cluster dump -z /tmp/dump.tgz -n "default" --skip-nodes" {
	dump_dir=$base_dump_dir-$RANDOM
	ocne cluster dump -z $dump_dir.tgz -n "default" --skip-nodes
	if [ ! -f $dump_dir.tgz ]; then
		exit 1
	fi
}
