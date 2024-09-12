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
command sets various configuration options in the cluster and configures each
cluster node to upgrade to the next Kubernetes version.

```
$ ocne cluster stage --version 1.30
```

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

## Example

In this example, a Kubernetes cluster is updated from Kubernetes 1.29 to 1.30.

### Creating a Cluster

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

### Staging the Update

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

### Updating Cluster Nodes

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

### Inspecting the Cluster

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
