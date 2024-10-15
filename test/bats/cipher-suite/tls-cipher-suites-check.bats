#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

setup_file() {
    export BATS_NO_PARALLELIZE_WITHIN_FILE=true
    unset KUBECONFIG
}

@test "Check for cipher-suites on ocne cluster with capi" {
  # Extract the cipherSuites value from capi_temp_config.yaml
  cipherSuites=$(yq '.cipherSuites' test/bats/cipher-suite/capi_temp_config.yaml)

  # Generate the manifest
  ocne cluster template -c test/bats/cipher-suite/capi_temp_config.yaml > test/bats/cipher-suite/capi_manifest.yaml

  # Check output of each yq command separately against the cipherSuites value
  api_server_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.apiServer.extraArgs.tls-cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$api_server_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in apiServer output."
    return 1
  fi

  etcd_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.etcd.local.extraArgs.cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$etcd_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in etcd output."
    return 1
  fi

  controller_manager_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.controllerManager.extraArgs.tls-cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$controller_manager_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in controllerManager output."
    return 1
  fi

  scheduler_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.scheduler.extraArgs.tls-cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$scheduler_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in scheduler output."
    return 1
  fi

  init_node_registration_output=$(yq '.spec.kubeadmConfigSpec.initConfiguration.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$init_node_registration_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in initConfiguration nodeRegistration output."
    return 1
  fi

  join_node_registration_output=$(yq '.spec.kubeadmConfigSpec.joinConfiguration.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' test/bats/cipher-suite/capi_manifest.yaml)
  echo "$join_node_registration_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in joinConfiguration nodeRegistration output."
    return 1
  fi

  #remove capi_manifest.yaml
  rm test/bats/cipher-suite/capi_manifest.yaml

  # If all checks passed
  echo "Test passed. '$cipherSuites' found in all relevant outputs."
}
