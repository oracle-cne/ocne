# Updating Clusters

Keeping a Kubernetes cluster and its components up to date is an essential part
of maintaing the cluster.

## Updating Delivery

All updates are delivered by container images.  Every node in a Kubernetes
cluster periodically polls a container image for changes.  When a change is
detected, the image is automatically pulled, verified, and unpacked.

## Updating a Kubernetes Cluster

The process for updating a Kubernetes cluster from one version to the next
involves staging an update to the next version and then updating each node
in sequence.

### Staging an Update

The `ocne cluster stage` command is used to stage a cluster upgrade.  The
command does different things depending on the target cluster and the provider.
For providers implemented with Cluster API, staging an update involves creating
new boot media, uploading that media, creating new machine templates, and
suggesting patches that could be applied to Cluster API resources to perform
the upgrade of the cluster nodes.  For providers that are not implemented with
Cluster API, staging an update sets various configuraiton options in the cluster
and configures each cluster node to upgrade to the next Kubernetes minor version.

```
$ ocne cluster stage --version 1.30
```

### Updating a Cluster Managed By Cluster API

Clusters managed with Cluster API have different update behavior than other
providers.  Nodes in a Cluster API managed cluster are never updated in place.
All updates are perfomed by instantiating a new node and removing an existing
one.  Updates are initiated from the management cluster rather than targetting
the managed cluster directly.  To stage an update for a Cluster API managed
cluster, use the kubeconfig for the management cluster and supply either the
cluster configuration file or the name of the cluster.

Unlike with other providers, staging an update for the same Kubernetes minor
version is useful.  When staging without a version, a check is performed to
determine if new boot media is available for the provider for that Kubernetes
version.  If new media is available, it is uploaded, new templates are created,
and update instructions are suggested.  That is, it behaves exactly like staging
an update to the next minor version of Kubernetes.

```
$ export KUBECONFIG=$(ocne cluster show -C management)
$ ocne cluster stage --version 1.30  -C managed
```

Note that there are cases where doing an in-place upgrade of nodes in a Cluster
API cluster is useful.  Some cluster nodes must be long lived because the host
resources that are difficult to migrate.  These can be updated in place if
necessary.

### Updating a Cluster Node

The `ocne node update` command is used to apply an available update to a cluster
node.  Nodes update by rebooting.  `ocne node update` configures the system to
reboot into the latest version and then reboots it.  When the node finishes
starting, it will perform various tasks that are necessary to complete the
update process.  The same command is used and the same process is followed to
apply incremental updates as well as updating from one Kubernetes minor version
to the next.

```
$ ocne node update --node mynode
```

## Examples

In this example, a Kubernetes cluster is updated from Kubernetes 1.29 to 1.30.

### In-Place Updates

#### Creating a Cluster

Create a small cluster with the libvirt provider.  The same process is used for
most providers.  The oci provider is an exception because the the Cluster API
update process is driven through Cluster API.

```
$ ocne cluster start -n 3 -w 2 --version 1.29
INFO[2024-08-06T16:49:07Z] Creating new Kubernetes cluster named ocne
INFO[2024-08-06T16:49:58Z] Waiting for the Kubernetes cluster to be ready: ok
INFO[2024-08-06T16:50:02Z] Installing flannel into kube-flannel: ok
INFO[2024-08-06T16:50:03Z] Installing ui into ocne-system: ok
INFO[2024-08-06T16:50:03Z] Installing app-catalog into ocne-system: ok
INFO[2024-08-06T16:50:03Z] Kubernetes cluster was created successfully
INFO[2024-08-06T16:51:04Z] Waiting for the UI to be ready: ok

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/home/opc/.kube/kubeconfig.ocne.local' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y

INFO[2024-08-06T16:51:08Z] Post install information:

To access the cluster from the VM host:
    copy /home/opc/.kube/kubeconfig.ocne.vm to that host and run kubectl there
To access the cluster from this system:
    use /home/opc/.kube/kubeconfig.ocne.local
To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system

```

#### Staging the Update

The cluster is currently running Kubernetes 1.29

```
$ export KUBECONFIG=$(ocne cluster show)

$ kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
ocne111     Ready    control-plane   31s   v1.29.3+3.el8
ocne19535   Ready    <none>          36s   v1.29.3+3.el8
ocne20042   Ready    <none>          36s   v1.29.3+3.el8
ocne227     Ready    control-plane   32s   v1.29.3+3.el8
ocne3244    Ready    control-plane   83s   v1.29.3+3.el8
```

