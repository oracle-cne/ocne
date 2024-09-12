#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=BACKUP

# Define the file to create the etcd backup in a temporary directory in setup() and delete it in tearDown()
setup() {
    tmp_dir=$(mktemp -d)
    backupFile=$tmp_dir/clusterbackup.db
}

teardown() {
    rm -rf $tmp_dir
}

@test "The etcd backup can be created" {
    run ocne cluster backup --out $backupFile
    [ "$status" -eq 0 ]

    # Simple assertion to assert the file specified for --out is created
    [[ -f "$backupFile" ]]
    [ "$status" -eq 0 ]
}

