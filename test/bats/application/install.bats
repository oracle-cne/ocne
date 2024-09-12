#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Installing an application that does not exist fails" {
	run ocne application install --name foobar --release foobar
	[ $status -ne 0 ]
}

@test "Installing an application succeeds" {
    kubectl create namespace istio-system || true
	ocne application install --release grafana --name grafana

	ocne application show --release grafana
	run ocne application list
	[ $status -eq 0 ]
	[[ "$output" =~ grafana ]]

	run ocne application ls
	[ $status -eq 0 ]
	[[ "$output" =~ grafana ]]
}

@test "Installing an application from an ArtifactHub catalog succeeds" {
	ocne catalog add -u https://artifacthub.io -N "ACC-app-inst" -p artifacthub
	ocne application install --release ingress-nginx --name ingress-nginx --catalog "ACC-app-inst"
}

@test "Installing an application from the embedded catalog succeeds" {
    ocne application install --release metallb --name metallb --namespace metallb  --catalog embedded
}