Stage an update to Kubernetes 1.30

```
$ ocne cluster stage --version 1.30
INFO[2024-08-06T16:52:09Z] Running node stage
INFO[2024-08-06T16:52:15Z] Waiting for pod ocne-system/stage-node-ocne111-pod to be ready: ok
INFO[2024-08-06T16:52:15Z] Node ocne111 successfully staged
INFO[2024-08-06T16:52:15Z] Running node stage
INFO[2024-08-06T16:52:21Z] Waiting for pod ocne-system/stage-node-ocne19535-pod to be ready: ok
INFO[2024-08-06T16:52:22Z] Node ocne19535 successfully staged
INFO[2024-08-06T16:52:22Z] Running node stage
INFO[2024-08-06T16:52:28Z] Waiting for pod ocne-system/stage-node-ocne20042-pod to be ready: ok
INFO[2024-08-06T16:52:28Z] Node ocne20042 successfully staged
INFO[2024-08-06T16:52:28Z] Running node stage
INFO[2024-08-06T16:52:34Z] Waiting for pod ocne-system/stage-node-ocne227-pod to be ready: ok
INFO[2024-08-06T16:52:35Z] Node ocne227 successfully staged
INFO[2024-08-06T16:52:35Z] Running node stage
INFO[2024-08-06T16:52:41Z] Waiting for pod ocne-system/stage-node-ocne3244-pod to be ready: ok
INFO[2024-08-06T16:52:41Z] Node ocne3244 successfully staged
```

After some time, usually several minutes, the update will be available on
every cluster node.  The progress can be monitored by following the update
service logs on each node.  An update can be performed when the node is
annotated to indicate that an update is available.  The message can be seen
at the end of the log below.

```
$ ocne cluster console --node ocne111
sh-4.4# chroot /hostroot
sh-4.4# journalctl -fu ocne-update.service
-- Logs begin at Tue 2024-08-06 16:50:15 UTC. --
Aug 06 16:55:09 ocne111 ocne-update.sh[3804]: Fetched ostree chunk sha256:0f34e3208fce
Aug 06 16:55:09 ocne111 ocne-update.sh[3804]: Fetching ostree chunk sha256:bae8b2999145 (119.9 MB)
Aug 06 16:55:13 ocne111 ocne-update.sh[3804]: Fetched ostree chunk sha256:bae8b2999145
Aug 06 16:55:13 ocne111 ocne-update.sh[3804]: Fetching ostree chunk sha256:ad312c5c40cc (2.3 kB)
Aug 06 16:55:13 ocne111 ocne-update.sh[3804]: Fetched ostree chunk sha256:ad312c5c40cc
Aug 06 16:55:13 ocne111 ocne-update.sh[3804]: Fetching ostree chunk sha256:f91bb2a291c6 (3.3 MB)
Aug 06 16:55:14 ocne111 ocne-update.sh[3804]: Fetched ostree chunk sha256:f91bb2a291c6
Aug 06 16:55:26 ocne111 ocne-update.sh[3804]: Update downloaded.
Aug 06 16:55:27 ocne111 ocne-update.sh[4377]: node/ocne111 annotated
```

#### Updating Cluster Nodes

Once an update is available to a node, that node can be updated.  The
`ocne cluster info` command show whether or not a node can be updated.

```
$ ocne cluster info
INFO[2024-08-06T16:56:48Z] Collecting node data
Cluster Summary:
  control plane nodes: 3
  worker nodes: 2
  nodes with available updates: 5

Nodes:
  Name		Role		State	Version		Update Available
  ----		----		-----	-------		----------------
  ocne111	control plane	Ready	v1.29.3+3.el8	true
  ocne19535	worker		Ready	v1.29.3+3.el8	true
  ocne20042	worker		Ready	v1.29.3+3.el8	true
  ocne227	control plane	Ready	v1.29.3+3.el8	true
  ocne3244	control plane	Ready	v1.29.3+3.el8	true
```

When an update is available, it can be applied to a node.  When updating cluster
nodes, always update all control plane nodes first and then the worker nodes.

