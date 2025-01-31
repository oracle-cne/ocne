#!/bin/bash -x
#
# Copyright (c) 2019-2021 Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.

# Allow for this script to be ran from anywhere
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REGISTRY="container-registry.oracle.com"
NGINX_IMAGE="nginx:1.17.7-1"
RUN_SNO=0
UNSET_PROXY=0         # If enabled, proxy can block curl requests.  Unset if necessary, i.e. set to '1'.

if [[ ${UNSET_PROXY} -eq 1 ]] ; then
        unset http_proxy https_proxy no_proxy
fi

TLS13="${TLS_MIN_VERSION:-}"
if [[ ! -z $(ps -fe|grep kube-apiserver|grep VersionTLS13) ]]; then
        TLS13="--tlsv1.3"
fi

if [[ ! -z ${TLS13} ]]; then
	if [[ "${TLS13}" != "--tlsv1.3" ]]; then
		TLS13=""
	fi
fi

CURL=(curl '"$TLS13"' '*' --fail --max-time 10 --silent --output /dev/null --write-out '%{http_code}\\n')

if [ -z "${KUBECONFIG}" ]; then
    # set default kubeconfig location
    export KUBECONFIG=/etc/kubernetes/admin.conf
fi

# Checks if the function has been declared
function isFuncDeclared() {
    declare -Ff "$1" >/dev/null 2>&1;
    echo $?;
};

# Calls log function if delcared, otherwise calls echo.
function logInfo() {
    if [ $(isFuncDeclared log) -eq 0 ]; then
        log $1;
    else
        echo $1;
    fi;
};

function get_apiserver_address {
    # use a dumb terminal to disable the color output. https://github.com/daviddengcn/go-colortext/blob/186a3d44e9200d7eb331356ca4864f52708e1399/ct_ansi.go#L11-L13
    TERM=dumb kubectl cluster-info | grep -Eo 'https?://[^ ]+' | head -1
};

: '
    Check whether a package is installed or  not
'
function isinstalled() {
    os_name=$(uname -s)
    if [[ "$os_name" == "Darwin" ]]; then
        if which "$@" >/dev/null 2>&1; then
            true
        else
            false
        fi
    else
        if yum list installed "$@" >/dev/null 2>&1; then
            true
        else
            false
        fi
    fi
};


: '
    Get the installed version of a package
    parameter:
        input: package name
        output: version format x.x
'
function get_package_version()
{
    version=$(rpm -q $1 | awk -F'-' '{print $2}' | awk -F'.' '{b=$1"."$2;print b}')
    logInfo $version
};


function do_exit() {
    logInfo "$1"
    exit $2
};

: '
    check the exit code and exit with proper message if code is not zero
'
function check_exit_code() {
    if [ $1 -ne 0 ]; then
        do_exit "$2 failed" 1
    fi
};


function validate_system {
    logInfo "Validating system"
    package=kubectl
    if ! isinstalled $package; then
        do_exit "${package} not installed. Exiting the script" 1
    fi
    current_version=$(get_package_version ${package})
    logInfo "$package version $current_version"

    # Validate kubernetes is running
    kubectl get no

    # Validate access to api server
    ip=$(get_apiserver_address)
    "${CURL[@]}" --insecure "$ip/healthz"
    check_exit_code $? "Checking access to cluster api server"

    logInfo "Successfully validated system"
}

function validate_golang() {
    # Validate golang is installed
    package=golang
    if ! isinstalled $package; then
        do_exit "${package} not installed. Exiting the script" 1
    fi
    current_version=$(get_package_version ${package})
    logInfo "$package version $current_version"
};

function createTestResources() {
    if [ -f ${SOURCE_DIR}/nginx.yaml ]; then
        # This section is used by Jenkins Pipeline e2e
        logInfo "File exists: ${SOURCE_DIR}/nginx.yaml"
        kubectl apply -f ${SOURCE_DIR}/nginx.yaml
        kubectl get nodes,pod,svc,namespaces -A -o wide
    else
        kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: testdns
spec:
  selector:
    matchLabels:
      name: testdns
  template:
    metadata:
      labels:
        name: testdns
    spec:
      containers:
        - name: testdns
          image: ${REGISTRY}/os/oraclelinux:8
          command: ["sh", "-c", "sleep 1000"]
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      name: nginx
  template:
    metadata:
      labels:
        name: nginx
    spec:
      containers:
      - name: nginx
        image: ${REGISTRY}/olcne/${NGINX_IMAGE}
        ports:
        - containerPort: 80
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  type: NodePort
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  selector:
    name: nginx
EOF
    fi

    waitForPod "condition=Ready" "name=testdns"
    waitForPod "condition=Ready" "name=nginx"

    # Show test pods
    kubectl get po -o wide
}

