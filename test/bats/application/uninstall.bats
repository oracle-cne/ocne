#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Uninstalling an application that does not exist fails" {
    run ocne application uninstall --release foobar
    [ $status -ne 0 ]
}

@test "Uninstalling an application succeeds" {
    ocne application install --release kube-state-metrics --name kube-state-metrics --namespace kube-state-metrics
    ocne application uninstall --release kube-state-metrics --namespace kube-state-metrics
}

