#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

setup_file() {
  export BATS_NO_PARALLELIZE_WITHIN_FILE=true
  
  VIRT_CLUSTER_NAME=myvirt
  BYO_CLUSTER_NAME=byocluster
  
  export BYO_CONFIG="test/bats/cipher-suite/byo_temp_config.yaml"
  export CAPI_CONFIG="test/bats/cipher-suite/capi_temp_config.yaml"
  export CAPI_MANIFEST= "test/bats/cipher-suite/capi_manifest.yaml"
  export BYO_WORKER_IGN="test/bats/cipher-suite/byo_worker_ignition.json"
  export BYO_CLUSTER_IGN="test/bats/cipher-suite/byo_cluster_ignition.json"
  export BYO_CONTROL_IGN="test/bats/cipher-suite/byo_control_plane_ignition.json"
  export KUBEADM_EXTRACTED="test/bats/cipher-suite/kubeadm_encoded.yaml"
  export CIPHER_SUITES="TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

  CLUSTER_CONFIG=test/bats/cipher-suite/config.yaml
  echo "cipherSuites: $CIPHER_SUITES" > $CLUSTER_CONFIG

  ocne cluster start --config $CLUSTER_CONFIG -C $VIRT_CLUSTER_NAME --auto-start-ui=false
  export KUBECONFIG="$HOME/.kube/kubeconfig.$VIRT_CLUSTER_NAME.local"
  export BYO_KUBECONFIG="$HOME/.kube/kubeconfig.$BYO_CLUSTER_NAME"
}

teardown_file() {
  ocne cluster delete -C $VIRT_CLUSTER_NAME
  ocne cluster delete -C $BYO_CLUSTER_NAME

  # Remove generated files
  rm $CLUSTER_CONFIG
  rm $BYO_CLUSTER_IGN
  rm $BYO_CONTROL_IGN
  rm $BYO_WORKER_IGN
  rm $KUBEADM_EXTRACTED
  rm $CAPI_MANIFEST
}

hasMatchingCipherSuites() {
  if [[ "$1" != "$CIPHER_SUITES" ]]; then
    echo "The default tls-cipher-suites were not found"
    return 1
  fi
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

@test "Check for cipher-suites on ocne cluster with capi" {
  # Extract the cipherSuites value from capi_temp_config.yaml
  cipherSuites=$(yq '.cipherSuites' $CAPI_CONFIG)

  # Generate the manifest
  ocne cluster template -c $CAPI_CONFIG > $CAPI_MANIFEST

  # Check output of each yq command separately against the cipherSuites value
  api_server_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.apiServer.extraArgs.tls-cipher-suites' $CAPI_MANIFEST)
  echo "$api_server_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in apiServer output."
    return 1
  fi

  etcd_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.etcd.local.extraArgs.cipher-suites' $CAPI_MANIFEST)
  echo "$etcd_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in etcd output."
    return 1
  fi

  controller_manager_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.controllerManager.extraArgs.tls-cipher-suites' $CAPI_MANIFEST)
  echo "$controller_manager_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in controllerManager output."
    return 1
  fi

  scheduler_output=$(yq '.spec.kubeadmConfigSpec.clusterConfiguration.scheduler.extraArgs.tls-cipher-suites' $CAPI_MANIFEST)
  echo "$scheduler_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in scheduler output."
    return 1
  fi

  init_node_registration_output=$(yq '.spec.kubeadmConfigSpec.initConfiguration.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $CAPI_MANIFEST)
  echo "$init_node_registration_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in initConfiguration nodeRegistration output."
    return 1
  fi

  join_node_registration_output=$(yq '.spec.kubeadmConfigSpec.joinConfiguration.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $CAPI_MANIFEST)
  echo "$join_node_registration_output" | grep "$cipherSuites"
  if [ $? -ne 0 ]; then
    echo "Test failed. '$cipherSuites' not found in joinConfiguration nodeRegistration output."
    return 1
  fi

  # If all checks passed
  echo "Test passed. '$cipherSuites' found in all relevant outputs."
}

@test "Check for cipher-suites on ocne cluster with byo" {
  rm -f $KUBECONFIG_BYO
  ocne cluster start -c $BYO_CONFIG > $BYO_CLUSTER_IGN

  cp $KUBECONFIG $BYO_KUBECONFIG

  ocne cluster join -c $BYO_CONFIG -k $BYO_KUBECONFIG -n 1 > $BYO_CONTROL_IGN
  ocne cluster join -c $BYO_CONFIG -k $BYO_KUBECONFIG -w 1 > $BYO_WORKER_IGN

  # Parse out the contents of kubeadm.conf generated by ignition
  CLUSTER_CIPHER_ENCODED=$(jq '.storage.files[3].contents.source' $BYO_CLUSTER_IGN | cut -d "," -f 2 | sed 's/.$//')
  CONTROL_CIPHER_ENCODED=$(jq '.storage.files[-1].contents.source' $BYO_CONTROL_IGN | cut -d "," -f 2 | sed 's/.$//')
  WORKER_CIPHER_ENCODED=$(jq '.storage.files[-1].contents.source' $BYO_WORKER_IGN | cut -d "," -f 2 | sed 's/.$//')

  echo $CLUSTER_CIPHER_ENCODED | base64 -d | gunzip > $KUBEADM_EXTRACTED

  # Decode contents extracted from kubeadm.conf
  CONTROL_CIPHER_DECODED=$(echo $CONTROL_CIPHER_ENCODED | base64 -d | gunzip | grep -o $CIPHER_SUITES)
  WORKER_CIPHER_DECODED=$(echo $WORKER_CIPHER_ENCODED | base64 -d | gunzip | grep -o $CIPHER_SUITES)
  KUBELET_CIPHER_DECODED=$(yq '.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $KUBEADM_EXTRACTED | grep $CIPHER_SUITES)
  API_SERVER_CIPHER_DECODED=$(yq '.apiServer.extraArgs.tls-cipher-suites' $KUBEADM_EXTRACTED | grep $CIPHER_SUITES)
  MANAGER_CIPHER_DECODED=$(yq '.controllerManager.extraArgs.tls-cipher-suites' $KUBEADM_EXTRACTED | grep $CIPHER_SUITES)
  SCHEDULER_CIPHER_DECODED=$(yq '.scheduler.extraArgs.tls-cipher-suites' $KUBEADM_EXTRACTED | grep $CIPHER_SUITES)
  ETCD_CIPHER_DECODED=$(yq '.etcd.local.extraArgs.cipher-suites' $KUBEADM_EXTRACTED | grep $CIPHER_SUITES)

  hasMatchingCipherSuites $CONTROL_CIPHER_DECODED
  hasMatchingCipherSuites $WORKER_CIPHER_DECODED
  hasMatchingCipherSuites $KUBELET_CIPHER_DECODED
  hasMatchingCipherSuites $API_SERVER_CIPHER_DECODED
  hasMatchingCipherSuites $MANAGER_CIPHER_DECODED
  hasMatchingCipherSuites $SCHEDULER_CIPHER_DECODED
  hasMatchingCipherSuites $ETCD_CIPHER_DECODED
}
