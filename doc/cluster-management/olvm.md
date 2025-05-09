**NOTE: This is a developer release**

# OLVM Cluster API Provider
The Oracle Linux Virtualization Manager (OLVM) Cluster API Provider allows you to create Kubernetes clusters on an
existing OLVM instance. The cluster nodes can be spread
across multiple OLVM hosts, where both the control plane and worker nodes can
be scaled in and out as desired. Using the Oracle Cloud Native Environment CLI (`ocne`), 
you can create and upload the required OLVM compatible OCK image to the OLVM instance,
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
"bootstrap cluster" or an "ephemeral cluster" depending on the context.

The OLVM Cluster API Provider implements an infrastructure Cluster controller (OLVMCluster CRD) along with
an infrastructure Machine controller (OLVMMachine CRD).  Both are housed in a single operator. This
provider interacts with OLVM using the [oVirt REST API.](https://www.ovirt.org/documentation/doc-REST_API_Guide/)

Machine and OLVMMachines are CAPI resources. There is an OLVM VM created for each Machine.  Each VM contains a single Kubernetes node.

The oVirt instance is the same as the oVirt installation.  It is where the oVirt console runs, the oVirt engine, etc.

The term `external IPs` describes a range of static IPs within the subnet of the network that the VMs are attached to.  These IPs
much be reachable from each node in the cluster, and from the Cluster API boostrap cluster.  The `virtual IP` is the
IP used to access the Kubernetes API server.  This must be also be reachable from each node in the cluster, 
from the Cluster API boostrap cluster, and from the OCNE client.

## IPV4/IPV6 Dual-Stack Support
The OLMV Cluster API Provider supports IPV4 alone or an IPV4/IPV6 dual-stack configuration. An IPV6-only configuration is not supported.

## Prerequisites
* You must have an existing OLVM installation that can be accessed via a set of external IPs.
* You will need an IP for the Kubernetes control plane node and an IP for each cluster node.
* The CA certificate used for the oVirt rest API must be downloaded to a local file, even if it is not self-signed.  See [oVirt CA](https://www.ovirt.org/documentation/doc-REST_API_Guide/#obtaining-the-ca-certificate)
* All the network interface devices should be the same, such as enp1s0.

## Restrictions
* The OLVM Cluster API Provider does not support DHCP, you must allocate a contiguous range of IPs for
the cluster nodes and for the Kubernetes API server (the virtual IP).

#  Oracle Cloud Native Environment CLI cluster configuration for OLVM
Before running any `ocne` command related to OLVM, you must create the configuration file.
The sample below shows a fully functional configuration (with some redacted fields.  This
configuration introduces a new olvm provider with custom configuration with 4 sections:

* Global OLVM configuration
* OLVMCluster configuration
* OLVMMachine configuration for the control plane nodes
* OLVMMachine configuration for the worker nodes

The following YAML shows the required fields only:
```
name: demo
provider: olvm
providers:
  olvm:
    olvmDatacenterName: Default
    olvmOvirtAPIServer:
      serverURL: https://example.com/ovirt-engine
      serverCAPath: "/tmp/ca.crt"
    olvmOCK:
      storageDomainName: olvm-data
      diskName: ock-1.31
      diskSize: 16GB
    controlPlaneMachine:
      olvmOvirtClusterName: Default
      vmTemplateName: ock-1.31
      olvmNetwork:
        networkName: kvm-vlan
        vnicProfileName: kvm-vlan
      virtualMachine:
        memory: "7GB"
        network:
          gateway: 2.3.4.1
          ipv4:
            subnet: 2.3.4.160/24
            ipAddresses: 2.1.4.161/30, 2.3.4.196, 2.3.4.200-2.3.4.220
    workerMachine:
      olvmOvirtClusterName: Default
      vmTemplateName: ock-1.31
      olvmNetwork:
        networkName: kvm-vlan
        vnicProfileName: kvm-vlan
      virtualMachine:
        memory: "16GB"
        network:
          gateway: 1.2.3.1
          interfaceType: virtio
          ipv4:
            autoconf: false
            ipAddresses: 1.2.3.161/30, 1.2.3.196, 1.2.4.200-1.2.4.220
```


The following YAML shows all the fields, including optional fields:
```
name: demo
workerNodes: 1
controlPlaneNodes: 1
podSubnet: 10.244.0.0/16,fdXY:IJKL:MNOP:15::/64
serviceSubnet: 10.96.0.0/12,fdXY:IJKL:42::/112
virtualIp: 1.2.3.160
provider: olvm
providers:
  olvm:
    namespace: olvm
    olvmDatacenterName: Default
    olvmOvirtAPIServer:
      serverURL: https://example.com/ovirt-engine
      serverCAPath: "/tmp/ca.crt"
      credentialsSecret:
        name: olvm-creds
        namespace: olvm
      caConfigMap:
        name: olvm-ca
        namespace: opvm
      insecureSkipTLSVerify: true
    olvmOCK:
      storageDomainName: olvm-data
      diskName: ock-1.31
      diskSize: 16GB
    controlPlaneMachine:
      olvmOvirtClusterName: Default
      vmTemplateName: ock-1.31
      olvmNetwork:
        networkName: kvm-vlan
        vnicName: nic-1
        vnicProfileName: kvm-vlan
      virtualMachine:
        memory: "7GB"
        cpu:
          topology:
            cores: 7
            sockets: 9
            threads: 2
        network:
          gateway: 1.2.3.1
          interface: enp1s0
          interfaceType: virtio
          ipv4:
            subnet: 1.2.3.160/24
            ipAddresses: 1.2.3.161/30, 1.2.3.196, 1.2.3.200-1.2.3.220
          ipv6:
            autoconf: false
            ipAddresses: fdXY:IJKL::2222-fdXY:IJKL::2232, fdXY:ABCX::2000/64
    workerMachine:
      olvmOvirtClusterName: Default
      vmTemplateName: ock-1.31
      olvmNetwork:
        networkName: kvm-vlan
        vnicName: nic-1
        vnicProfileName: kvm-vlan
      virtualMachine:
        memory: "16GB"
        cpu:
          topology:
            cores: 6
            sockets: 8
            threads: 3
        network:
          gateway: 1.2.3.1
          interface: enp1s0
          interfaceType: virtio
          ipv4:
            autoconf: false
            ipAddresses: 1.2.3.161/30, 1.2.3.196, 1.2.4.200-1.2.4.220
          ipv6:
            autoconf: false
            ipAddresses: fdXY:IJKL::2250-fdXY:IJKL::2259, fdXY:ABCX::3000/64
```

## Global fields
The global fields also include the [Oracle Cloud Native Environment Configuration](https://docs.oracle.com/en/operating-systems/olcne/2/cli/config_concept.html)
```
provider: olvm
virtualIp: 1.2.3.100 
podSubnet: 10.244.0.0/16,fdXY:IJKL:MNOP:15::/64
serviceSubnet: 10.96.0.0/12,fdXY:IJKL:42::/112
...
providers:
  olvm:
    namespace: olvm
    olvmDatacenterName: Default
```
**provider**  
The provider must be olvm.

**virtualIp**  
The virtual IP is used as the Kubernetes control plane endpoint (the server field in the kubeconifg file).
This IP must be external, and cannot be in the range used by the VMs.

**podSubnet**  
An IPV4 Pod subnet comma-separated by an optional IPV6 subnet

**serviceSubnet**  
An IPV4 Service subnet comma-separated by an optional IPV6 subnet

**namespace**  
The namespace where CLUSTER API resources will be created in your management cluster.

**ovirtDatacenterName**  
The oVirt datacenter name

## olvmOvirtAPIServer
The olvmOvirtAPIServer section specifies the configuration needed to access the OLVM oVirt REST API server.
```
    olvmOvirtAPIServer:
      serverURL: https://example.com/ovirt-engine
      serverCAPath: "/tmp/ca.crt"
      credentialsSecret:
        name: olvm-creds
        namespace: olvm
      caConfigMap:
        name: olvm-ca
        namespace: opvm
      insecureSkipTLSVerify: true
```
**serverURL**  
The serverURL is the URL of the OLVM oVirt engine that is accessed via the OLVM oVirt REST API.

**serverCAPath**  
The local file that contains the OLVM CA certificate.
Optional if insecureSkipTLSVerify is true.

**credentialsSecret**  
The name and namespace of the OLVM credentials Kubernetes secret. 
Optional.

**caConfigMap**  
The name and namespace of the OLVM Kubernetes ConfigMap containing the OLVM CA.
Optional.

**insecureSkipTLSVerify**  
If true, the skip TLS verify when connecting to OLVM oVirt server.  The CA is not needed or used.

## olvmOCK
The ovirtOCK section specifies the information needed to upload the OLVM OCK image using the `ocne image upload` command.
```text
    olvmOCK:
      storageDomainName: olvm-data
      diskName: ock-1.31
      diskSize: 16GB
```

**storageDomainName**  
The name of an existing oVirt storage domain where the image will be uploaded.

**diskName**  
The name of the disk that will be created in the storage domain as a result of the upload.
This is the disk name that you specify when you create a VM template.

**diskSize**  
The provisioned virtual disk size name to be used for the disk created in the storage domain.  This is the disk space that will
be allocated for the VM regardless of the size of the image on disk.  For example, the image might be 2.5GB, but the provisioned size
could be 16GB.

## controlPlaneMachine
```text
    controlPlaneMachine:
      olvmOvirtClusterName: Default
      vmTemplateName: ock-1.31
      olvmNetwork:
        networkName: kvm-vlan
        vnicName: nic-1
        vnicProfileName: kvm-vlan
      virtualMachine:
        memory: "7GB"
        cpu:
          topology:
            cores: 7
            sockets: 9
            threads: 2
        network:
          gateway: 1.2.3.1
          interface: enp1s0
          interfaceType: virtio
          ipv4:
            subnet: 1.2.3.160/24
            ipAddresses: 1.2.3.161/30, 1.2.3.196, 1.2.3.200-1.2.3.220
          ipv6:
            autoconf: false
            ipAddresses: fdXY:IJKL::2222-fdXY:IJKL::2232, fdXY:ABCX::2000/64
```
**olvmOvirtClusterName**  
The OLVM oVirt cluster. This cluster must exist in the OLVM datacenter.
**Note**This is not to be confused with the OLVMCluster which is a Cluster API resource.

**vmTemplateName**  
The OLVM vmTemplate name.  This must exist in the OLVM instance
(Note: you will need to create a vmTemplate with the OCK image, see instructions later in this document).


### controlPlaneMachine - olvmNetwork
**olvmNetwork.networkName**  
The OLVM network name.  This must exist in the OLVM instance.

**olvmNetwork.vnicName**  
The OLVM vnicName.  The scope of this name is the VM.
Optional.

**olvmNetwork.vnicProfileName**  
The OLVM vnic profile name.  This must exist in the oVirt instance

### controlPlaneMachine - virtualMachine
**memory**  
VM memory allocated for each Kubernetes node.

**network.interface**  
The interface used by VM.  Currently, the value `enp1s0` is required.

**network.interfaceType**  
The interface type.  
Optional, Default is virtio.

### IPV4
The IPV4 configuration is required. 

**network.ipv4.subnet**   
The IPV4 subnet used by the VM.

**network.ipv4.ipAddresses**  
The IPV4 addresses used by the VM.  This is a comma separated list with any combination as follows:
Ranges are inclusive. Space after the comma is optional.
10.1.2.0/30, 10.1.2.10-10.1.2.20, 10.1.2.27

### IPV6
The IPV4 configuration is optional.

**network.ipv6.autoconf**  
If true, then auto-configure the IPV6 using SLAAC.
Optional, default false.

**network.ipv6.ipAddresses**  
The IPV4 addresses used by the VM.  This is a comma separated list with any combination as follows:
Ranges are inclusive. Space after the comma is optional.
10.1.2.0/30, 10.1.2.10-10.1.2.20, 10.1.2.27
Optional




## Machine configuration
The control plane and worker fields are identical, but the values may be different.  These values
apply to all the control plane nodes and worker nodes in the cluster.
```
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

```
**memory**  
VM memory allocated for each Kubernetes node.

**cpu.topology**  
The cpu cores, sockets and threads.  
Optional.  Default is 2 cores, 2 sockets, 1 thread.


**network.Name**
The oVirt network name.  This must exist in the oVirt instance.



## Ovirt CSI Driver configuration (optional)
The ovirt-csi-driver configuration is optional, there is no need to explicitly configure anything. The driver is
automatically installed and the required namespace, credential Secret, CA ConfigMap, and CsiDriver resources are created.
See [ovirt-csi-driver usage example](https://github.com/oracle-cne/ovirt-csi-driver/blob/master/docs/usage-example.md).

Following is the structure of the config showing the default values:
```
providers:
  olvm:
    ovirtCsiDriver:
      install: true
      caProvided: true
      caConfigmapName: ovirt-csi-ca.crt
      controllerPluginName: ovirt-csi-controller
      credsSecretName: ovirt-csi-creds 
      csiDriverName: csi.ovirt.org 
      namespace: ovirt-csi
      nodePluginName: ovirt-csi-node
  ...    
```
**install**
If install is true, install the ovirt-csi-driver and required resources.

**caProvided**
If caProvided is true, the ovirt-csi-driver expects to find the ConfigMap containing the CA and will use that
CA during network connections to the OLVM server.

**caConfigmapName**
The caConfigmapName is the name of the ConfigMap containing the CA.

**credsSecretName**
The credsSecretName is the name of the Secret containing the credentials needed to communicate with the OVLM server.

**controllerPluginName**
The controllerPluginName is the name of the deployment for the controller plugin, which is part of the driver.

**csiDriverName**
The csiDriverName is the name of the CsiDriver.  This name is used when you create a StorageClass, it is the value
of the `provisioner` field.

**namespace**
The namespace where the ovirt-csi-driver and all related resources are created.

**nodePluginName**
The nodePluginName is the name of the daemonset for the node plugin, which is part of the driver.


## Ignition fields
The ignition fields need to be updated with your nameserver IP.  The other fields should stay as is.
```
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
          nameserver <name-server-ip>
```


# Preparing to Create a Cluster
This section describes the steps needed to create a cluster.  Make sure your cluster configuration file
exists and has the correct values as described in previous sections.

## Credentials 
Before using the OLVM Cluster API provider, you need to define the following environment variables on
the machine where you are running `ocne`.

OCNE_OLVM_USERNAME  
OCNE_OLVM_PASSWORD  
OCNE_OLVM_SCOPE  

Use "ovirt-app-api" as the scope, unless you have created a user with a different scope.
The username must have @internal suffix.  So if you log into the OLVM console with "admin", then
the OCNE_OLVM_USERNAME is "admin@internal"

## oVirt REST API CA Certificate
You also must download the oVirt REST API CA certificate and put it into a file referenced by the cluster configuration (see below).
Make sure you only use the second certificate returned by the instructions at [oVirt CA](https://www.ovirt.org/documentation/doc-REST_API_Guide/#obtaining-the-ca-certificate).


## Creating a workload cluster
First create a workload cluster.  Even though and ephemeral cluster is automatically created, you can 
also use any cluster you want as a workload cluster.  For this exercise, we will create the workload cluster.
The workload cluster will be used to create an image so it needs to have enough resources.  The settings
below were used to test this.

~/.ocne/defaults.yaml
```
name: ocne
provider: libvirt
workerNodes: 0
communityCatalog: false
proxy:
  (use your proxies)
providers:
  libvirt:
    controlPlaneNode:
       storage: 30G
       memory: 10000000K
       cpu: 3
```

Now create the workload cluster:
```
ocne cluster start -u false
INFO[2024-12-20T13:37:50-05:00] Creating new Kubernetes cluster with version 1.30 named ocne 
INFO[2024-12-20T13:38:13-05:00] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-12-20T13:38:24-05:00] Installing flannel into kube-flannel: ok 
INFO[2024-12-20T13:38:26-05:00] Installing ui into ocne-system: ok 
INFO[2024-12-20T13:38:27-05:00] Installing ocne-catalog into ocne-system: ok 
INFO[2024-12-20T13:38:27-05:00] Kubernetes cluster was created successfully 
```

## Creating the OLVM OCK image
The first step is to create an OCK image with the default Kubernetes version (1.30 at the time of this writing).

```
export KUBECONFIG=/Users/user/.kube/kubeconfig.ocne.local

ocne image create --type olvm --kubeconfig $KUBECONFIG
INFO[2024-12-20T13:46:23-05:00] Creating Image                               
INFO[2024-12-20T13:46:23-05:00] Preparing pod used to create image           
INFO[2024-12-20T13:46:28-05:00] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-12-20T13:46:28-05:00] Getting local boot image for architecture: amd64 
INFO[2024-12-20T13:46:56-05:00] Uploading boot image to pod ocne-system/ocne-image-builder: ok 
INFO[2024-12-20T13:47:41-05:00] Downloading boot image from pod ocne-system/ocne-image-builder: ok 
INFO[2024-12-20T13:47:41-05:00] New boot image was created successfully at /Users/user/.ocne/images/boot.qcow2-1.30-amd64.olvm 
```

## Uploading the OLVM OCK image
Upload the OCK image that you just created to your 

```
ocne image upload --type olvm --arch amd64 --file  /Users/user/.ocne/images/boot.qcow2-1.30-amd64.olvm   --config /Users/user/.ocne/olvm-demo-cluster-config.yaml --kubeconfig $KUBECONFIG
INFO[2024-12-20T13:49:48-05:00] Starting uploaded OCK image `/Users/user/.ocne/images/boot.qcow2-1.30-amd64.olvm` to disk `demo-1-ock-1.30` in storage domain `oblock` 
INFO[2024-12-20T13:49:48-05:00] Waiting for disk status to be OK             
INFO[2024-12-20T13:49:53-05:00] Waiting for image transfer phase transferring 
INFO[2024-12-20T13:51:56-05:00] Uploading image /Users/user/.ocne/images/boot.qcow2-1.30-amd64.olvm with 2826567680 bytes to demo-1-ock-1.30: ok 
INFO[2024-12-20T13:51:58-05:00] Waiting for image transfer phase finished_success 
INFO[2024-12-20T13:52:11-05:00] Successfully uploaded OCK image    
```

### Creating a VM template
Now you need to use the OLVM oVirt console to create a template that uses the image you just uploaded.

1. Navigate to Compute->Virtual Machines  
2. Click the New button to create a virtual machine  
3. Fill in the form, only change the following fields:  
   
   Name: enter a VM name (this is temporary, it will be deleted after the template is created).  
   Template: leave blank  
   Operating System: Red Hat Enterprise Linux CoreOS  
   Instance Images: click Attach and select the boot.qcow2 disk/image that you created, select the OS (boot) checkbox, which is the last checkbox on the right, then save.

4. After the VM creation is finished, select but do NOT run it, rather click the "Make Template" menu selection.
Make sure the template name matches the vmTemplateName in your Oracle Cloud Native Environment CLI cluster configuration.
5. Delete the VM that was used to create the template.

# Create the cluster
Now you are ready to create a cluster.  As the cluster is being created, you can look at the Virtual Machine page in your OLVM console and see the VMs being created.
First the control plane VM (Kubernetes node) is created, followed by the worker VM/

```
ocne cluster start --provider olvm  --cluster-name demo --config /Users/user/.ocne/olvm-demo-cluster-config.yaml
INFO[2024-12-20T14:09:31-05:00] Installing cert-manager into cert-manager: ok 
INFO[2024-12-20T14:09:32-05:00] Installing core-capi into capi-system: ok 
INFO[2024-12-20T14:09:33-05:00] Installing olvm-capi into cluster-api-provider-olvm: ok 
INFO[2024-12-20T14:09:34-05:00] Installing bootstrap-capi into capi-kubeadm-bootstrap-system: ok 
INFO[2024-12-20T14:09:35-05:00] Installing control-plane-capi into capi-kubeadm-control-plane-system: ok 
INFO[2024-12-20T14:10:05-05:00] Waiting for Core Cluster API Controllers: ok 
INFO[2024-12-20T14:10:25-05:00] Waiting for Olvm Cluster API Controllers: ok 
INFO[2024-12-20T14:10:45-05:00] Waiting for Kubadm Boostrap Cluster API Controllers: ok 
INFO[2024-12-20T14:11:15-05:00] Waiting for Kubadm Control Plane Cluster API Controllers: ok 
INFO[2024-12-20T14:11:15-05:00] Applying Cluster API resources               
INFO[2024-12-20T14:11:17-05:00] Waiting for kubeconfig: ok       
INFO[2024-12-20T14:13:35-05:00] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-12-20T14:13:35-05:00] Installing applications into workload cluster 
INFO[2024-12-20T14:13:43-05:00] Installing flannel into kube-flannel: ok 
INFO[2024-12-20T14:13:45-05:00] Installing ui into ocne-system: ok 
INFO[2024-12-20T14:13:46-05:00] Installing ocne-catalog into ocne-system: ok 
INFO[2024-12-20T14:13:46-05:00] Kubernetes cluster was created successfully  
INFO[2024-12-20T14:16:47-05:00] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/Users/user/.kube/kubeconfig.demo' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y
```

The kubeconfig file needed to access your new CAPI cluster is at ~/.ocne/kubeconfig.<cluster-name>
You can see the Kubernetes nodes and access your new cluster as follows:
```
kubectl --kubeconfig ~/.kube/kubeconfig.demo get node
NAME                       STATUS   ROLES           AGE     VERSION
demo-control-plane-l2zrs   Ready    control-plane   2m27s   v1.30.3+1.el8
demo-md-0-v5xsk-hsbcw      Ready    <none>          8s      v1.30.3+1.el8
```

# Scale the cluster
You can scale the cluster control plane and worker nodes independently.

Scale the control plane to 3 nodes:
```
kubectl get kubeadmcontrolplane -A
NAMESPACE      NAME                 CLUSTER   INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE     VERSION
olvm-cluster   demo-control-plane   demo      true          true                   1          1       1         0             9m13s   v1.30.3

kubectl scale kubeadmcontrolplane  -n olvm-cluster   demo-control-plane --replicas 3
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/demo-control-plane scaled
```

Scale the workers to 5 nodes:
```
kubectl get machinedeployment -A
NAMESPACE      NAME        CLUSTER   REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE     AGE   VERSION
olvm-cluster   demo-md-0   demo      1          1       1         0             Running   11m   v1.30.3

kubectl scale machinedeployment -n olvm-cluster   demo-md-0  --replicas 5
machinedeployment.cluster.x-k8s.io/demo-md-0 scaled
```
As mentioned previously, you can observe the Kubernetes cluster being scaled by looking at the OLVM console Virtual Machine page.
The Cluster API controller creates control plane nodes one at the time, waiting until the new node is ready before
creating the next.  However, all the worker nodes are created concurrently.

Also, can watch the CAPI infrastructure machines being created (IPs redacted):
```
kubectl get OLVMMachine -A -o wide
NAMESPACE      NAME                       CLUSTER   READY   AGE     OVIRT-CLUSTER   MEMORY   CORES   SOCKETS   VMSTATUS   VMIPADDRESS
olvm-cluster   demo-control-plane-l2zrs   demo      true    14m     Default         7GB      2       2         up         1.2.3.1
olvm-cluster   demo-control-plane-mkd4p   demo      true    2m19s   Default         7GB      2       2         up         1.2.3.2
olvm-cluster   demo-control-plane-t5gvv   demo      true    5m      Default         7GB      2       2         up         1.2.3.3
olvm-cluster   demo-md-0-v5xsk-hsbcw      demo      true    14m     Default         16GB     2       2         up         1.2.3.4
olvm-cluster   demo-md-0-v5xsk-s9sm4      demo      true    3m2s    Default         16GB     2       2         up         1.2.3.5
olvm-cluster   demo-md-0-v5xsk-sfmfg      demo      true    3m2s    Default         16GB     2       2         up         1.2.3.6
olvm-cluster   demo-md-0-v5xsk-v6dw9      demo      true    3m2s    Default         16GB     2       2         up         1.2.3.7
olvm-cluster   demo-md-0-v5xsk-wfhjg      demo      true    3m2s    Default         16GB     2       2         up         1.2.3.8
```

Eventually, you should see all the nodes created and ready:
```
kubectl --kubeconfig ~/.kube/kubeconfig.demo get node
NAME                       STATUS   ROLES           AGE     VERSION
demo-control-plane-l2zrs   Ready    control-plane   14m     v1.30.3+1.el8
demo-control-plane-mkd4p   Ready    control-plane   2m32s   v1.30.3+1.el8
demo-control-plane-t5gvv   Ready    control-plane   4m56s   v1.30.3+1.el8
demo-md-0-v5xsk-hsbcw      Ready    <none>          12m     v1.30.3+1.el8
demo-md-0-v5xsk-s9sm4      Ready    <none>          3m1s    v1.30.3+1.el8
demo-md-0-v5xsk-sfmfg      Ready    <none>          3m      v1.30.3+1.el8
demo-md-0-v5xsk-v6dw9      Ready    <none>          3m6s    v1.30.3+1.el8
demo-md-0-v5xsk-wfhjg      Ready    <none>          3m      v1.30.3+1.el8
```

See the pods:
```
kubectl --kubeconfig ~/.kube/kubeconfig.demo get pods -A
NAMESPACE      NAME                                               READY   STATUS    RESTARTS        AGE
kube-flannel   kube-flannel-ds-4n8sc                              1/1     Running   1 (3m35s ago)   4m7s
kube-flannel   kube-flannel-ds-d2x6x                              1/1     Running   0               15m
kube-flannel   kube-flannel-ds-fwzk7                              1/1     Running   1 (3m35s ago)   4m6s
kube-flannel   kube-flannel-ds-p6ktg                              1/1     Running   0               4m10s
kube-flannel   kube-flannel-ds-pgtf9                              1/1     Running   1 (5m29s ago)   6m2s
kube-flannel   kube-flannel-ds-sbbwm                              1/1     Running   1 (3m4s ago)    3m36s
kube-flannel   kube-flannel-ds-tpwfq                              1/1     Running   1 (3m35s ago)   4m6s
kube-flannel   kube-flannel-ds-vvtrq                              1/1     Running   1 (13m ago)     13m
kube-system    coredns-f7d444b54-gbcm5                            1/1     Running   0               15m
kube-system    coredns-f7d444b54-smj8z                            1/1     Running   0               15m
kube-system    etcd-demo-control-plane-l2zrs                      1/1     Running   0               15m
kube-system    etcd-demo-control-plane-mkd4p                      1/1     Running   0               3m37s
kube-system    etcd-demo-control-plane-t5gvv                      1/1     Running   0               6m1s
kube-system    kube-apiserver-demo-control-plane-l2zrs            1/1     Running   0               15m
kube-system    kube-apiserver-demo-control-plane-mkd4p            1/1     Running   0               3m37s
kube-system    kube-apiserver-demo-control-plane-t5gvv            1/1     Running   0               6m1s
kube-system    kube-controller-manager-demo-control-plane-l2zrs   1/1     Running   0               15m
kube-system    kube-controller-manager-demo-control-plane-mkd4p   1/1     Running   0               3m37s
kube-system    kube-controller-manager-demo-control-plane-t5gvv   1/1     Running   0               6m1s
kube-system    kube-proxy-55wp4                                   1/1     Running   0               4m7s
kube-system    kube-proxy-599b2                                   1/1     Running   0               6m2s
kube-system    kube-proxy-6lc2t                                   1/1     Running   0               4m10s
kube-system    kube-proxy-f5md2                                   1/1     Running   0               15m
kube-system    kube-proxy-fqm9g                                   1/1     Running   0               3m36s
kube-system    kube-proxy-kwp2d                                   1/1     Running   0               4m6s
kube-system    kube-proxy-pmvnw                                   1/1     Running   0               13m
kube-system    kube-proxy-vshtx                                   1/1     Running   0               4m6s
kube-system    kube-scheduler-demo-control-plane-l2zrs            1/1     Running   0               15m
kube-system    kube-scheduler-demo-control-plane-mkd4p            1/1     Running   0               3m37s
kube-system    kube-scheduler-demo-control-plane-t5gvv            1/1     Running   0               6m1s
ocne-system    ocne-catalog-578c959566-5gxbd                      1/1     Running   0               15m
ocne-system    ui-84dd57ff69-vv2c6                                1/1     Running   0               15m
```

# Deleting a Cluster
Finally, you can delete the cluster as follows:
```
ocne cluster delete --cluster-name demo
INFO[2024-12-20T14:32:30-05:00] Installing cert-manager into cert-manager: ok 
INFO[2024-12-20T14:32:30-05:00] Installing core-capi into capi-system: ok 
INFO[2024-12-20T14:32:31-05:00] Installing olvm-capi into cluster-api-provider-olvm: ok 
INFO[2024-12-20T14:32:31-05:00] Installing bootstrap-capi into capi-kubeadm-bootstrap-system: ok 
INFO[2024-12-20T14:32:32-05:00] Installing control-plane-capi into capi-kubeadm-control-plane-system: ok 
INFO[2024-12-20T14:32:32-05:00] Waiting for Kubadm Control Plane Cluster API Controllers: ok 
INFO[2024-12-20T14:32:32-05:00] Waiting for Olvm Cluster API Controllers: ok 
INFO[2024-12-20T14:32:33-05:00] Waiting for Kubadm Boostrap Cluster API Controllers: ok 
INFO[2024-12-20T14:32:42-05:00] Waiting for Core Cluster API Controllers: ok 
INFO[2024-12-20T14:32:42-05:00] Deleting Cluster olvm-cluster/demo           
INFO[2024-12-20T14:33:09-05:00] Waiting for deletion: ok     
```
If the cluster does not appear in the output of `ocne cluster ls`, an error may have occurred during cluster creation (e.g., the command was manually aborted). An alternative way to delete the cluster is to specify the cluster config file.
```
ocne cluster delete --config /Users/user/.ocne/olvm-demo-cluster-config.yaml
```


See that the CAPI cluster is gone:
```
kubectl get cluster -A
No resources found
```

# Troubleshooting
This section list troubleshooting tips.

## Proxies
A common problem is misconfigured proxies.  Make sure the proxy settings in your cluster configuration file is correct for both httpsProxy and noProxy.

## External IPs.
You must have available IPs that are reachable from the machine where you are running the Oracle Cloud Native Environment CLI.
This includes the virtual IP and all the IPs for the Kubernetes nodes.

## Capacity
Make sure your workload cluster has the capacity as specified in the instructions in the document.  If not, then you will see
problems like pods being evicted, etc.

## Cleanup
If for any reason the `ocne cluster start` command fails.  You need to do some manual steps to completely cleanup.

1. ocne cluster delete   --cluster-name demo
2. kubectl delete cl -n olvm-cluster demo
3. rm ~/.kube/kubeconfig.demo