```
$ ocne node update --node ocne111
INFO[2024-08-06T16:57:43Z] Draining node ocne111 before updating it
INFO[2024-08-06T16:57:45Z] Draining node ocne111: ok
INFO[2024-08-06T16:57:45Z] Running node update
INFO[2024-08-06T16:57:51Z] Waiting for pod ocne-system/update-node-ocne111-pod to be ready: ok
INFO[2024-08-06T16:58:13Z] Node ocne111 has been updated and rebooted
INFO[2024-08-06T16:58:19Z] Waiting for the node ocne111 to be ready: ok
INFO[2024-08-06T16:58:20Z] Un-cordoning node ocne111: ok
INFO[2024-08-06T16:58:20Z] Node ocne111 successfully updated
```

Node updates are asynchronous.  The update is complete when the node reports the
new Kubernetes version.  It may take a few minutes.

```
$ kubectl get node
NAME        STATUS   ROLES           AGE     VERSION
ocne111     Ready    control-plane   8m9s    v1.30.3+1.el8
ocne19535   Ready    <none>          8m14s   v1.29.3+3.el8
ocne20042   Ready    <none>          8m14s   v1.29.3+3.el8
ocne227     Ready    control-plane   8m10s   v1.29.3+3.el8
ocne3244    Ready    control-plane   9m1s    v1.29.3+3.el8
```

It is possible to use the console to see the new content.  Ignore the connection
refused error.  That is due to the fact that no kubeconfig was set.

```
$ ocne cluster console --node ocne111
sh-4.4# chroot /hostroot
sh-4.4# kubectl version
Client Version: v1.30.3+1.el8
Kustomize Version: v5.0.4-0.20230601165947-6ce0bf390ce3
The connection to the server localhost:8080 was refused - did you specify the right host or port?
```

`ocne cluster info` will no longer show that an update is available for the node.

```
$ ocne cluster info
INFO[2024-08-06T17:00:06Z] Collecting node data
Cluster Summary:
  control plane nodes: 3
  worker nodes: 2
  nodes with available updates: 4

Nodes:
  Name		Role		State	Version		Update Available
  ----		----		-----	-------		----------------
  ocne111	control plane	Ready	v1.30.3+1.el8	false
  ocne19535	worker		Ready	v1.29.3+3.el8	true
  ocne20042	worker		Ready	v1.29.3+3.el8	true
  ocne227	control plane	Ready	v1.29.3+3.el8	true
  ocne3244	control plane	Ready	v1.29.3+3.el8	true
```

Update the rest of the nodes in sequence, starting with the control plane nodes.

```
$ ocne node update --node ocne227
INFO[2024-08-06T17:00:39Z] Draining node ocne227 before updating it
INFO[2024-08-06T17:00:40Z] Draining node ocne227: ok
INFO[2024-08-06T17:00:40Z] Running node update
...

$ ocne node update --node ocne3244
INFO[2024-08-06T17:02:38Z] Draining node ocne3244 before updating it
INFO[2024-08-06T17:02:46Z] Draining node ocne3244: ok
INFO[2024-08-06T17:02:46Z] Running node update
...

$ ocne node update --node ocne19535
INFO[2024-08-06T17:05:12Z] Draining node ocne19535 before updating it
INFO[2024-08-06T17:05:20Z] Draining node ocne19535: ok
INFO[2024-08-06T17:05:20Z] Running node update
...

$ ocne node update --node ocne20042 --delete-emptydir-data
INFO[2024-08-06T17:07:00Z] Draining node ocne20042 before updating it
INFO[2024-08-06T17:07:08Z] Draining node ocne20042: ok
INFO[2024-08-06T17:07:08Z] Running node update
...
```

#### Inspecting the Cluster

Once the cluster is complete, the status will report the new version for all the
cluster nodes.

```
$ kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
ocne111     Ready    control-plane   18m   v1.30.3+1.el8
ocne19535   Ready    <none>          18m   v1.30.3+1.el8
ocne20042   Ready    <none>          18m   v1.30.3+1.el8
ocne227     Ready    control-plane   18m   v1.30.3+1.el8
ocne3244    Ready    control-plane   19m   v1.30.3+1.el8
```

`ocne cluster info` shows the same.

```
$ ocne cluster info
INFO[2024-08-06T17:09:23Z] Collecting node data
Cluster Summary:
  control plane nodes: 3
  worker nodes: 2
  nodes with available updates: 0

Nodes:
  Name		Role		State	Version		Update Available
  ----		----		-----	-------		----------------
  ocne111	control plane	Ready	v1.30.3+1.el8	false
  ocne19535	worker		Ready	v1.30.3+1.el8	false
  ocne20042	worker		Ready	v1.30.3+1.el8	false
  ocne227	control plane	Ready	v1.30.3+1.el8	false
  ocne3244	control plane	Ready	v1.30.3+1.el8	false
```

