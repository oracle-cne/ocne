#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CIPHER

setup_file() {
  export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

hasMatchingCipherSuites() {
  if [[ "$1" != "$CIPHER_SUITES" ]]; then
    echo "The default tls-cipher-suites were not found"
    return 1
  fi
}

getEncodedKubeAdmConf() {
  jq -c '.[]' $1 |
    while read -r item; do
      if [[ $(jq -r ".path" <<<"$item") == "/etc/kubernetes/kubeadm.conf" ]]; then
        echo "$item" | jq '.contents.source' | cut -d "," -f 2 | sed 's/.$//'
        break
      fi
    done
}

@test "Check for cipher-suites on ocne cluster with libvirt" {
  if [[ $CIPHER_SUITES == "" ]]; then
    skip "CIPHER_SUITES was not set, skipping test"
  else
    NODE=$(kubectl get node --kubeconfig $KUBECONFIG -o=jsonpath='{.items[0].metadata.name}')
    # Get the cipher-suites through an ocne cluster console for each component
    KUBELET_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /var/lib/kubelet/kubeadm-flags.env)
    ETCD_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/etcd.yaml | grep -o $CIPHER_SUITES)
    API_SERVER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-apiserver.yaml | grep -o $CIPHER_SUITES)
    CONTROLLER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-controller-manager.yaml | grep -o $CIPHER_SUITES)
    SCHEDULER_CIPHER_SUITES=$(ocne cluster console --kubeconfig $KUBECONFIG --node $NODE --direct -- cat /etc/kubernetes/manifests/kube-scheduler.yaml | grep -o $CIPHER_SUITES)

    # kubeadm requires some trimming to extract the cipher-suites value
    KUBELET_CIPHER_SUITES_TRIMMED=$(echo $KUBELET_CIPHER_SUITES | sed 's/ --/\n/g' | grep $CIPHER_SUITES | cut -d "=" -f 2)
    hasMatchingCipherSuites $KUBELET_CIPHER_SUITES_TRIMMED
    hasMatchingCipherSuites $ETCD_CIPHER_SUITES
    hasMatchingCipherSuites $API_SERVER_CIPHER_SUITES
    hasMatchingCipherSuites $CONTROLLER_CIPHER_SUITES
    hasMatchingCipherSuites $SCHEDULER_CIPHER_SUITES
  fi
}

@test "Check for cipher-suites on ocne cluster with capi" {
  if [[ $CIPHER_SUITES == "" ]]; then
    skip "CIPHER_SUITES was not set, skipping test"
  else
    CAPI_CONFIG=bats/cipher-suite/capi_config.yaml
    CAPI_MANIFEST=bats/cipher-suite/capi_manifest.yaml

    # Extract the cipherSuites value from capi_config.yaml
    cipherSuites=$(yq '.cipherSuites' $CAPI_CONFIG)

    # Generate the manifest
    ocne cluster template -c $CAPI_CONFIG >$CAPI_MANIFEST

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

    # Cleanup 
    rm $CAPI_MANIFEST
  fi
}