# Adds a retry around waitng for the pods since sometimes the api server resource doesn't exist
function waitForPod {
    action=$1
    labelSelector=$2

    for _ in $(seq 5); do
        kubectl wait po --for="${action}" -l "${labelSelector}" --timeout=5m && break

        sleep 30
    done
}

function testResourceCleanup() {
    kubectl delete ds testdns
    waitForPod "delete" "name=testdns"

    kubectl delete svc nginx
    kubectl delete deploy nginx
    waitForPod "delete" "name=nginx"
}

function curl_test {
    max_attempts=10
    node="$1"
    url="$2"
    attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        logInfo "Attempt ${attempt} Testing  Curl ${url}"
        ocne cluster console --node "$node" -- "${CURL[@]}" "${url}"
        exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            logInfo $exit_code
            break
        fi
        logInfo "curl ${url} failed with exit code $exit_code"
        logInfo "Retrying in 5 seconds..."
        sleep 5
        attempt=$((attempt + 1))
    done
    if [[ $attempt -gt $max_attempts ]]; then
        logInfo "Maximum attempts reached. Failed to curl ${url} successfully."
    fi
    logInfo -1
}

function run_sniff_tests {
    createTestResources

    # Test accessing nodeport on every host
    nodePort=$(kubectl get svc nginx -o jsonpath='{.spec.ports[0].nodePort}')
    logInfo "Node Port: ${nodePort}"

    srcNode=$(kubectl get no --selector='!node-role.kubernetes.io/control-plane' --no-headers -o wide | head -1 | awk '{ print $1 }')

    for node in $(kubectl get no --selector='!node-role.kubernetes.io/control-plane' --no-headers -o wide | awk '{ print $6 }'); do
        logInfo "Testing Node Port Curl ${node}:${nodePort}"
        # "${CURL[@]}" "${node}:${nodePort}"
	curl_test "$srcNode" "${node}:${nodePort}"
        check_exit_code $? "Checking access to cluster nodeport"
    done

    kubernetes_cluster_ip=$(kubectl get svc kubernetes -o jsonpath='{.spec.clusterIP}')

    test_dns_pos=$(kubectl get po | grep testdns | awk '{ print $1 }')
    for dns_po in $test_dns_pos; do
        logInfo "Test accessing cluster dns in a pod on every host"
        kubectl exec -i ${dns_po} -- "${CURL[@]}" nginx.default.svc.cluster.local
        check_exit_code $? "testing cluster dns $dns_po"

        logInfo "Test accessing kubernetes api cluster ip in a pod on every host"
        kubectl exec -i ${dns_po} -- "${CURL[@]}" --insecure "https://${kubernetes_cluster_ip}:443/healthz"
        check_exit_code $? "testing kubernetes api cluster ip $dns_po"
    done

    testResourceCleanup
};

