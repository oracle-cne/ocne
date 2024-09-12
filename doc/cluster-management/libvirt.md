# Libvirt Provider

The libvirt provider manages clusters on a single host using libvirt.  Libvirt
clusters are useful primarily for test and development.  While it is possible
to use these clusters for serious workloads, the fact that all Kubernetes nodes
are deployed to the same underlying host makes them unsuitable for workloads
that have availability requirements.

## Installing and Configuring Libvirt

Libvirt is a virtualization technology that encapsulates several types of
virtualization and abstracts away many of the details of running virtual
machines.  It is available for multiple platforms, such as Oracle Linux
and Mac OS.

### Oracle Linux 8

Install Libvirt via `dnf`.  Once this is complete it is possible to use `ocne`
as the root user.

```
dnf config-manager --enable ol8_kvm_appstream
dnf module reset virt:ol
dnf module install -y virt:kvm_utils3/common
systemctl enable --now libvirtd.service
```

If you wish to configure your user to have privileged access to libvirtd,
add the user to the `libvirt` and `qemu` groups.

```
sudo usermod -a -G libvirt,qemu $USER
```

Then log out and log back in.

### Oracle Linux 9

Install Libvirt via `dnf`.  Once this is complete it is possible to use `ocne`
as the root user.

```
dnf install -y libvirt qemu-kvm
for drv in qemu network nodedev nwfilter secret storage interface proxy; do
	systemctl start virt${drv}d{,-ro,-admin}.socket;
done
```

If you with to configure your user to have privileged access to libvirtd,
add the user to the `libvirt` and `qemu` groups.

```
sudo usermod -a -G libvirt,qemu $USER
```

### Mac

Install libvirt via `brew`.

```
brew install libvirt
brew services start libvirt
```

There are some limitations to be aware of when running a Kubernetes cluster on
a Mac.  The networking model for Macs is limited compared to Oracle Linux.  It
is not possible to create clusters with multiple nodes.  It is also not possible
to leverage common Kubernetes services like LoadBalancer services.  Accessing
in-cluster services requires using `kubectl port-forward` to tunnel those
services to your system.

## Network Requirements

Clusters created with this provider create a tunnel that allows the cluster to
be accessed through a port on the host where the cluster is deployed.  The
port range starts at 6443 and and increments from there.  As clusters are
deleted, the ports are freed.  If deploying a cluster on a remote system, make
sure a range of ports are accessible through the system firewall starting
at 6443.

## Kubeconfigs

For clusters started on systems with access to privileged libvirt instances,
two kubeconfigs are created.  One gives direct access to the true Kubernetes
API server endpoint.  The other gives access to a dedicated tunnel implemented
with SLiRP that allows access from remote systems.  If a cluster is started on
a local system, either kubeconfig can be used to access the cluster.  If the
cluster is started on a remote system, only the kubeconfig that access the
cluster via the tunnel will work.  The other kubeconfig can be used by copying
it to the libvirt host and using it there.

## Creating a Single Node Cluster on a Local System

For quick testing of Oracle Cloud Native Environment or your applications, a
single node local cluster can be created as an unprivilged user using the
libvirt provider.

```
$ ocne cluster start
INFO[2024-07-11T10:40:40-05:00] Creating new Kubernetes cluster named ocne   
INFO[2024-07-11T10:41:21-05:00] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-07-11T10:41:32-05:00] Installing flannel into kube-flannel: ok 
INFO[2024-07-11T10:41:33-05:00] Installing ui into ocne-system: ok 
INFO[2024-07-11T10:41:33-05:00] Installing app-catalog into ocne-system: ok 
INFO[2024-07-11T10:41:33-05:00] Kubernetes cluster was created successfully  
INFO[2024-07-11T10:42:24-05:00] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/Users/user/.kube/kubeconfig.ocne.local' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y

INFO[2024-07-11T10:42:27-05:00] Post install information:

To access the cluster from the VM host:
    copy /Users/user/.kube/kubeconfig.ocne.vm to that host and run kubectl there
To access the cluster from this system:
    use /Users/user/.kube/kubeconfig.ocne.local
To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system
```

## Creating Named Clusters

Multiple clusters can be created from the same host by giving unique names to
each cluster.  The default name for a cluster is "ocne".

```
$ ocne cluster start -C mycluster
```

## Creating a Cluster on a Remote System

Clusters can be created on remote system by using a libvirt connection URI.  If
the URI provides privileged access to the system, it is possible to create
complex configurations that involve multiple nodes and sophisticated networks.

