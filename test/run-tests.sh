#! /bin/bash
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
PATTERN=${1:-'.*'}

SUITE_JOBS=${SUITE_JOBS:-2}
TEST_JOBS=${TEST_JOBS:-2}
INCLUDE_CAPI=${2:-0}
TEST_FILTERS=${TEST_FILTERS:-'--filter-tags !CAPI,!CLUSTER_UPGRADE'}
TEST_K8S_VERSION=${TEST_K8S_VERSION:=""}

# Default to the latest k8s version
export K8S_VERSION_FLAG=""
if [[ $TEST_K8S_VERSION != "" ]]; then
  export K8S_VERSION_FLAG="--version ${TEST_K8S_VERSION}"
fi

# This silly thing makes mktemp work on Mac and Linux
TMP_HOME=$(mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir')
if [[ $? -ne 0 ]]; then
	echo Could not make temporary directory
	exit 1
fi

# Copy the ips.yaml so that existing clusters are not overwritten
# This might fail if the file doesn't exist, which is fine.
mkdir "$TMP_HOME/.ocne"
cp ~/.ocne/ips.yaml "$TMP_HOME/.ocne"
cp ~/.ocne/defaults.yaml "$TMP_HOME/.ocne/defaults.yaml"

# Link in the oci configuration
if [[ $INCLUDE_CAPI -eq 1 ]]; then
	ln -s ~/.oci $TMP_HOME/.oci
fi

# also symlink ssh keys
ln -s ~/.ssh $TMP_HOME/.ssh

# Link in the libvirt socket
mkdir -p $TMP_HOME/.cache/libvirt
ln -s ~/.cache/libvirt/libvirt-sock "$TMP_HOME/.cache/libvirt/libvirt-sock"

# Create .kube folder that is required for some tests
mkdir "$TMP_HOME/.kube"

export HOME="$TMP_HOME"


set -x
setups=$(find ./setups -type d -mindepth 1 | grep "$PATTERN")

for setup in $setups; do
	mkdir -p "$BATS_RESULT_DIR/$(basename $setup)"
done

# If parallel is installed, parallel execution becomes available.  Test fixtures
# should not stomp on each other as they will all have distinct cluster names.
#
# If parallel is not installed, fall back to a linear test run.
which parallel
if [[ $? -eq 0 ]] && [[ $INCLUDE_CAPI -ne 1 ]]; then
	parallel -j "$SUITE_JOBS" bats -j "$TEST_JOBS" --setup-suite-file {}/setup -o "$BATS_RESULT_DIR/{/.}" --report-formatter tap -r ./bats $(echo $TEST_FILTERS) ::: $(echo $setups)
elif [[ $INCLUDE_CAPI -ne 1 ]]; then
	for setup in $setups; do
		bats --setup-suite-file "$setup/setup" -o "$BATS_RESULT_DIR/$setup" -r ./bats $(echo $TEST_FILTERS)
	done
else
  for setup in $setups; do
  	bats --setup-suite-file "$setup/setup" -o "$BATS_RESULT_DIR/$setup" -r ./bats $(echo $TEST_FILTERS)
  done
fi

# Run any tests that require an ephemeral cluster.  These are always run in
# sequence because the ephemeral cluster is not very tolerant of being used
# by multiple processes at once.
bats -o "$BATS_RESULT_DIR/$setup" -r ./ephemeral-bats $(echo $TEST_FILTERS)

unlink "$TMP_HOME/.cache/libvirt/libvirt-sock"
if [[ $INCLUDE_CAPI -eq 1 ]]; then
	unlink $TMP_HOME/.oci
fi
rm -rf "$TMP_HOME"
