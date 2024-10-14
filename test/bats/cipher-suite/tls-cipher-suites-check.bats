#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

setup_file() {
    export BATS_NO_PARALLELIZE_WITHIN_FILE=true
    unset KUBECONFIG
}

@test “Creating a cluster with capi as provider” {
CIPHER_SUITES_CONFIG=“
name: oci
cipherSuites: TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
providers:
  oci:
    compartment: COMPARTMENT_ID
”
echo “$CIPHER_SUITES_CONGIG” > temp_config.yaml
ocne cluster template -c ~/.ocne/temp_config.yaml > capi_manifest.yaml

yq '.spec.kubeadmConfigSpec.clusterConfiguration.apiServer.extraArgs.tls-cipher-suites' capi_manifest.yaml
}
