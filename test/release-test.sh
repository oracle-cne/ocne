#! /bin/bash

# Make sure all relevant environment variables are set
if [ -z "$CAPI_MANAGEMENT_CLUSTER_DEFINITION" ]; then
	echo 'The CAPI_MANAGEMENT_CLUSTER_DEFINITION environment variable must point to a cluster definition file'
	exit 1
fi

if [ ! -f "$CAPI_MANAGEMENT_CLUSTER_DEFINITION" ]; then
	echo "The CAPI_MANAGEMENT_CLUSTER_DEFINITION file, $CAPI_MANAGEMENT_CLUSTER_DEFINITION, does not exist"
	exit 1
fi

CAPI_MANAGEMENT_CLUSTER_DEFINITION=$(realpath "$CAPI_MANAGEMENT_CLUSTER_DEFINITION")

# Run the complete test suite
./run-tests.sh '' 1 1
