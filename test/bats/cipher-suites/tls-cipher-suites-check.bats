#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
# 

CIPHER_SUITES="cipherSuites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

setup_file() {
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
	unset KUBECONFIG
}

@test "Creating a cluster with libvirt as provider" {
	echo "$CIPHER_SUITES" > temp_config.yaml
	ocne cluster start --config temp_config.yaml
}	

@test "Check for default tls-cipher-suite" {
   ocne cluster console -d -N ocne-control-plane-1
   tls_cipher_suites=$(cat /var/lib/kubelet/kubeadm-flags.env | grep tls-cipher-suites)
   grep cipher /etc/kubernetes/manifests/*
}

@test "Deleting delete" {
	ocne cluster delete
	rm temp_config.yaml
}

