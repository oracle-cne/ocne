#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

setup_file() {
  export BATS_NO_PARALLELIZE_WITHIN_FILE=true
  VIRT_CLUSTER_NAME=myvirt
  CLUSTER_CONFIG=test/bats/cipher-suite/config.yaml
  export CIPHER_SUITES="TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
  echo "cipherSuites: $CIPHER_SUITES" > $CLUSTER_CONFIG

  ocne cluster start --config $CLUSTER_CONFIG -C $VIRT_CLUSTER_NAME --auto-start-ui=false
  export KUBECONFIG="$HOME/.kube/kubeconfig.$VIRT_CLUSTER_NAME.local"
}

teardown_file() {
  ocne cluster delete -C $VIRT_CLUSTER_NAME

  # Remove the temp cluster config file
  rm $CLUSTER_CONFIG
}

@test "Check for cipher-suites on ocne cluster with libvirt" {
	NODE=$(kubectl get node --kubeconfig $KUBECONFIG -o=jsonpath='{.items[0].metadata.name}')

  # Get the cipher-suites through an ocne cluster console for each component 
	KUBEADM_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /var/lib/kubelet/kubeadm-flags.env)
  ETCD_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/etcd.yaml | grep -o $CIPHER_SUITES)
  API_SERVER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-apiserver.yaml | grep -o $CIPHER_SUITES)
  CONTROLLER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-controller-manager.yaml | grep -o $CIPHER_SUITES)
  SCHEDULER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-scheduler.yaml | grep -o $CIPHER_SUITES)

  # kubeadm requires some trimming to extract the cipher-suites value
  KUBEADM_CIPHER_SUITES_TRIMMED=$(echo $KUBEADM_CIPHER_SUITES | sed 's/ --/\n/g' | grep $CIPHER_SUITES | cut -d "=" -f 2)
  hasMatchingCipherSuites $KUBEADM_CIPHER_SUITES_TRIMMED
  hasMatchingCipherSuites $ETCD_CIPHER_SUITES
  hasMatchingCipherSuites $API_SERVER_CIPHER_SUITES
  hasMatchingCipherSuites $CONTROLLER_CIPHER_SUITES
  hasMatchingCipherSuites $SCHEDULER_CIPHER_SUITES
}

hasMatchingCipherSuites() {
  if [[ "$1" != "$CIPHER_SUITES" ]]; then
    echo "The default tls-cipher-suites were not found"
    return 1
  fi
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