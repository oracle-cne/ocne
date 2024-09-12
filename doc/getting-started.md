# Getting Started

The Oracle Cloud Native Environment command line interface managed Kubernetes
clusters and the workloads that run inside of them.

## System Configuration

### Libvirt

Using the libvirt providers requires the target system to be running libvirt
and optionally requires that your user be configured to have access to the
same.

#### Oracle Linux 8

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

#### Mac

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

## Quick Start

The default cluster provisioning provider uses libvirt to create and manage
virtual machines.  The simplest way to get started is:

```
ocne cluster start
```

This will create a one-node cluster running Flannel, the Oracle Cloud Native
Environment Dashboard, and the Oracle Application Catalog.

The first execution of `ocne cluster start` isn't especially quick.  It needs
to download some resources, notably a base image for the virtual machines that
it creates.  Subsequent invocations will use the cached image, and downloading
it again is not required.

## Accessing the UI

When the cluster running and all resources have been installed, the Oracle Cloud
Native Environment UI is opened in a browser window.  When you are done poking
around, close the window and follow the prompts in the shell to terminate
`ocne`.  The UI can be opened again by forwarding the service to a local port.

This example assumes you have created a cluster named `ocne`, which is the
default when calling `ocne cluster start` with no arguments.

```
export KUBECONFIG=~/.kube/kubeconfig.ocne.local

# Once it is up and running, get an access token.
kubectl -n ocne-system create token ui

# Forward the services to your local system
kubectl --insecure-skip-tls-verify port-forward -n ocne-system service/ui 8443:443
```

Once the port forward has been established, navigate to `http://127.0.0.1:8443`
in your browser of choice.  You will be presented with an authentication screen.
Copy the token that was generated before starting the port forward and paste
it into the box.

When you're done, stop both port forwards.

## Cleaning Up

When you have finished with the cluster, delete the cluster and all of its
resources.

```
ocne cluster delete
```
