# OCI Cluster API Provider

Clusters are deployed to Oracle Cloud Infrastructure (OCI) using the OCI
Cluster API Provider.  Cluster API is an API implemented as Kubernetes custom
resources that are serviced by applications running in a Kubernetes cluster.

Cluster API has a large interface.  Please refer to the [community documentation](https://cluster-api.sigs.k8s.io/)
for a complete description of the configuration options that are available.


## Terminology

The controllers that implement Cluster API run inside a Kubernetes cluster.
These clusters are known as "management clusters".  Management clusters control
the lifecycles of other clusters, known as "workload clusters".  A workload
cluster can be its own management cluster.

Using Cluster API always requires that a Kubernetes cluster is available.  If
a cluster is not available, the Oracle Cloud Native Environment CLI will create
one using the [libvirt](libvirt.md) provider.  It is referred to as a 
"boostrap cluster" or an "ephemeral cluster" depending on the context.

## Configuration

Creating a cluster in OCI using Cluster API requires credentials to an existing
OCI tenancy.  The privileges that are required depend on the configuration of
the cluster that is created.  For some types of deployments, it may be enough
to be able to create and destroy compute instances.  For other deployments, more
privilege is required.

A valid OCI configuration file is required.  Here is an example

```
$ cat ~/.oci/config
[DEFAULT]
user=ocid1.user.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
fingerprint=aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
tenancy=ocid1.tenancy.oc1..aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
region=us-ashburn-1
key_file=/home/myuser/.oci/key.pem
```
Clusters are deployed into specific compartments.  The OCI provider requires
that a comparment is available.  The compartment can be specified in the
configuration defaults or in the configuration for a specific cluster.
Compartments can be specified either by the OCID or by its path in the
compartment hierarchy (e.g. "parentcompartment/mycompartment".

```
$ cat ~/.ocne/defaults.yaml
providers:
  oci:
    compartment: parentcompartment/mycompartment
```

## Creating a Cluster API Cluster

A default cluster can be installed by specifying the "oci" provider with no
additional arguments.  The CLI will detect if a bootstrap cluster is available.
If not, it will start an ephemeral cluster to act as a bootstrap cluster.  Any
resources required to start the ephemeral cluster are fetched and installed.
Once the bootstrap cluster is available, the configured compartment is checked
for compatible compute images.  If there are no images available, they will be
uploaded and imported.  Once this process is complete, all Cluster API providers
are installed into the bootstrap cluster.  When they are started, the Cluster
API resources are installed into the bootstrap cluster.

```
$ ocne cluster start --provider oci
INFO[2024-06-10T17:43:52Z] Installing cert-manager into cert-manager: ok
INFO[2024-06-10T17:43:53Z] Installing core-capi into capi-system: ok
INFO[2024-06-10T17:43:53Z] Installing capoci into cluster-api-provider-oci-system: ok
INFO[2024-06-10T17:43:53Z] Installing bootstrap-capi into capi-kubeadm-bootstrap-system: ok
INFO[2024-06-10T17:43:53Z] Installing control-plane-capi into capi-kubeadm-control-plane-system: ok
INFO[2024-06-10T17:43:54Z] Waiting for Core Cluster API Controllers: ok
INFO[2024-06-10T17:43:54Z] Waiting for Kubadm Boostrap Cluster API Controllers: ok
INFO[2024-06-10T17:43:54Z] Waiting for Kubadm Control Plane Cluster API Controllers: ok
INFO[2024-06-10T17:43:54Z] Waiting for OCI Cluster API Controllers: ok
INFO[2024-06-10T17:43:54Z] Preparing pod used to create image
INFO[2024-06-10T17:44:00Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok
INFO[2024-06-10T17:44:00Z] Getting local boot image for architecture: arm64
Getting image source signatures
Copying blob cdd3f185fcec done   |
Copying config 2ee1288c5b done   |
Writing manifest to image destination
INFO[2024-06-10T17:44:37Z] Copying boot image to pod ocne-system/ocne-image-builder: ok
INFO[2024-06-10T17:44:37Z] Modifying the image ignition provider type
INFO[2024-06-10T17:45:40Z] Copying boot image from pod ocne-system/ocne-image-builder: ok
INFO[2024-06-10T17:45:40Z] New boot image was created successfully at /home/opc/.ocne/images/boot.qcow2-1.28.3-arm64.oci
INFO[2024-06-10T17:46:02Z] Uploading image to object storage: ok
INFO[2024-06-10T17:46:03Z] Preparing pod used to create image
INFO[2024-06-10T17:46:09Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok
INFO[2024-06-10T17:46:09Z] Getting local boot image for architecture: amd64
Getting image source signatures
Copying blob b429dce0e73f done   |
Copying config 7779d76b75 done   |
Writing manifest to image destination
INFO[2024-06-10T17:46:41Z] Copying boot image to pod ocne-system/ocne-image-builder: ok
INFO[2024-06-10T17:46:41Z] Modifying the image ignition provider type
INFO[2024-06-10T17:47:37Z] Copying boot image from pod ocne-system/ocne-image-builder: ok
INFO[2024-06-10T17:47:37Z] New boot image was created successfully at /home/opc/.ocne/images/boot.qcow2-1.28.3-amd64.oci
INFO[2024-06-10T17:47:58Z] Uploading image to object storage: ok
INFO[2024-06-10T17:58:04Z] Importing control plane image: [##########]: ok
INFO[2024-06-10T17:58:44Z] Importing worker image: [##########]: ok
INFO[2024-06-10T17:58:50Z] Applying Cluster API resources
INFO[2024-06-10T17:59:30Z] Waiting for kubeconfig: ok
INFO[2024-06-10T18:01:10Z] Waiting for the Kubernetes cluster to be ready: ok
INFO[2024-06-10T18:01:10Z] Installing applications into workload cluster
INFO[2024-06-10T18:01:11Z] Installing oci-ccm into kube-system: ok
INFO[2024-06-10T18:01:11Z] Installing flannel into kube-flannel: ok
INFO[2024-06-10T18:01:12Z] Installing ui into ocne-system: ok
INFO[2024-06-10T18:01:12Z] Installing app-catalog into ocne-system: ok
INFO[2024-06-10T18:01:12Z] Kubernetes cluster was created successfully
^CFO[2024-06-10T18:02:18Z] Waiting for the UI to be ready: waiting


[opc@instance-20240110-1300 ocne]$ kubectl -n ocne edit machinedeployments.cluster.x-k8s.io ocne-md-0
machinedeployment.cluster.x-k8s.io/ocne-md-0 edited
[opc@instance-20240110-1300 ocne]$ kubectl -n ocne get machinedeployments.cluster.x-k8s.io ocne-md-0
NAME             CLUSTER     REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE       AGE   VERSION
ocne-md-0   ocne   2                  2         2             ScalingUp   5m    v1.28.3

[opc@instance-20240110-1300 ocne]$ kubectl get nodes -o wide
NAME                            STATUS   ROLES           AGE    VERSION         INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                   KERNEL-VERSION                      CONTAINER-RUNTIME
ocne-control-plane-76vkg   Ready    control-plane   4m8s   v1.28.3+3.el8   10.0.0.2      <none>        Oracle Linux Server 8.10   5.15.0-206.153.7.1.el8uek.aarch64   cri-o://1.28.4
ocne-md-0-9v8bd-bmtlj      Ready    <none>          74s    v1.28.3+3.el8   10.0.72.245   <none>        Oracle Linux Server 8.10   5.15.0-206.153.7.1.el8uek.x86_64    cri-o://1.28.4
ocne-md-0-9v8bd-s7cmq      Ready    <none>          9s     v1.28.3+3.el8   10.0.78.64    <none>        Oracle Linux Server 8.10   5.15.0-206.153.7.1.el8uek.x86_64    cri-o://1.28.4
```

## Generating a Cluster Template

The provider defaults result in a useful cluster, but in most cases additional
configuration is required.  To customize the deployment, generate a template to
use as a basis for the cluster.

The template command is responsive to the cluster configuration and local
defaults.  It also fetches things like compute image OCIDs from the configured
compartment automatically.

```
$ ocne cluster template
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  labels:
    cluster.x-k8s.io/cluster-name: "ocne"
  name: "ocne"
  namespace: "ocne"
spec:
  clusterNetwork:
    pods:
…
kind: OCIMachineTemplate
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
metadata:
  name: "ocne-control-plane"
  namespace: "ocne"
spec:
  template:
    spec:
      imageId: "ocid1.image.oc1.iad.aaaaaaaaexb4nt3gupt3rneq4g35u543bkhmzqqj2j2jphsw7q2zbft4mp2a"
      compartmentId: "ocid1.compartment.oc1..aaaaaaaahimqcfto5vfgp5odmuvxjjbfmtwuiyx5tmaril4rwvnlpxmdmu7a"
      shape: "VM.Standard.A1.Flex"
      shapeConfig:
        ocpus: "2"
…
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: OCIMachineTemplate
metadata:
  name: "ocne-md-0"
  namespace: "ocne"
spec:
  template:
    spec:
      imageId: "ocid1.image.oc1.iad.aaaaaaaaddkwz6amwuqwuvv4vjdcchdtk6r4xnxpehcqiu5sksrltw4upcoq"
      compartmentId: "ocid1.compartment.oc1..aaaaaaaahimqcfto5vfgp5odmuvxjjbfmtwuiyx5tmaril4rwvnlpxmdmu7a"
      shape: "VM.Standard.E4.Flex"
      shapeConfig:
        ocpus: "4"
```

## Using a Configuration File to Generate a Template and Start a Cluster

A configuration file can be used to populated template values as well as start
a cluster.

```
$ cat capi.yaml
provider: oci
name: ocne
workerNodes: 2
clusterDefinition: mycluster.oci
providers:
  oci:
    compartment: myparent/mycompartment

$ ocne cluster template -c capi.yaml > mycluster.oci
$ ocne cluster start -c capi.yaml
...
```

## Managing Cluster API Clusters

Once a cluster has been deployed, it is managed using the Cluster API resources
in the management cluster.  Please refer to the community documentation for
details.

## Self-Managed Clusters

A workload cluster can be its own management cluster.  This is known as a
"self-managed" cluster.  Once the cluster has been deployed by a boostrap
cluster, the API resources are migrated from the bootstrap cluster into the
new cluster.

Fully deleting a self-managed cluster requires that a second cluster is
available to run the controllers.  This is because the final stages of cluster
destruction will terminate any remaining compute instances and by extension
cluster nodes.  When this is complete, there will be no compute instances
available to run the controllers.  If a second cluster is not available, an
ephemeral cluster is created to service this need.

## Deleting a Cluster
You can delete the cluster as follows:
```
ocne cluster delete --cluster-name ocne
```
If the cluster does not appear in the output of `ocne cluster ls`, an error may have occurred during cluster creation (e.g., the command was manually aborted). An alternative way to delete the cluster is to specify the cluster config file.
```
ocne cluster delete --config capi.yaml
```

See that the CAPI cluster is gone:
```
kubectl get cluster -A
No resources found
```