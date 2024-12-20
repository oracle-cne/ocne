**NOTE: This is a developer release**

# OLVM Cluster API Provider
The OLVM Cluster API Provider allows you to create Kubernetes clusters on an
existing OLVM deployment. The cluster nodes can be spread
across multiple OLVM hosts, where both the control plane and worker nodes can
be scaled in and out as desired. Using the Oracle Cloud Native Environment CLI (`ocne`), 
you can create and upload the required OLVM compatible OCK image to the OLVM deployment,
then create the cluster.

## Terminology
Cluster API is an API implemented as Kubernetes custom
resources (CRDs) that are serviced by applications running in a Kubernetes cluster.
Cluster API has a large interface.  Please refer to the [community documentation](https://cluster-api.sigs.k8s.io/)
for a complete description of the configuration options that are available.

The controllers that implement Cluster API run inside a Kubernetes cluster.
These clusters are known as "management clusters".  Management clusters control
the lifecycles of other clusters, known as "workload clusters".  A workload
cluster can be its own management cluster. 

Using Cluster API always requires that a Kubernetes cluster is available.  If
a cluster is not available, the Oracle Cloud Native Environment CLI will create
one using the [libvirt](libvirt.md) provider.  It is referred to as a
"boostrap cluster" or an "ephemeral cluster" depending on the context.

The OLVM Cluster API Provider implements an infrastructure Cluster controller along with
an infrastructure Machine controller.  Both are housed in a single operator. This
provider interacts with OLVM using the [oVirt REST API.](https://www.ovirt.org/documentation/doc-REST_API_Guide/)

## Prerequisites
* You must have an existing OLVM installation that can be accessed via a set of external IPs.
* You will need an IP for the Kubernetes control plane node and an IP for each cluster node.
* The CA certificate used for the oVirt rest API must be downloaded to a local file, even if it is not self-signed.  See [oVirt CA](https://www.ovirt.org/documentation/doc-REST_API_Guide/#obtaining-the-ca-certificate)

## Restrictions
* The OLVM Cluster API Provider does not support DHCP, you must allocate a range of external IPs.

#  Oracle Cloud Native Environment CLI cluster configuration for OLVM
Before running any `ocne` command related to OLVM, you must create the configuration file.
The sample below shows a fully functional configuration (with some redacted fields.  This
configuration introduces a new olvm provider with custom configuration with 4 sections:

* Global OLVM configuration
* OLVMCluster configuration
* OLVMMachine configuration for the control plane nodes
* OLVMMachine configuraiton for the worker nodes

```
name: demo
workerNodes: 2
controlPlaneNodes: 3
virtualIp: 100.101.70.160 
password: "$6...1"
provider: olvm
providers:
  olvm:
    networkInterface: enp1s0
    namespace: olvm-cluster
    olvmCluster:
      ovirtDatacenterName: Default
      olvmVmIpProfile:
        name: default-ip
        gateway: 1.2.3.1
        netmask: 255.255.255.0
        device: enp1s0
        startingIpAddress: 1.2.3.161
      ovirtAPI:
        serverURL: https://ovirt.example.oraclevcn.com/ovirt-engine
        serverCAPath: "~/olvm/ca.crt"
      ovirtOCK:
        storageDomainName: oblock
        diskName: olvm-ock-1.30
        diskSize: 16GB
    controlPlaneMachine:
      memory: "7GB"
      network:
        interfaceType: virtio
        networkName: vlan
        vnicName: nic-1
        vnicProfileName: vlan
      ovirtClusterName: Default
      olvmVmIpProfileName: default-ip
      vmTemplateName: olvm-tmplate-1.30
    workerMachine:
      memory: "16GB"
      network:
        interfaceType: virtio
        networkName: vlan
        vnicName: nic-1
        vnicProfileName: vlan
      ovirtClusterName: Default
      olvmVmIpProfileName: default-ip
      vmTemplateName: olvm-tmplate-1.30

kubernetesVersion: 1.30
proxy:
  httpsProxy: http://www-proxy-example.com:80
  noProxy: .mycorp.com,localhost,127.0.0.1,1.2.3.0/14,nip.io
extraIgnitionInline:
  variant: fcos
  version: 1.5.0
  storage:
    files:
    - path: /etc/resolv.conf
      mode: 0644
      overwrite: false
      contents:
        inline: |
          nameserver 1.2.3.250
```

## Top-levl OLVM configuration
The following global configuration specifies:

**networkInterface**
The interface used by OLVM virtual machines (VMs).

**namespace** 
The namespace where CLUSTER API resources will be created in your management cluster.
```
providers:
  olvm:
    networkInterface: enp1s0
    namespace: olvm-cluster
```

## Cluster configuration

## Machine congfiguration

# Creating a Cluster

## Overview

## Creating the OLVM OCK image

## Uploading the OLVM OCK image

### Creating a VM template

## Starting the cluster

# Creating a Cluster

# Deleting a Cluster
