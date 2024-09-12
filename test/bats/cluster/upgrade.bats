#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=CLUSTER, CLUSTER_UPGRADE

setup_file() {
	# Turn off parallelization within this file.
	export BATS_NO_PARALLELIZE_WITHIN_FILE=true
}

k8s_version=$(kubectl get nodes  | awk '{ print $5}' | tail -n1 | cut -c1-5)
num_nodes=$(kubectl get nodes | grep Ready | wc -l)

verify() {
	count=$1
	mode=${2:-Ready}
	curcount=$(kubectl get nodes | grep $mode | wc -l)
	if [[ $((curcount)) != $((count)) ]]; then
		exit 1
	fi
}

wait() {
	count=$1
	mode=${2:-Ready}
	curcount=$(kubectl get nodes | grep $mode | wc -l)
	n=0
	while [ $n -le 20 ] && [[ $((curcount)) != $((count)) ]]; do
		curcount=$(kubectl get nodes | grep $mode | wc -l)
		n=$((n+1))
		sleep 10
	done
}

wait1() {
	count=$1
	curcount=$(ocne cluster info -s | grep Ready | grep true | wc -l)
	n=0
	while [ $n -le 20 ] && [[ $((curcount)) != $((count)) ]]; do
		curcount=$(ocne cluster info -s | grep Ready | grep true | wc -l)
		n=$((n+1))
		sleep 30
	done
}

# Wait for a node to be upgraded to the specified version
wait_node_upgrade_version() {
	node_name=$1
	k8s_version=$2
	while true; do
		wait_apiserver_accessible
		if kubectl get node $node_name | grep -w Ready | grep $k8s_version; then
			break
		fi
	done
}

# Ensure the API server is accessible, it can be unreachable when
# control-plane nodes are rebooting.
wait_apiserver_accessible() {
	# Wait for no error before proceeding to avoid test failure while control-plane coming up
	sleep 10
	while ! kubectl get nodes > /dev/null; do
		sleep 10
	done
}

check_images() {
	k8s_version=$1
	etcd_version=$2
	coredns_version=$3
	control_planes="kube-apiserver kube-controller-manager kube-scheduler"
	# initial sleep for the cluster to stabilize
	sleep 30
	for p in ${control_planes}; do
		count=$(kubectl describe po -A | grep "Image:" | grep $k8s_version | grep $p | wc -l)
		if [[ $((count)) == 0 ]]; then
			echo "Image $p is not version $k8s_version"
			exit 1
		fi
	done
	# it takes a bit for kube-proxy to be changed
	n=0
	count=$(kubectl describe po -A | grep Image: | grep $k8s_version | grep "kube-proxy" | wc -l)
	while [ $n -le 20 ] && [[ $((count)) != $((num_nodes)) ]]; do
		count=$(kubectl describe po -A | grep Image: | grep $k8s_version | grep "kube-proxy" | wc -l)
		n=$((n+1))
		echo "$n sleep 60"
		sleep 60
	done
	count=$(kubectl describe po -A | grep Image: | grep $k8s_version | grep "kube-proxy" | wc -l)
	if [[ $((count)) != $((num_nodes)) ]]; then
		echo "Image kube-proxy is not version $k8s_version"
		exit 1
	fi
	count=$(kubectl describe po -A | grep Image: | grep $etcd_version | grep "etcd" | wc -l)
	if [[ $((count)) == 0 ]]; then
		echo "Image etcd is not version ${etcd_version}"
		exit 1
	fi
	count=$(kubectl describe po -A | grep Image: | grep $coredns_version | grep "coredns" | wc -l)
	if [[ $((count)) == 0 ]]; then
		echo "Image coredns is not version ${coredns_version}"
		exit 1
	fi
}

execute_upgrade() {
	upgrade_to_version=$1
	etcd_version=$2
	coredns_version=$3
	wait "$num_nodes"
	ocne cluster stage -v$upgrade_to_version
	wait1 "$num_nodes"
	control_planes=$(ocne cluster info -s | grep true | grep "control plane" | awk '{print $1}')
	for n in ${control_planes}; do
		ocne node update --node ${n}
		wait_node_upgrade_version "${n}" "$upgrade_to_version"
	done
	if [[ $((num_nodes)) != 1 ]]; then
		worker_nodes=$(ocne cluster info -s | grep true | awk '{print $1}')
		for n in ${worker_nodes}; do
			ocne node update --node ${n} --delete-emptydir-data
			wait_node_upgrade_version "${n}" "$upgrade_to_version"
		done
	fi
	wait "$num_nodes" "$upgrade_to_version"
	check_images "$upgrade_to_version" "$etcd_version" "$coredns_version"
}

@test "Start with k8s version $k8s_version 1 node" {
	if [[ $((num_nodes)) != 1 ]]; then
		skip
	fi
	verify 1 ""
}

@test "Start k8s version $k8s_version with 2 cp 2 workers" {
	if [[ $((num_nodes)) != 4 ]]; then
		skip
	fi
	verify 4 "$k8s_version"
}

@test "Test k8s upgrade from version $k8s_version with $num_nodes nodes" {
	if [[ "$k8s_version" == v1.26 ]]; then
		execute_upgrade "1.27" "3.5.9" "1.10.1"
		k8s_version="v1.27"
	fi
	if [[ "$k8s_version" == v1.27 ]]; then
		execute_upgrade "1.28" "3.5.10" "1.10.1-1"
		k8s_version="v1.28"
	fi
	if [[ "$k8s_version" == v1.28 ]]; then
		execute_upgrade "1.29" "3.5.10" "1.11.1"
		k8s_version="v1.29"
	fi
	if [[ "$k8s_version" == v1.29 ]]; then
		execute_upgrade "1.30" "3.5.12" "1.11.1"
		k8s_version="v1.30"
	fi
}