function run_conformance_tests {
    # Check the number of worker nodes.  At least two worker nodes are required for production test.
    number_of_workers=$(kubectl get node --no-headers --selector='!node-role.kubernetes.io/control-plane' | wc -l)
    if [[ number_of_workers -lt 2 ]]; then
        logInfo "[ERROR] Insufficient number of worker nodes.  At least two worker nodes are required for conformance tests."
        exit 1
    fi

    logInfo "Testing k8s conformance"
    # Validate external traffic
    curl -L -s -o /dev/null -w "%{http_code}" https://www.oracle.com -m 15
    check_exit_code $? "Checking external traffic access"
    logInfo "Checking golang if that is installed int he system"
    validate_golang

    # Install latest sonobuys
    go get -u -v github.com/heptio/sonobuoy
    check_exit_code $? "go get sonobuoy"
    export PATH=$PATH:$(go env GOPATH)/bin

    kubernetesVersion=$(kubectl version --short | grep Server | cut -d':' -f2  | cut -d'+' -f1 | cut -d'.' -f1-2 | awk '{ print $1 }')
    # wait 2 hours for tests to finish
    sonobuoy run --wait=7200 --kube-conformance-image-version=${kubernetesVersion}
    return_code=$?
    results=$(sonobuoy retrieve)
    if [ $return_code -eq 0 ] && [ $results == "" ]; then
         logInfo "[ERROR] Conformance tests failed with no result file. Please check your configuration"
         exit 1
    fi
    sonobuoy e2e $results | grep "failed tests: 0"
    if [ $? -ne 0 ]; then
        logInfo "[ERROR] Conformance tests failed"
        sonobuoy e2e $results --show failed
        exit 1
    else
        resultsDir="results-k8s-$kubernetesVersion-$(date "+%F-%T")"
        mkdir -p ./$resultsDir; tar xzf $results -C ./$resultsDir
        # https://github.com/cncf/k8s-conformance/blob/master/instructions.md
        logInfo "for conformance logs see directory: $resultsDir"
        logInfo "[SUCCESS] Conformance tests passed"
    fi
};

function print_help() {
    logInfo "Rune this script with either no arguemnt or optional argument -s to enable snobouy test"
    exit 0
};

function validate_ks_pods() {
    for i in $(seq 10); do
        all_pods=$(kubectl get po -n kube-system --no-headers)
        number_of_running_pods=$(logInfo "${all_pods}" | grep Running | wc -l)
        number_of_total_pods=$(logInfo "${all_pods}" | wc -l)

        logInfo "Running pods: $number_of_running_pods Total Pods: $number_of_total_pods"

        # Exit the loop on the 10th try
        if [[ $i -eq 10 ]]; then
            do_exit "Number of pods running in the kube-system namespace is not sufficient, Running pods: $count Total Pods: $len" 1
        fi

        if [[ ${number_of_running_pods} -eq ${number_of_total_pods} ]]; then
            break
        fi

        logInfo "Sleeping to wait for a healthy cluster"
        sleep 30
    done
};

function validate_ks_nodes() {
    oldifs="$IFS"
    IFS=$'\n'
    array=($(kubectl get no -n kube-system --no-headers))
    IFS="$oldifs"
    count=0
    oldifs="$IFS"
    for i in "${array[@]}"
    do
        IFS=" "
        parts=($i)
        pod_stat="${parts[1]}"
        if [[ "$pod_stat" = "Ready" ]]; then
            count=$((count + 1))
        fi
    done
    len=${#array[@]}
    IFS="$oldifs"
    if [[ count -ne len ]]; then
        do_exit "Number of nodes ready in the system is not sufficient, Ready nodes: $count Total Nodess: $len" 1
    fi
    logInfo "Ready nodes: $count Total Nodes: $len"

};


function main {
    # Set default namespace for Jenkins/Pipeline compatibility
    kubectl config set-context --current --namespace=default

    validate_ks_nodes
    validate_ks_pods

    number_of_masters=$(kubectl get no --no-headers -l node-role.kubernetes.io/control-plane= | wc -l)
    number_of_nodes=$(kubectl get no --no-headers | wc -l)
    if [[ number_of_masters -eq number_of_nodes ]]; then
        logInfo "WARNING Removing taint from master. You can't add this back :)"
        # Give the user a small chance of exiting the script before this happens
        sleep 10
        randomMaster=$(kubectl get node --selector='node-role.kubernetes.io/control-plane' -o wide --no-headers | awk '{ print $1 }' | head -$(( ( RANDOM % 3 )  + 1 )) | tail -1)
        kubectl taint nodes ${randomMaster} node-role.kubernetes.io/control-plane- || true
    fi

    validate_system

    logInfo "Running sniff tests"
    run_sniff_tests

    # If -s (sonobuoy) run conformance tests
    if [[ ${RUN_SNO} -eq 1 ]] ; then
        run_conformance_tests
    fi
    logInfo "Successfully ran k8s tests! :)"
};

RUN_SNO=0
while getopts ":s" opt; do
  case ${opt} in
    s ) RUN_SNO=1
      ;;
    \? ) logInfo "Usage: cmd [-h] [-t]"
      ;;
  esac
done

logInfo "Running basic kubernetes testing on a cluster"
main
