#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=IMAGE

setup_file() {
	# Turn off parallelization within this file.  Running two or more
	# image creates against the same cluster is not supported.
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

@test "ocne image create --type ostree" {
	ocne image create --type ostree > /tmp/out-ostree 2>&1
	img=$(grep "Saved image to" /tmp/out-ostree | awk '{print $NF}' | tr -d '"')
	if [ ! -f $img ]; then
		echo "missing image file $img"
		exit 1
	fi
	echo "image file created is $img"
	rm -rf $img /tmp/out-ostree
}

@test "ocne image create -a arm64 and amd64" {
	for arch in amd64 arm64; do
		ocne image create -a $arch > /tmp/out-$arch 2>&1
		img=$(tail -n1 /tmp/out-$arch | awk '{print $NF}' | tr -d '"')
		if [ ! -f $img ]; then
			echo "missing image file $img"
			exit 1
		fi
		echo "image file created is $img"
		rm -rf $img /tmp/out-$arch
	done
}
