# Phase Two: Oracle Cloud Native Environment 2.0 OCK Migration

### Version: v0.0.1-draft

## Overview
Instructions for performing an in-place upgrade of a Kubernetes cluster from Oracle Cloud Native Environment 1.x to 2.x.

## Prerequisite

Identify the VM type and Kubernetes of the existing Oracle Cloud Native Environment 1.x cluster and generate the OS image.

## OCK 2.0 Upgrade Steps

Set up a byo cluster for the existing Oracle Cloud Native Environment 1.x cluster using the 2.0 CLI. 
The example below assumes Kubernetes 1.26, change it to your version.

1. Create an OCI OCK image
    ```
    ocne image create --arch amd64 --type oci --version 1.26
    ocne image upload --arch amd64 --type oci --version 1.26 --bucket <oci-bucket-name> --compartment <oci-compartment-name> --image-name ocnos126 --file /home/opc/.ocne/images/boot.qcow2-1.26-amd64.oci
    ```
2. Create a Oracle Cloud Native Environment CLI config file using byo provider in ~/.ocne, for example ~/.ocne/byo.yaml
    ```
    provider: byo
    name: ocne1x
    kubernetesVersion: <version of 1.x cluster> # e.g. 1.26
    loadBalancer: n.n.n.n  # virtualIp for control plane load balancer
    providers:
      byo:
        networkInterface: ens3
    ```
3. Determine the control plane load balancer type of the existing 1.x cluster (loadBalancer or virtualIp) and its IP address. Update line 3 with it.
4. Determine networkInterface, it may need to be updated on the per-node basis in the [Prepare the upgrade step](#prepare-the-upgrade). Update line 6 with it.
5. If the node to be updated is an OCI compute instance and OCI-CSI is desired to be used (for example, OCI-CCM uses OCI-CSI), the following section should be added to the config file ~/.ocne/byo.yaml
    ```
    extraIgnitionInline: |
      variant: fcos
      version: 1.1.0
      systemd:
        units:
          - name: iscsid.service
            enabled: true
    ```
6. Obtain the kubeconfig from the existing 1.x cluster and copy to ~/.kube/kubeconfig.<CLUSTER_NAME>, in this example, ~/.kube/kubeconfig.ocne1x.


## Prepare the upgrade

1. Find the networkInterface bound by the target node IP and update it in ~/.ocne/byo.yaml if necessary.
2. Find the Kubernetes node name for the target node and record it to use later, e.g., $TARGET_NODE 
    ```
    KUBECONFIG=~/.kube/kubeconfig.ocne1x
    kubectl get nodes
    NAME                        STATUS   ROLES           AGE   VERSION
    cz-ocne-control-plane-001   Ready    control-plane   16m   v1.26.6+1.el8
    cz-ocne-control-plane-002   Ready    control-plane   13m   v1.26.6+1.el8
    cz-ocne-worker-001          Ready    <none>          10m   v1.26.6+1.el8
    cz-ocne-worker-002          Ready    <none>          16m   v1.26.6+1.el8

    TARGET_NODE=cz-ocne-control-plane-001
    ```
3. Drain the target node
    ```
    kubectl drain $TARGET_NODE
    ```
4. Reset the target node, run `kubeadm reset -f` on the node
    ```
    echo "chroot /hostroot kubeadm reset -f " | KUBECONFIG=~/.kube/kubeconfig.ocne1x ocne cluster console -N $TARGET_NODE
    ```
5. If the target node is a control plane node, stop or shutdown the target node host machine. This will make the following kubectl operations more responsive as the target node status may transit to NotReady after reset:
    ```
    kubectl get nodes
    NAME                        STATUS                        ROLES           AGE    VERSION
    cz-ocne-control-plane-001   NotReady,SchedulingDisabled   control-plane   106m   v1.26.6+1.el8
    cz-ocne-control-plane-002   Ready                         control-plane   103m   v1.26.6+1.el8
    cz-ocne-worker-001          Ready                         <none>          100m   v1.26.6+1.el8
    cz-ocne-worker-002          Ready                         <none>          106m   v1.26.6+1.el8
    ```
6. Generate the ignition file by running CLI
The CLI cluster join command will display two messages that have the cert-key and token string.
You must use those values in section 7a and 7b.Also be sure to create run those commands on a 
control plane node different from the target node.

   6a. If the target node is a **control plane node**, and the control-plane.ign is never generated or the control-plane.ign was generated over two hours ago
    ```
    # for control plane node
    ocne cluster join -c ~/.ocne/byo.yaml -k ~/.kube/kubeconfig.ocne1x -n 1 > control-plane.ign
    ```
   
   6b. If the target node is a **worker node**, and the worker.ign is never generated or the worker.ign was generated over two hours ago
    ```
    # for worker node
    ocne cluster join -c ~/.ocne/byo.yaml -k ~/.kube/kubeconfig.ocne1x -w 1 > worker.ign
    ```
7. Generate the join token and upload certs in case of control plane as per instruction printed from the previous step of running "ocne cluster join"

   7a. If the target node is a **control plane node**, and the commands were never executed or executed over two hours ago
    ```
    # The [control-plane-node] could be any of the control-plane-nodes other than the target node
    echo "chroot /hostroot kubeadm init phase upload-certs --certificate-key **** --upload-certs" | ocne cluster console --node [control-plane-node]
    echo "chroot /hostroot kubeadm token create ****" | ocne cluster console --node [control-plane-node]
    ```
   
   7b. If the target node is a **worker node**, and the commands were never executed or executed over two hours ago
    ```
    kubeadm token create ****
    ```

## Upgrade the node


