#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Install and update the WebLogic Kubernetes Operator" {
    ocne catalog add --uri https://oracle.github.io/weblogic-kubernetes-operator --name "WebLogic Kubernetes Operator"

    # Override the image used to later verify that --reset-values works
    ocne application install --release weblogic-operator --namespace weblogic-operator --name weblogic-operator --version 4.1.2 --catalog "WebLogic Kubernetes Operator" --values - <<EOF
image: ghcr.io/oracle/weblogic-kubernetes-operator:4.1.2
EOF
    kubectl rollout status deployment -n weblogic-operator weblogic-operator -w
    run kubectl get pods -n weblogic-operator -o yaml | grep image:
    [ $status -eq 0 ]
    [[ "$output" =~ 4.1.2 ]]

    # Update to version 4.1.3 but use previous set of overrides.  The image tag should remain 1.4.2.
    ocne application update --release weblogic-operator --namespace weblogic-operator --version 4.1.3 --catalog "WebLogic Kubernetes Operator"
    kubectl rollout status deployment -n weblogic-operator weblogic-operator -w
    run kubectl get pods -n weblogic-operator -o yaml | grep image:
    [ $status -eq 0 ]
    [[ "$output" =~ 4.1.2 ]]

    # Update to version 4.1.4 and reset previous override values.
    ocne application update --release weblogic-operator --namespace weblogic-operator --version 4.1.4 --catalog "WebLogic Kubernetes Operator"
    kubectl rollout status deployment -n weblogic-operator weblogic-operator -w
    run kubectl get pods -n weblogic-operator -o yaml | grep image:
    [ $status -eq 0 ]
    [[ "$output" =~ 4.1.4 ]]

    # Update to version 4.2.7 and use previous override values (of which there should be none)
    ocne application update --release weblogic-operator --namespace weblogic-operator --version 4.2.7 --catalog "WebLogic Kubernetes Operator"
    kubectl rollout status deployment -n weblogic-operator weblogic-operator -w
    run kubectl get pods -n weblogic-operator -o yaml | grep image:
    [ $status -eq 0 ]
    [[ "$output" =~ 4.2.7 ]]
}