# Bring Your Own Infrastructure

In some cases, it is necessary to deploy Oracle Cloud Native Environment in
environments that do not have any infrastructure management APIs or only
have access to APIs for which Oracle Cloud Native Environment does not have
built-in support.  There is no automatic infrastructure provisioning.  The
"Bring Your Own" Infrastructure provider is a set of tools that make it easy
to create custom boot images for various cluster roles and generate the
configuration materials required to launch hosts running those images and
participate in Kubernetes clusters.

## Prerequisites

At a high level, custom installations of OCK use an OSTree archive in
conjunction with Anaconda and Kickstart to create the bootable media.  Once
the base installation is complete, Ignition is used to complete any first-boot
configuration and provision Kubernetes services on the host.

It is necessary to have access to the following infrastructure:
* A kernel to boot
* An initrd that matches the boot kernel
* A root filesystem that can run Anaconda and Kickstart
* A means to perform an automated installation using Kickstart
* A means to serve the OCK OSTree archive
* A means to serve Ignition files

An easy way to achieve the first four bullet points is to download [https://yum.oracle.com/oracle-linux-isos.html](Oracle Linux installation media)
and following the [https://docs.oracle.com/en/operating-systems/oracle-linux/8/install/install-AutomatinganOracleLinuxInstallationbyUsingKickstart.html](automated installation guide).
Any of the Oracle Linux 8 or Oracle Linux 9 installation media is acceptable.
Only the kernel, initrd, and root filesystem from the media is used.  All
installed content comes from the OCK OSTree archive.

The Oracle Cloud Native Environment CLI can generate a container image that
serves an OSTree archive over http.  The container image can be served using
a container runtime such as Podman or inside a Kubernetes cluster.  It is also
possible to generate an OSTree archive manually and serve it over any http
server.

Ignition files can be served using any of the [https://coreos.github.io/ignition/supported-platforms/](supported Ignition providers).
It is also possible to embed the Ignition configuration file directly on to
the root filesystem of the host if the installation is done reasonably close
to when the Ignition configuration was generated.

## Cluster Lifecycle

### Creating a Cluster

Cluster creation is a two step process.  First, an cluster is initalized by
starting an initial control plane node.  Once that node it is available, it
is possible to either join new nodes or install the default cluster applications
such as the Oracle Cloud Native Environment UI, Application Catalog, and CNI.
Either approach is acceptable.

Unlike other providers, running `ocne cluster start` generates an Ignition file
that is used by the initial control plane node.  It contains the configuration
required to start a new cluster, such as credentials and other key material.
This is only true for the first invocation of the command.  Subsequent
invocations will install any applications defined by the cluster configuration
file.

### Adding New Cluster Nodes

There are two ways to add new nodes to a Kubernetes cluster: creating a new
host configured to join the cluster during the first boot, and migrating an
existing cluster node from one cluster to another.

#### Joining New Nodes

Similar to initializing a new cluster, joining a new node to a cluster with
this provider generates the materials required to join a new node to an existing
Kubernetes cluster.  It generates an Ignition file that contains all
configuration and other materials required to join an existing cluster.  In
addition, it generates any tokens and certificate upload keys required for the
new node to authentication with the cluster.

To join the new node, gather the ignition file and the token and certificate
upload commands.  Install the ignition file to an appropriate location to be
consumed by the new node.  Before booting the node, run the commands required to
create the join token and the certificate upload key (if applicable).  Once that
has been complete, boot the new node.

#### Migrating Nodes Between Clusters

It is possible to migrate nodes with similar configurations and Kubernetes
versions between clusters.  Migrating a node from one cluster to another is
most useful when it is not feasible to coordinate adding the key material
necessary to join a node to an existing cluster with the provisioning of the
new host.  In these cases, the easiest path forward is to create a single
node cluster and move the node from that cluster to the target cluster. In
this way, it is possible to stitch together several small clusters into
a single large one.

Migrating a node from one cluster to another requires the kubeconfig of the
cluster that contains the node, the kubeconfig of the target cluster, and the
name of the node to migrate.  With these three items, it is possible to remove
the node from its current cluster and join the new cluster.  Note that moving
the last control plane node in a cluster destroys that cluster.  Be sure to
migrate or otherwise remove all worker nodes first.

```
$ export NODE=mynode
$ export SOURCE_KUBECONFIG=~/.kube/kube.source
$ export TARGET_KUBECONFIG=~/.kube/kube.target
$ ocne cluster join \
	--kubeconfig "$SOURCE_KUBECONFIG" \
	--destionation "$TARGET_KUBECONFIG" \
	--node "$NODE"
```

### Removing Cluster Nodes

Nodes are manually removed from a Kubernetes cluster using typical `kubectl` and
`kubeadm` commands.  Be sure to reset the node with `kubeadm` before deleting
the node from the cluster.  If that step is skipped the node will automatically
re-register itself with the cluster and undo the node deletion if kubelet is
somehow restarted.

```
$ export NODE=mynode
$ echo "chroot /hostroot kubeadm reset" | ocne cluster console --node $NODE
$ kubectl delete node $NODE
```

## Generating the OSTree Archive Server

A container image that serves the OSTree archive over http can be generated
using the image creation tool in the Oracle Cloud Native Environment CLI.

```
$ ocne image create --type ostree
INFO[2024-06-25T17:55:07Z] Creating Image                               
INFO[2024-06-25T17:55:57Z] Preparing pod used to create image           y: ok 
INFO[2024-06-25T17:56:18Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-06-25T18:11:12Z] Generating container image: ok   
INFO[2024-06-25T18:12:20Z] Saving container image: ok       
INFO[2024-06-25T18:12:22Z] Saved image to /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar 
```

Once the image has been created, it can be run locally or uploaded to a
container registry.

To load the container image into the local image cache, use the Open Container
Initiative image loading facility built in to your container runtime or other
some other tool that performs the same task.
```
$ podman load < /home/opc/.ocne/images/ock-1.28.3-ostree.tar
```

To publish the container image to a registry, use the container image upload
tool built into the Oracle Cloud Native Environment CLI.  Container images can
be uploaded using any transport supported by libcontainer.  See
containers-transports(5) for the list of transports.  A transport is always
required.
```
$ ocne image upload --type ostree --file /home/opc/.ocne/images/ock-1.28.3-arm64-ostree.tar --destination docker://my-registry.my-organization.com/ocne/ock-ostree:1.28.3-arm64
```

## Examples

### Creating a Cluster and Joining Nodes

This example emulates a "Bring Your Own" installation using libvirt.  It is only
an example that can be used to familiarize yourself with the "Bring Your Own"
provider.  Any instructions in the example must be adjusted to fit the
environment where Oracle Cloud Native Environment is installed.

#### Gather the Installation Media

Download The Oracle Linux 8 Release 10 Boot Media.  For convenience, locate
it in the libvirt images directory so it can be access by a virtual machine
later in the process.
```
$ cd /var/lib/libvirt/images/ && wget https://yum.oracle.com/ISOS/OracleLinux/OL8/u10/x86_64/OracleLinux-R8-U10-x86_64-boot-uek.iso
```

Mount the media via loopback so that the kernel and initrd can be accessed
```
$ mkdir -p /mnt/ol-boot
$ export LOOPBACK_DEVICE=$(losetup -f)
$ losetup "$LOOPBACK_DEVICE" /var/lib/libvirt/images/OracleLinux-R8-U10-x86_64-boot-uek.iso
$ mount "$LOOPBACK_DEVICE" /mnt/ol-boot
```

Create a Kickstart file that defines an automated installation using a local
OSTree archive server.  The server is started at a later step.
```
$ export OSTREE_IP=$(ip route |  grep 'default.*src' | cut -d' ' -f9)
$ export OSTREE_PORT=8080
$ export OSTREE_REF=ock
$ export OSTREE_PATH=ostree
$ mkdir ks
$ envsubst > ks/ks.cfg << EOF
logging

keyboard us
lang en_US.UTF-8
timezone UTC
text
reboot

selinux --permissive
firewall --use-system-defaults
network --bootproto=dhcp

zerombr
clearpart --all --initlabel
part /boot --fstype=xfs --label=boot --size=1024
part /boot/efi --fstype=efi --label=efi --size=512
part / --fstype=xfs --label=root --grow

user --name=ocne --groups=wheel --password=welcome

services --enabled=ostree-remount

bootloader --append "rw ip=dhcp rd.neednet=1 ignition.platform.id=metal ignition.config.url=http://$OSTREE_IP:$OSTREE_PORT/ks/ignition.ign ignition.firstboot=1"

ostreesetup --nogpg --osname ock --url http://$OSTREE_IP:$OSTREE_PORT/$OSTREE_PATH --ref $OSTREE_REF

%post

%end
EOF
```

#### Generating and Serving an OSTree Archive

Generate an OSTree archive container image and load it into the local cache.
```
$ ocne image create --type ostree
INFO[2024-06-25T17:55:07Z] Creating Image                               
INFO[2024-06-25T17:55:57Z] Preparing pod used to create image           y: ok 
INFO[2024-06-25T17:56:18Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-06-25T18:11:12Z] Generating container image: ok   
INFO[2024-06-25T18:12:20Z] Saving container image: ok       
INFO[2024-06-25T18:12:22Z] Saved image to /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar 
...
$ podman load < /home/opc/.ocne/images/ock-1.28.3-ostree.tar
```

Start a container using the image, and include the kickstart configuration from
a previous step.
```
$ podman run -d --name ock-content-server -p $OSTREE_PORT:80 -v `pwd`/ks:/usr/share/nginx/html/ks localhost/ock-ostree:latest
```

#### Create an Installation Virtual Machine

Define a virtual machine that generates a golden image using the installation
media, OSTree archive, and kickstart file from the previous steps
```
$ qemu-img create -f qcow2 /var/lib/libvirt/images/install.qcow2 15G
$ virsh pool-refresh images

$ envsubst > domain.xml << EOF
<domain type='kvm' id='1' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>ocne-install</name>
  <memory unit='KiB'>4194304</memory>
  <currentMemory unit='KiB'>4194304</currentMemory>
  <vcpu placement='static'>2</vcpu>
  <resource>
    <partition>/machine</partition>
  </resource>
  <os>
    <type arch='x86_64' machine='pc-q35-7.2'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.pure-efi.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS.pure-efi.fd'>/var/lib/libvirt/qemu/nvram/boot-vm_VARS.fd</nvram>
    <kernel>/mnt/ol-boot/images/pxeboot/vmlinuz</kernel>
    <initrd>/mnt/ol-boot/images/pxeboot/initrd.img</initrd>
    <cmdline>inst.stage2=hd:LABEL=OL-8-10-0-BaseOS-x86_64 quiet console=ttyS0 inst.ks=http://${OSTREE_IP}:${OSTREE_PORT}/ks/ks.cfg</cmdline>
  </os>
  <features>
    <acpi/>
    <apic/>
    <smm state='off'/>
  </features>
  <cpu mode='host-passthrough' check='none' migratable='on'>
    <feature policy='disable' name='pdpe1gb'/>
  </cpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>destroy</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='file' device='cdrom'>
	    <driver name='qemu' type='raw' cache='none'/>
	    <source file='/var/lib/libvirt/images/OracleLinux-R8-U10-x86_64-boot-uek.iso'/>
	    <target dev='sdb' bus='sata'/>
	    <readonly/>
	    <address type='drive' controller='0' bus='0' target='0' unit='1'/>
    </disk>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='images' volume='install.qcow2' index='1'/>
      <target dev='sda' bus='scsi'/>
      <alias name='scsi0-0-0-0'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'>
      <alias name='scsi0'/>
      <address type='pci' domain='0x0000' bus='0x03' slot='0x00' function='0x0'/>
    </controller>
    <controller type='sata' index='0'>
      <alias name='ide'/>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x1f' function='0x2'/>
    </controller>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <audio id='1' type='none'/>
    <memballoon model='virtio'>
      <alias name='balloon0'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </memballoon>
    <rng model='virtio'>
      <backend model='random'>/dev/urandom</backend>
      <alias name='rng0'/>
      <address type='pci' domain='0x0000' bus='0x04' slot='0x00' function='0x0'/>
    </rng>
  </devices>
  <seclabel type='dynamic' model='selinux' relabel='yes'>
    <label>system_u:system_r:svirt_t:s0:c334,c524</label>
    <imagelabel>system_u:object_r:svirt_image_t:s0:c334,c524</imagelabel>
  </seclabel>
  <seclabel type='dynamic' model='dac' relabel='yes'>
    <label>+107:+107</label>
    <imagelabel>+107:+107</imagelabel>
  </seclabel>
</domain>
EOF

$ virsh define domain.xml
$ virsh start ocne-install
```

Wait for the virtual machine to terminate.  The easiest way to do this
is observe the installation process using the console.
```
$ virsh console ocne-install
```

Destroy and undefine the virtual machine.  Destroying the virtual machine is
almost certain to fail, but it is good practice to do so.
```
$ virsh destroy ocne-install
$ virsh undefine --nvram ocne-install
```

#### Create a Cluster Definition

Generate a cluster configuration file that defines the Kubernetes cluster to
create.  In this example, a virtual IP of 192.168.124.230 is used.  It is
necessary to adjust this value to match your environment.  In some cases
copy-pasting this example will work.  In other cases it must be modified.
The virtual IP must be an unused IP address of the libvirt network that the
Kubernetes node virtual machines will be attached to.
```
$ cat >  byo.yaml << EOF
provider: byo
name: byocluster
virtualIp: 192.168.124.230
providers:
  byo:
    networkInterface: enp1s0
EOF
```

#### Create a Kubernetes Cluster

The "Bring Your Own" provider does not provision any infrastructure resources.
Unlike other providers that create Kubernetes cluster nodes automatically,
the BYO provider emits the ignition configuration that applies to the
node type and cluster configuration.

Generate the ignition that starts the first control plane node
```
$ ocne cluster start -c byo.yaml > cluster-init.ign
```

The result can be inspected with `jq`
```
$ jq . < cluster-init.ign
{
  "ignition": {
    "config": {
      "replace": {
        "verification": {}
      }
    },
    "proxy": {},
    "security": {
      "tls": {}
...
```

#### Start the First Control Plane Node

The Ignition configuration needs to be available to all hosts during their
first boot.  In this example, the file is exposed via the same OSTree server
that was serving the kickstart file.  It is not good practice to do this.
Ignition files often contain secrets that need to be protected.  Exposing
them over plain http and without authentication is not a great idea.  However,
this is just an example.  Ignition files should be served in a way that
protects them from any threat surface in the corresponding environment.

Expose the ignition file over http.
```
$ cp cluster-init.ign ks/ignition.ign
```

Create a virtual machine that is configured to use that ignition file, and
then start that machine.
```
$ qemu-img create -f qcow2 -F qcow2 -b /var/lib/libvirt/images/install.qcow2 /var/lib/libvirt/images/control-plane.qcow2
$ virsh pool-refresh images

$ cat > controlplane.xml << EOF
<domain type='kvm' id='1' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>control-plane</name>
  <memory unit='KiB'>4194304</memory>
  <currentMemory unit='KiB'>4194304</currentMemory>
  <vcpu placement='static'>2</vcpu>
  <resource>
    <partition>/machine</partition>
  </resource>
  <os>
    <type arch='x86_64' machine='pc-q35-7.2'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.pure-efi.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS.pure-efi.fd'>/var/lib/libvirt/qemu/nvram/control-plane.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
    <smm state='off'/>
  </features>
  <cpu mode='host-passthrough' check='none' migratable='on'>
    <feature policy='disable' name='pdpe1gb'/>
  </cpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='images' volume='control-plane.qcow2' index='1'/>
      <target dev='sda' bus='scsi'/>
      <alias name='scsi0-0-0-0'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'>
      <alias name='scsi0'/>
      <address type='pci' domain='0x0000' bus='0x03' slot='0x00' function='0x0'/>
    </controller>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <audio id='1' type='none'/>
    <memballoon model='virtio'>
      <alias name='balloon0'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </memballoon>
    <rng model='virtio'>
      <backend model='random'>/dev/urandom</backend>
      <alias name='rng0'/>
      <address type='pci' domain='0x0000' bus='0x04' slot='0x00' function='0x0'/>
    </rng>
  </devices>
  <seclabel type='dynamic' model='selinux' relabel='yes'>
    <label>system_u:system_r:svirt_t:s0:c334,c524</label>
    <imagelabel>system_u:object_r:svirt_image_t:s0:c334,c524</imagelabel>
  </seclabel>
  <seclabel type='dynamic' model='dac' relabel='yes'>
    <label>+107:+107</label>
    <imagelabel>+107:+107</imagelabel>
  </seclabel>
</domain>
EOF

$ virsh define controlplane.xml
$ virsh start control-plane
```

Re-run the Oracle Cloud Native Environment CLI cluster start command to
install any configured software into the cluster.
```
$ ocne cluster start -c byo.yaml
INFO[2024-06-26T14:55:37Z] Installing flannel into kube-flannel: ok 
INFO[2024-06-26T14:55:45Z] Installing ui into ocne-system: ok 
INFO[2024-06-26T14:55:45Z] Installing app-catalog into ocne-system: ok 
INFO[2024-06-26T14:55:45Z] Kubernetes cluster was created successfully
...
```

#### Join a Worker Node

Generate the ignition that joins a node to the cluster, and expose it over http.
Overwrite the original configuration to keep things simple.
```
$ ocne cluster join -w 1 -c byo.yaml > cluster-join.ign
Run this command before booting the new node to allow it to join the cluster: kubeadm token create n7fdiv.omaikcs0qvt7vnyo
$ cp cluster-join.ign ks/ignition.ign
```

Create the bootstrap token that the node expects.  This is dynamically created
by the cluster join command.  Copy-pasting the next commands out of this
document will almost surely fail.  Use the versions that are printed from the
cluster join command in the previous step.
```
$ export KUBECONFIG=$(ocne cluster show -C byocluster)
$ kubeadm token create n7fdiv.omaikcs0qvt7vnyo
n7fdiv.omaikcs0qvt7vnyo
```

Create a virtual machine that is configured to use the ignition file and then
start that machine.
```
$ qemu-img create -f qcow2 -F qcow2 -b /var/lib/libvirt/images/install.qcow2 /var/lib/libvirt/images/worker.qcow2
$ virsh pool-refresh images

$ cat > worker.xml << EOF
<domain type='kvm' id='1' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>worker</name>
  <memory unit='KiB'>4194304</memory>
  <currentMemory unit='KiB'>4194304</currentMemory>
  <vcpu placement='static'>2</vcpu>
  <resource>
    <partition>/machine</partition>
  </resource>
  <os>
    <type arch='x86_64' machine='pc-q35-7.2'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.pure-efi.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS.pure-efi.fd'>/var/lib/libvirt/qemu/nvram/worker-vm_VARS.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
    <smm state='off'/>
  </features>
  <cpu mode='host-passthrough' check='none' migratable='on'>
    <feature policy='disable' name='pdpe1gb'/>
  </cpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='images' volume='worker.qcow2' index='1'/>
      <target dev='sda' bus='scsi'/>
      <alias name='scsi0-0-0-0'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'>
      <alias name='scsi0'/>
      <address type='pci' domain='0x0000' bus='0x03' slot='0x00' function='0x0'/>
    </controller>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <audio id='1' type='none'/>
    <memballoon model='virtio'>
      <alias name='balloon0'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </memballoon>
    <rng model='virtio'>
      <backend model='random'>/dev/urandom</backend>
      <alias name='rng0'/>
      <address type='pci' domain='0x0000' bus='0x04' slot='0x00' function='0x0'/>
    </rng>
  </devices>
  <seclabel type='dynamic' model='selinux' relabel='yes'>
    <label>system_u:system_r:svirt_t:s0:c334,c524</label>
    <imagelabel>system_u:object_r:svirt_image_t:s0:c334,c524</imagelabel>
  </seclabel>
  <seclabel type='dynamic' model='dac' relabel='yes'>
    <label>+107:+107</label>
    <imagelabel>+107:+107</imagelabel>
  </seclabel>
</domain>
EOF

$ virsh define worker.xml
$ virsh start worker
```

Eventually the node will join the cluster.
```
$ kubectl get nodes
NAME        STATUS   ROLES           AGE     VERSION
ocne11395   Ready    <none>          22s     v1.28.3+3.el8
ocne27491   Ready    control-plane   4m24s   v1.28.3+3.el8
```

#### Join a Control Plane Node

Generate the ignition that joins a control plane node to the cluster, and expose
it over http.  Overwrite theh original ignition to keep things simple.
```
$ ocne cluster join -n 1 -c byo.yaml > cluster-join.ign
Run these commands before booting the new node to allow it to join the cluster:
	echo "chroot /hostroot kubeadm init phase upload-certs --certificate-key 573049f928d66c0f11cd90384d34d2d1717587b54525cdb6a33e11a9a464e2d5 --upload-certs" | ./out/linux_amd64/ocne cluster console --node ocne14040
	kubeadm token create anf2rr.g1wi5kyclx2rqf7c
$ cp cluster-join.ign ks/ignition.ign
```

Create the bootstrap materials within the Kubernetes cluster.  In thise case
there are two things that need to be created: a join token and an encrypted
certificate bundle.  These are dynamically created by the cluster join command.
Copy-pasting the next commands out of this document will almost certainly fail.
Use the versions that are printed from the cluster join command in the previous
step.
```
$ echo "chroot /hostroot kubeadm init phase upload-certs --certificate-key 573049f928d66c0f11cd90384d34d2d1717587b54525cdb6a33e11a9a464e2d5 --upload-certs" | ./out/linux_amd64/ocne cluster console --node ocne14040
W0626 05:08:06.295700   20074 version.go:104] could not fetch a Kubernetes version from the internet: unable to get URL "https://dl.k8s.io/release/stable-1.txt": Get "https://dl.k8s.io/release/stable-1.txt": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
W0626 05:08:06.295944   20074 version.go:105] falling back to the local client version: v1.28.3
[upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
[upload-certs] Using certificate key:
573049f928d66c0f11cd90384d34d2d1717587b54525cdb6a33e11a9a464e2d5

$ kubeadm token create anf2rr.g1wi5kyclx2rqf7c
anf2rr.g1wi5kyclx2rqf7c
```

Create a virtual machine that is configured to use the ignition file and then
start the machine.
```
$ qemu-img create -f qcow2 -F qcow2 -b /var/lib/libvirt/images/install.qcow2 /var/lib/libvirt/images/control-plane2.qcow2
$ virsh pool-refresh images

$ cat > controlplane2.xml << EOF
<domain type='kvm' id='1' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>control-plane2</name>
  <memory unit='KiB'>4194304</memory>
  <currentMemory unit='KiB'>4194304</currentMemory>
  <vcpu placement='static'>2</vcpu>
  <resource>
    <partition>/machine</partition>
  </resource>
  <os>
    <type arch='x86_64' machine='pc-q35-7.2'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.pure-efi.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS.pure-efi.fd'>/var/lib/libvirt/qemu/nvram/control-plane2.fd</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
    <smm state='off'/>
  </features>
  <cpu mode='host-passthrough' check='none' migratable='on'>
    <feature policy='disable' name='pdpe1gb'/>
  </cpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>/usr/libexec/qemu-kvm</emulator>
    <disk type='volume' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source pool='images' volume='control-plane2.qcow2' index='1'/>
      <target dev='sda' bus='scsi'/>
      <alias name='scsi0-0-0-0'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'>
      <alias name='scsi0'/>
      <address type='pci' domain='0x0000' bus='0x03' slot='0x00' function='0x0'/>
    </controller>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x01' slot='0x00' function='0x0'/>
    </interface>
    <serial type='pty'>
      <target port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <audio id='1' type='none'/>
    <memballoon model='virtio'>
      <alias name='balloon0'/>
      <address type='pci' domain='0x0000' bus='0x05' slot='0x00' function='0x0'/>
    </memballoon>
    <rng model='virtio'>
      <backend model='random'>/dev/urandom</backend>
      <alias name='rng0'/>
      <address type='pci' domain='0x0000' bus='0x04' slot='0x00' function='0x0'/>
    </rng>
  </devices>
  <seclabel type='dynamic' model='selinux' relabel='yes'>
    <label>system_u:system_r:svirt_t:s0:c334,c524</label>
    <imagelabel>system_u:object_r:svirt_image_t:s0:c334,c524</imagelabel>
  </seclabel>
  <seclabel type='dynamic' model='dac' relabel='yes'>
    <label>+107:+107</label>
    <imagelabel>+107:+107</imagelabel>
  </seclabel>
</domain>
EOF

$ virsh define controlplane2.xml
$ virsh start control-plane2
```

Eventually the node will join the cluster.  While control plane nodes
are joining the cluster, there may be periodic errors reported by kubectl
as various control plane components adapt to the new node.  Under normal
circumstances, these errors should stop after a few seconds.
```
$ kubectl get node
NAME        STATUS   ROLES           AGE     VERSION
ocne11395   Ready    <none>          3m30s   v1.28.3+3.el8
ocne12558   Ready    control-plane   18s     v1.28.3+3.el8
ocne27491   Ready    control-plane   7m32s   v1.28.3+3.el8
```

### Migrating Cluster Nodes



#### Provisioning Clusters

Migrating a node from one Kubernetes cluster to another requires two clusters
as well as a node to move.  In this example, the libvirt provider is used to
quickly create two clusters.  This is done for efficiency and is it not
recommended to actually do this.

```
$ ocne cluster start -C source
INFO[2024-08-02T17:30:19Z] Creating new Kubernetes cluster named source 
INFO[2024-08-02T17:31:11Z] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-08-02T17:31:11Z] Installing flannel into kube-flannel: ok 
INFO[2024-08-02T17:31:12Z] Installing ui into ocne-system: ok
...

$ ocne cluster start -C target
INFO[2024-08-02T17:31:33Z] Creating new Kubernetes cluster named target 
INFO[2024-08-02T17:32:24Z] Waiting for the Kubernetes cluster to be ready: ok 
INFO[2024-08-02T17:32:24Z] Installing flannel into kube-flannel: ok 
INFO[2024-08-02T17:32:25Z] Installing ui into ocne-system: ok 
INFO[2024-08-02T17:32:25Z] Installing app-catalog into ocne-system: ok
...
```

#### Migrating a Node

With all the requirements met, pick a node and migrate it.  In this case the
only node in the source cluster is moved to the target cluster.  The source
cluster is implicitly destroyed in the process.

```
$ export SOURCE_KUBECONFIG=$(ocne cluster show -C source)
$ export TARGET_KUBECONFIG=$(ocne cluster show -C target)
$ KUBECONFIG="$SOURCE_KUBECONFIG" kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
ocne10758   Ready    control-plane   13m   v1.29.3+3.el8

$ ocne cluster join --kubeconfig "$SOURCE_KUBECONFIG" --destination "$TARGET_KUBECONFIG" --node ocne10758

# Wait some amount of time

$ KUBECONFIG="$SOURCE_KUBECONFIG" kubectl get nodes
Unable to connect to the server: EOF

KUBECONFIG="$TARGET_KUBECONFIG" kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
ocne24327   Ready    <none>          15s   v1.29.3+3.el8
ocne7402    Ready    control-plane   14m   v1.29.3+3.el8
```
