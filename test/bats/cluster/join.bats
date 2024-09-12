#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CLUSTER, CLUSTER_JOIN

# Used to make generated files unique across test setups
SUFFIX=$RANDOM

setup() {
    cat > $HOME/.ocne/byo-$SUFFIX.yaml - <<EOF
provider: byo
name: ocnejoin-$SUFFIX
loadBalancer: 1.2.3.4
providers:
  byo:
    networkInterface: ens3
EOF

    cp $KUBECONFIG $HOME/.kube/kubeconfig.ocnejoin-$SUFFIX
}

@test "Cluster Join BYO (self-provisioned)" {
    run ocne cluster join --config $HOME/.ocne/byo-$SUFFIX.yaml --kubeconfig $KUBECONFIG -n 1 > $HOME/.ocne/joinbyo-$SUFFIX.ign
    [ "$status" -eq 0 ]
    echo $output | grep '\"ignition\":'
}
