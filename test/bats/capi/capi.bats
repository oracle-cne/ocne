#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
# bats file_tags=CAPI
setup_file() {
   export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}
@test "Creating a CAPI cluster succeeds" {
	ocne cluster start --provider oci -C "$CAPI_NAME" --auto-start-ui=false
}

@test "Deleting a CAPI cluster succeeds" {
	waitForCAPIDeletion
}


waitForCAPIDeletion() {
        ocne cluster delete -C "$CAPI_NAME" --log-level info
        n=0
        while [ $n -le 60 ]; do
           x=$(kubectl get clusters -A)
           if ! [[ $x =~ "$CAPI_NAME" ]]; then
              return 0
           fi
           n=$((n+1))
           sleep 10
        done
        return 1
}
