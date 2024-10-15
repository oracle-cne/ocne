#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
# 

setup_file() {
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
	unset KUBECONFIG
}	

@test "Check for cipher-suites on ocne cluster with libvirt" {
	CLUSTER_NAME=myvirt
	CLUSTER_CONFIG=test/bats/cipher-suites/config.yaml
	ocne cluster start --config $CLUSTER_CONFIG -C $CLUSTER_NAME --auto-start-ui=false

	KUBECONFIG=$HOME/.kube/kubeconfig.$CLUSTER_NAME.local	
	kubectl get node -o=jsonpath='{.items[0].metadata.name}' --kubeconfig $KUBECONFIG
	NODE="$output"
	echo "$output"

	timeout 1m ocne cluster console --node $NODE --kubeconfig $KUBECONFIG --direct -- cat /var/lib/kubelet/kubeadm-flags.env | grep tls-cipher-suites
	KUBEADM_CIPHER_SUITES="$output"
	echo "$output"

	timeout 1m ocne cluster console --node $NODE --kubeconfig $KUBECONFIG --direct -- grep cipher /etc/kubernetes/manifests/*
	MANIFESTS_CIPHER_SUITES="$output"
	echo "$output"
	
	ocne cluster delete -C $CLUSTER_NAME
}

