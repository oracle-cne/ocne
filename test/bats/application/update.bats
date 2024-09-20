#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Install and update the WebLogic Kubernetes Operator" {
    ocne catalog add --uri https://oracle.github.io/weblogic-kubernetes-operator --name "WebLogic Kubernetes Operator"

    # Override the image used to later verify that --reset-values works
    ocne application install --release weblogic-operator --namespace weblogic-operator --name weblogic-operator --version 4.2.7 --catalog "WebLogic Kubernetes Operator" --values - <<EOF
image: ghcr.io/oracle/weblogic-kubernetes-operator:4.1.2
EOF
    kubectl rollout status deployment -n weblogic-operator weblogic-operator -w
    run kubectl get pods -n weblogic-operator -o yaml | grep image:
    [ $status -ne 0 ]
    [[ "$output" =~ 1.4.2 ]]
}