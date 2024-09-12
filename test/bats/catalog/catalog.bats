#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CATALOG

@test "Listing catalogs gives output" {
	run ocne catalog list
	[ $status -eq 0 ]
	[ $(echo "$output" | wc -l) -gt 1 ]

	run ocne catalog ls
	[ $status -eq 0 ]
	[ $(echo "$output" | wc -l) -gt 1 ]
}

@test "Catalogs can be added" {
	ocne catalog add -u https://artifacthub.io -N "ArtifactHub Community Catalog" -p artifacthub
}

@test "ArtifactHub catalogs can be searched" {
	ocne catalog add -u https://artifacthub.io -N "ACC-search" -p artifacthub
	ocne catalog search --name "ACC-search" --pattern ingress-nginx
}

@test "Embedded catalog can be searched" {
    run ocne catalog search --name embedded
    [ $status -eq 0 ]
    echo $output | grep -e 'flannel' -e 'cert-manager'
}

@test "Catalogs can be removed" {
	ocne catalog add -u https://artifacthub.io -N "ACC-remove" -p artifacthub
	ocne catalog remove -N "ACC-remove"
	run ocne catalog list
	[ "$status" -eq 0 ]
	[[ ! "$output" =~ "ACC-remove" ]]
}

@test "Getting a catalog produces output" {
	run ocne catalog get
	[ "$status" -eq 0 ]
	[ -n "$output" ]
}