```
$ ocne cluster start -C remotecluster -s qemu+ssh://user@host/system
INFO[2024-07-11T10:44:36-05:00] Creating new Kubernetes cluster named remotecluster 
INFO[2024-07-11T10:45:28-05:00] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-07-11T10:45:29-05:00] Installing flannel into kube-flannel: ok 
INFO[2024-07-11T10:45:31-05:00] Installing ui into ocne-system: ok 
INFO[2024-07-11T10:45:32-05:00] Installing app-catalog into ocne-system: ok 
INFO[2024-07-11T10:45:32-05:00] Kubernetes cluster was created successfully  
INFO[2024-07-11T10:46:13-05:00] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/Users/user/.kube/kubeconfig.remotecluster.local' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y

INFO[2024-07-11T10:46:22-05:00] Post install information:

To access the cluster from the VM host:
    copy /Users/user/.kube/kubeconfig.remotecluster.vm to that host and run kubectl there
To access the cluster from this system:
    use /Users/user/.kube/kubeconfig.remotecluster.local
To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system 
```

## Creating Clusters with Multiple Nodes

If the target libvirt instance is running on a Linux system, and the connection
URI gives privileged access, it is possible to create clusters that have
multiple nodes.  Nodes are joined to the cluster asychronously.  When the
command completes, some nodes may not yet be joined to the cluster.

```
$ ocne cluster start -C multinode -s qemu+ssh://user@host/system --control-plane-nodes 3 --worker-nodes 2
INFO[2024-07-11T10:53:14-05:00] Creating new Kubernetes cluster named multinode 
INFO[2024-07-11T10:54:06-05:00] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-07-11T10:54:13-05:00] Installing flannel into kube-flannel: ok 
INFO[2024-07-11T10:54:15-05:00] Installing ui into ocne-system: ok 
INFO[2024-07-11T10:54:16-05:00] Installing app-catalog into ocne-system: ok 
INFO[2024-07-11T10:54:16-05:00] Kubernetes cluster was created successfully  
INFO[2024-07-11T10:55:55-05:00] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/Users/user/.kube/kubeconfig.multinode.local' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y

INFO[2024-07-11T10:56:04-05:00] Post install information:

To access the cluster from the VM host:
    copy /Users/user/.kube/kubeconfig.multinode.vm to that host and run kubectl there
To access the cluster from this system:
    use /Users/user/.kube/kubeconfig.multinode.local
To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system
```

After some period of time, the cluster will contain all the nodes.

```
$ kubectl get nodes
NAME        STATUS   ROLES           AGE     VERSION
ocne1603    Ready    control-plane   87s     v1.28.3+3.el8
ocne16890   Ready    <none>          90s     v1.28.3+3.el8
ocne17586   Ready    control-plane   87s     v1.28.3+3.el8
ocne22260   Ready    <none>          90s     v1.28.3+3.el8
ocne2393    Ready    control-plane   2m20s   v1.28.3+3.el8
```

## Using a Configuration File To Start a Cluster

Configuration options for clusters can be provided with a configuration file
rather than the command line.  See [Configuration Managment](/doc/configuration/configuration.md) or `ocne-config.yaml(5)`.

```
$ cat config.yaml << EOF
name: fromconfig
workerNodes: 2
providers:
  libvirt:
    uri: qemu+ssh://user@host/system
    workerNode:
      storage: 50Gi
EOF

$ ocne cluster start -c config.yaml
...
```

Once the cluster is up, notice that the disk size of the worker nodes is
50Gi rather than 14Gi.

```
$ export KUBECONFIG='/Users/user/.kube/kubeconfig.fromconfig.local'
$ kubectl get node               
NAME        STATUS   ROLES           AGE     VERSION
ocne21532   Ready    control-plane   3m2s    v1.28.3+3.el8
ocne32559   Ready    <none>          2m30s   v1.28.3+3.el8
ocne6734    Ready    <none>          2m29s   v1.28.3+3.el8

$ echo "chroot /hostroot df -h /" | ocne cluster console --node ocne21532
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda3        14G  3.6G   10G  27% /

$ echo "chroot /hostroot df -h /" | ocne cluster console --node ocne32559
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda3        49G  4.0G   45G   9% /
```

## Destroying Clusters

Clusters and cluster resources can be deleted by name.  The name of the cluster
to delete can be specified on the command line or a configuration file.  If a
name is not specified, the cluster named "ocne" is destroyed.

```
$ ocne cluster delete -C mycluster
```
