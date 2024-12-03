#! /bin/bash

# Make sure all relevant environment variables are set
if [ -z "$OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION" ]; then
	echo 'The OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION environment variable must point to a cluster definition file'
	exit 1
fi

if [ ! -f "$OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION" ]; then
	echo "The OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION file, $OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION, does not exist"
	exit 1
fi

OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION=$(realpath "$OCI_CAPI_MANAGEMENT_CLUSTER_DEFINITION")

# Run the complete test suite
./run-tests.sh '' 1 1
