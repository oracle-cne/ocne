#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=IMAGE

setup_file() {
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
	unset KUBECONFIG
}

@test "ocne image create --type ostree" {
	run -0 ocne image create --type ostree
	img=$(echo "$output" | grep "Saved image to" | awk '{print $NF}' | tr -d '"')
	if [ ! -f $img ]; then
		echo "missing image file $img"
		exit 1
	fi
	echo "image file created is $img"
	rm -rf $img /tmp/out-ostree
}

@test "ocne image create -a arm64 and amd64" {
	for arch in amd64 arm64; do
		run -0 ocne image create -a $arch
		img=$(echo "$output" | tail -n1 | awk '{print $NF}' | tr -d '"')
		if [ ! -f $img ]; then
			echo "missing image file $img"
			exit 1
		fi
		echo "image file created is $img"
		rm -rf $img /tmp/out-$arch
	done
}