#### Using Alternate Transports

The default tranport for upgrades is `ostree-unverified-registry`.  This value
is a good choice whenver there is a container mage registry available.  If a
container image registry is not available, other transports are available.
Refer to [OSTree Native Conainers URL Format](https://coreos.github.io/rpm-ostree/container/#url-format-for-ostree-native-containers) for detailed documentation
on what transports are available.  Note that `containers-storage` is not
supported.

##### Archive File

It is common to have some kind of read-only media that can easily be attached
to compute resources.  Maybe this is optical media that is inserted into a
reader.  Perhaps it's on a USB stick.  It may also be on a virtual disk that
is attached to one or more hosts.  In these cases, the most convenient way to
transport ostree images might be to leverage Open Container Initiative archive
files.

Start a cluster with an archive file configured as the ostree registry.
```
$ ocne cluster start -c <( cat << EOF
osRegistry: ostree-unverified-image:oci-archive:/var/ocne/archive.tgz
kubernetesVersion: 1.30
EOF
)
```

Stage an update
```
$ ocne cluster stage --version 1.31
```

Load the archive file onto the system.  In this case `podman save` is used for
convenience.  Any file transport is workable, including things like automounts.
```
$ podman save --format oci-archive container-registry.oracle.com/olcne/ock-ostree:1.31 | ocne cluster console --direct --node ocne-control-plane-1 -- cp /dev/stdin /var/ocne/archive.tgz
```

The node will pick up the update
```
]$ ocne cluster console --direct --node ocne-control-plane-1 -- journalctl --no-pager -u ocne-update.service
-- Logs begin at Mon 2025-03-17 19:54:34 UTC, end at Mon 2025-03-17 20:04:35 UTC. --
Mar 17 19:54:54 ocne-control-plane-1 systemd[1]: Started Update service for OCNE.
Mar 17 19:54:54 ocne-control-plane-1 ocne-update.sh[2652]: time="2025-03-17T19:54:54Z" level=fatal msg="Error parsing image name \"oci-archive:/var/ocne/archive.tgz\": creating temp directory: archive file not found: \"/var/ocne/archive.tgz\""
Mar 17 19:54:54 ocne-control-plane-1 ocne-update.sh[2626]: Could not inspect image: /var/ocne/archive.tgz
Mar 17 19:56:11 ocne-control-plane-1 systemd[1]: Stopping Update service for OCNE...
Mar 17 19:56:11 ocne-control-plane-1 systemd[1]: ocne-update.service: Succeeded.
Mar 17 19:56:11 ocne-control-plane-1 systemd[1]: Stopped Update service for OCNE.
Mar 17 19:56:11 ocne-control-plane-1 systemd[1]: Started Update service for OCNE.
Mar 17 19:56:11 ocne-control-plane-1 ocne-update.sh[4240]: time="2025-03-17T19:56:11Z" level=fatal msg="Error parsing image name \"oci-archive:/var/ocne/archive.tgz\": creating temp directory: archive file not found: \"/var/ocne/archive.tgz\""
Mar 17 19:56:11 ocne-control-plane-1 ocne-update.sh[4199]: Could not inspect image: /var/ocne/archive.tgz
Mar 17 19:58:25 ocne-control-plane-1 ocne-update.sh[4199]: Checking for new content
Mar 17 19:58:25 ocne-control-plane-1 ocne-update.sh[4199]: Image has ostree commit label: 532e26918676fde4049050d8123d881c9a86b3fdb7092662c4e29dd6cf0d928e
Mar 17 19:58:25 ocne-control-plane-1 ocne-update.sh[5402]: error: No such metadata object 532e26918676fde4049050d8123d881c9a86b3fdb7092662c4e29dd6cf0d928e.commit
Mar 17 19:58:25 ocne-control-plane-1 ocne-update.sh[4199]: Unencapsulating image
Mar 17 19:58:25 ocne-control-plane-1 ocne-update.sh[5417]: Pulling manifest: ostree-unverified-image:oci-archive:/var/ocne/archive.tgz
Mar 17 19:58:38 ocne-control-plane-1 ocne-update.sh[5417]: Importing: ostree-unverified-image:oci-archive:/var/ocne/archive.tgz (digest: sha256:4c4910d5bb0f6b3b314ccb83ba44fc3b6e49e1e5db732ca33108e96dee29091d)
...
Mar 17 19:59:23 ocne-control-plane-1 ocne-update.sh[5417]: Fetching ostree chunk sha256:69c8d90632bf (3.8 MB)
Mar 17 19:59:23 ocne-control-plane-1 ocne-update.sh[5417]: Fetched ostree chunk sha256:69c8d90632bf
Mar 17 19:59:34 ocne-control-plane-1 ocne-update.sh[5417]: Update downloaded.
Mar 17 19:59:34 ocne-control-plane-1 ocne-update.sh[5921]: node/ocne-control-plane-1 annotated

$ ocne cluster info
INFO[2025-03-17T20:05:35Z] Collecting node data                         
Cluster Summary:
  control plane nodes: 1
  worker nodes: 0
  nodes with available updates: 1

Nodes:
  Name			Role		State	Version		Update Available
  ----			----		-----	-------		----------------
  ocne-control-plane-1	control plane	Ready	v1.30.10+1.el8	true


Node: ocne-control-plane-1
  Registry and tag for ostree patch images:
    registry: /var/ocne/archive.tgz
    tag: 1.31
    transport: ostree-unverified-image:oci-archive
  Ostree deployments:
      ock e55046b74b1c525c74415f67f53d64700ab67d8b7ff63555fa82f96ea64cba68.1 (staged)
    * ock e55046b74b1c525c74415f67f53d64700ab67d8b7ff63555fa82f96ea64cba68.0
```

Update the node

```
$ ocne node update --node ocne-control-plane-1
INFO[2025-03-17T20:06:07Z] When updating control plane nodes, it is possible to lose connection to the Kubernetes API Server temporarily.  Any upcoming log messages about connection errors can be ignored. 
INFO[2025-03-17T20:06:07Z] Running node update                          
INFO[2025-03-17T20:06:13Z] Waiting for pod ocne-system/update-node-ocne-control-plane-1-pod to be ready: ok 
INFO[2025-03-17T20:06:46Z] Node ocne-control-plane-1 has been updated and rebooted 
INFO[2025-03-17T20:07:39Z] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2025-03-17T20:07:41Z] Waiting for the node ocne-control-plane-1 to be ready: ok 
INFO[2025-03-17T20:07:41Z] Un-cordoning node ocne-control-plane-1: ok 
INFO[2025-03-17T20:07:42Z] Node ocne-control-plane-1 successfully updated 

$ kubectl get node
NAME                   STATUS   ROLES           AGE   VERSION
ocne-control-plane-1   Ready    control-plane   15m   v1.31.6+1.el8
```


### Upgrading Cluster API Clusters

#### Create a Cluster

Instantiate a small cluster with the oci provider using Kubernetes 1.29.  If a
management cluster is available, it can be used.  This example uses an ephemeral
cluster for simplicity.

```
$ ocne cluster start -c mycapi.yaml
INFO[2025-02-03T22:44:07Z] Installing cert-manager into cert-manager: ok 
INFO[2025-02-03T22:44:08Z] Installing core-capi into capi-system: ok 
INFO[2025-02-03T22:44:08Z] Installing capoci into cluster-api-provider-oci-system: ok 
INFO[2025-02-03T22:44:09Z] Installing bootstrap-capi into capi-kubeadm-bootstrap-system: ok 
```

#### Stage an Update

Set the kubeconfig to the ephemeral cluster and stage an update to
Kubernetes 1.30.  A new OCI Custom Image is uploaded with the new Kubernetes
version.  Next, any existing OCIMachineTemplates that are used by other
resources are cloned and updated.  Finally, two commands are suggested.  The
first updates the cluster version and control plane nodes.  The second updates
the worker nodes.

```
$ export KUBECONFIG=$(ocne cluster show -C ocne-ephemeral)
$ ocne cluster stage --version 1.30 -c ~/tools/capi-1.29.yaml 
INFO[2025-02-03T22:52:41Z] Installing cert-manager into cert-manager: ok 
INFO[2025-02-03T22:52:41Z] Installing core-capi into capi-system: ok 
INFO[2025-02-03T22:52:42Z] Installing capoci into cluster-api-provider-oci-system: ok 
INFO[2025-02-03T22:52:42Z] Installing bootstrap-capi into capi-kubeadm-bootstrap-system: ok 
INFO[2025-02-03T22:52:43Z] Installing control-plane-capi into capi-kubeadm-control-plane-system: ok 
INFO[2025-02-03T22:52:43Z] Waiting for Core Cluster API Controllers: ok 
INFO[2025-02-03T22:52:43Z] Waiting for Kubadm Boostrap Cluster API Controllers: ok 
INFO[2025-02-03T22:52:43Z] Waiting for Kubadm Control Plane Cluster API Controllers: ok 
INFO[2025-02-03T22:52:43Z] Waiting for OCI Cluster API Controllers: ok 
INFO[2025-02-03T22:52:44Z] Preparing pod used to create image           
INFO[2025-02-03T22:52:49Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2025-02-03T22:52:49Z] Getting local boot image for architecture: amd64 
INFO[2025-02-03T22:53:25Z] Uploading boot image to pod ocne-system/ocne-image-builder: ok 
INFO[2025-02-03T22:54:45Z] Downloading boot image from pod ocne-system/ocne-image-builder: ok 
INFO[2025-02-03T22:54:45Z] New boot image was created successfully at /home/opc/.ocne/images/boot.qcow2-1.30-amd64.oci 
INFO[2025-02-03T22:55:08Z] Uploading image to object storage: ok 
INFO[2025-02-03T23:10:10Z] Importing updated image for ock: [##########]: ok 
INFO[2025-02-03T23:10:11Z] Running node stage                           
INFO[2025-02-03T23:10:16Z] Waiting for pod ocne-system/stage-node-ocne-control-plane-9xqmv-pod to be ready: ok 
INFO[2025-02-03T23:10:16Z] Node ocne-control-plane-9xqmv successfully staged 
INFO[2025-02-03T23:10:16Z] Running node stage                           
INFO[2025-02-03T23:10:22Z] Waiting for pod ocne-system/stage-node-ocne-md-0-jt8hb-sswfq-pod to be ready: ok 
INFO[2025-02-03T23:10:22Z] Node ocne-md-0-jt8hb-sswfq successfully staged 
INFO[2025-02-03T23:10:22Z] Running node stage                           
INFO[2025-02-03T23:10:28Z] Waiting for pod ocne-system/stage-node-ocne-md-0-jt8hb-vhndn-pod to be ready: ok 
INFO[2025-02-03T23:10:28Z] Node ocne-md-0-jt8hb-vhndn successfully staged 
To update KubeadmControlPlane ocne-control-plane in ocne, run:
    kubectl patch -n ocne kubeadmcontrolplane ocne-control-plane --type=json -p='[{"op":"replace","path":"/spec/version","value":"1.30.3"},{"op":"replace","path":"/spec/machineTemplate/infrastructureRef/name","value":"ocne-control-plane-1"},{"op":"add","path":"/spec/kubeadmConfigSpec/joinConfiguration/patches","value":{"directory":"/etc/ocne/ock/patches"}}]'

To update MachineDeployment ocne-md-0 in ocne, run:
    kubectl patch -n ocne machinedeployment ocne-md-0 --type=json -p='[{"op":"replace","path":"/spec/template/spec/version","value":"1.30.3"},{"op":"replace","path":"/spec/template/spec/infrastructureRef/name","value":"ocne-md-1"}]'
```

#### Update the Control Plane

Executing the first suggested command from the previous step will update the
control plane.  The cluster version is updated and then a new set of control
plane nodes are instantiated using that version.  It can take several minutes
to update the control plane based on how many control plane nodes are in the
cluster.

```
$ kubectl patch -n ocne kubeadmcontrolplane ocne-control-plane --type=json -p='[{"op":"replace","path":"/spec/version","value":"1.30.3"},{"op":"replace","path":"/spec/machineTemplate/infrastructureRef/name","value":"ocne-control-plane-1"},{"op":"add","path":"/spec/kubeadmConfigSpec/joinConfiguration/patches","value":{"directory":"/etc/ocne/ock/patches"}}]'
kubeadmcontrolplane.controlplane.cluster.x-k8s.io/ocne-control-plane patched
```

#### Wait for the Update to Complete

The control plane update will take some amount of time.  It is possible to
watch the progress of the update by inspecting the Cluster API resources.

```
# A new node is being created.  Notice the version.
$ kubectl -n ocne get machine
NAME                       CLUSTER   NODENAME                   PROVIDERID                                                                                  PHASE          AGE   VERSION
ocne-control-plane-9xqmv   ocne      ocne-control-plane-9xqmv   oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running        29m   v1.29.3
ocne-control-plane-p55sl   ocne                                                                                                                             Provisioning   23s   v1.30.3
ocne-md-0-jt8hb-sswfq      ocne      ocne-md-0-jt8hb-sswfq      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running        29m   v1.29.3
ocne-md-0-jt8hb-vhndn      ocne      ocne-md-0-jt8hb-vhndn      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running        29m   v1.29.3

# The previous node has been deleted
$ kubectl -n ocne get machine
NAME                       CLUSTER   NODENAME                   PROVIDERID                                                                                  PHASE     AGE    VERSION
ocne-control-plane-p55sl   ocne      ocne-control-plane-p55sl   oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   5m3s   v1.30.3
ocne-md-0-jt8hb-sswfq      ocne      ocne-md-0-jt8hb-sswfq      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   34m    v1.29.3
ocne-md-0-jt8hb-vhndn      ocne      ocne-md-0-jt8hb-vhndn      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   34m    v1.29.3

# The cluster nodes report the same information
$ export KUBECONFIG=$(ocne cluster show -C capi)
$ kubectl get node
NAME                       STATUS   ROLES           AGE     VERSION
ocne-control-plane-p55sl   Ready    control-plane   4m30s   v1.30.3+2.el8
ocne-md-0-jt8hb-sswfq      Ready    <none>          31m     v1.29.3+3.el8
ocne-md-0-jt8hb-vhndn      Ready    <none>          30m     v1.29.3+3.el8
```

#### Update the Worker Nodes

Once all control plane nodes are updated, it is possible to update the worker
nodes.  The overall behavior of updating worker nodes is identitical to the
control plane nodes.

```
$ export KUBECONFIG=$(ocne cluster show -C ocne-ephemeral)
$ kubectl patch -n ocne machinedeployment ocne-md-0 --type=json -p='[{"op":"replace","path":"/spec/template/spec/version","value":"1.30.3"},{"op":"replace","path":"/spec/template/spec/infrastructureRef/name","value":"ocne-md-1"}]'
machinedeployment.cluster.x-k8s.io/ocne-md-0 patched
```

#### Watch the Worker Node Update Progress

The worker node update can take a while, especially if there are a lot of
nodes.  The time it takes to update nodes depends on many factors, such as
cluster size, workloads, and more.

```
# A new node is rolling out
$ kubectl -n ocne get machine
NAME                       CLUSTER   NODENAME                   PROVIDERID                                                                                  PHASE         AGE     VERSION
ocne-control-plane-p55sl   ocne      ocne-control-plane-p55sl   oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running       8m50s   v1.30.3
ocne-md-0-9xzrv-dphhk      ocne      ocne-md-0-9xzrv-dphhk      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running       2m51s   v1.30.3
ocne-md-0-9xzrv-qsbbj      ocne                                 oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Provisioned   40s     v1.30.3
ocne-md-0-jt8hb-vhndn      ocne      ocne-md-0-jt8hb-vhndn      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running       38m     v1.29.3

# The nodes have been updated
$ kubectl -n ocne get machine
NAME                       CLUSTER   NODENAME                   PROVIDERID                                                                                  PHASE     AGE     VERSION
ocne-control-plane-p55sl   ocne      ocne-control-plane-p55sl   oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   11m     v1.30.3
ocne-md-0-9xzrv-dphhk      ocne      ocne-md-0-9xzrv-dphhk      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   5m27s   v1.30.3
ocne-md-0-9xzrv-qsbbj      ocne      ocne-md-0-9xzrv-qsbbj      oci://ocid1.instance.oc1.iad.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa   Running   3m16s   v1.30.3

# The managed cluster shows the same information
$ export KUBECONFIG=$(ocne cluster show -C capi)
$ kubectl get node
NAME                       STATUS   ROLES           AGE     VERSION
ocne-control-plane-p55sl   Ready    control-plane   10m     v1.30.3+2.el8
ocne-md-0-9xzrv-dphhk      Ready    <none>          4m43s   v1.30.3+2.el8
ocne-md-0-9xzrv-qsbbj      Ready    <none>          2m43s   v1.30.3+2.el8
```