@test "Check for cipher-suites on ocne cluster with byo" {
  if [[ $CIPHER_SUITES == "" ]]; then
    skip "CIPHER_SUITES was not set, skipping test"
  else
    BYO_CONFIG=bats/cipher-suite/byo_config.yaml
    # The files created in setup_file are either generated by the tests,
    # or used to hold temporary data for the test to pass
    BYO_WORKER_IGN=bats/cipher-suite/byo_worker_ignition.json
    BYO_CLUSTER_IGN=bats/cipher-suite/byo_cluster_ignition.json
    BYO_CONTROL_IGN=bats/cipher-suite/byo_control_plane_ignition.json
    BYO_CLUSTER_FILES=bats/cipher-suite/byo_cluster_files.json
    BYO_CONTROL_FILES=bats/cipher-suite/byo_control_files.json
    BYO_WORKER_FILES=bats/cipher-suite/byo_worker_files.json
    CLUSTER_EXTRACTED=bats/cipher-suite/cluster_encoded.yaml
    CONTROL_EXTRACTED=bats/cipher-suite/control_encoded.yaml
    WORKER_EXTRACTED=bats/cipher-suite/worker_encoded.yaml
    ocne cluster start --config "$BYO_CONFIG" >"$BYO_CLUSTER_IGN"
    BYO_KUBECONFIG=~/.kube/kubeconfig.byocluster

    cp $KUBECONFIG $BYO_KUBECONFIG
    ocne cluster join --config "$BYO_CONFIG" --kubeconfig $BYO_KUBECONFIG -n 1 >"$BYO_CONTROL_IGN"
    ocne cluster join --config "$BYO_CONFIG" --kubeconfig $BYO_KUBECONFIG -w 1 >"$BYO_WORKER_IGN"

    # Use jq to get the list of objects generated by ignition
    jq '.storage.files' $BYO_CLUSTER_IGN >$BYO_CLUSTER_FILES
    jq '.storage.files' $BYO_CONTROL_IGN >$BYO_CONTROL_FILES
    jq '.storage.files' $BYO_WORKER_IGN >$BYO_WORKER_FILES

    # Get the encoded contents of the object with path /etc/kubernetes/kubeadm.conf
    CLUSTER_CIPHER_ENCODED=$(getEncodedKubeAdmConf $BYO_CLUSTER_FILES)
    CONTROL_CIPHER_ENCODED=$(getEncodedKubeAdmConf $BYO_CONTROL_FILES)
    WORKER_CIPHER_ENCODED=$(getEncodedKubeAdmConf $BYO_WORKER_FILES)

    # Decode contents and write out to be read later
    echo $CONTROL_CIPHER_ENCODED | base64 -d | gunzip >$CONTROL_EXTRACTED
    echo $WORKER_CIPHER_ENCODED | base64 -d | gunzip >$WORKER_EXTRACTED
    echo $CLUSTER_CIPHER_ENCODED | base64 -d | gunzip >$CLUSTER_EXTRACTED

    CONTROL_CIPHER_DECODED=$(yq '.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $CONTROL_EXTRACTED)
    WORKER_CIPHER_DECODED=$(yq '.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $WORKER_EXTRACTED)
    KUBELET_CIPHER_DECODED=$(yq '.nodeRegistration.kubeletExtraArgs.tls-cipher-suites' $CLUSTER_EXTRACTED | grep -o $CIPHER_SUITES)
    API_SERVER_CIPHER_DECODED=$(yq '.apiServer.extraArgs.tls-cipher-suites' $CLUSTER_EXTRACTED | grep -o $CIPHER_SUITES)
    MANAGER_CIPHER_DECODED=$(yq '.controllerManager.extraArgs.tls-cipher-suites' $CLUSTER_EXTRACTED | grep -o $CIPHER_SUITES)
    SCHEDULER_CIPHER_DECODED=$(yq '.scheduler.extraArgs.tls-cipher-suites' $CLUSTER_EXTRACTED | grep -o $CIPHER_SUITES)
    ETCD_CIPHER_DECODED=$(yq '.etcd.local.extraArgs.cipher-suites' $CLUSTER_EXTRACTED | grep -o $CIPHER_SUITES)

    hasMatchingCipherSuites $CONTROL_CIPHER_DECODED
    hasMatchingCipherSuites $WORKER_CIPHER_DECODED
    hasMatchingCipherSuites $KUBELET_CIPHER_DECODED
    hasMatchingCipherSuites $API_SERVER_CIPHER_DECODED
    hasMatchingCipherSuites $MANAGER_CIPHER_DECODED
    hasMatchingCipherSuites $SCHEDULER_CIPHER_DECODED
    hasMatchingCipherSuites $ETCD_CIPHER_DECODED

    # Cleanup
    ocne cluster delete --cluster-name=byocluster
    rm $BYO_WORKER_IGN
    rm $BYO_CLUSTER_IGN
    rm $BYO_CONTROL_IGN
    rm $BYO_CLUSTER_FILES
    rm $BYO_CONTROL_FILES
    rm $BYO_WORKER_FILES
    rm $CLUSTER_EXTRACTED
    rm $CONTROL_EXTRACTED
    rm $WORKER_EXTRACTED
  fi
}
