# Custom Oracle Container Host for Kubernetes Installations

If the standard Oracle Continer Host for Kubernetes (OCK) images do not
fit the requirements, it is possible to create customized media using
[ock-forge](https://github.com/oracle-cne/ock-forge).  The output of this tool
can be fed into other components of the Oracle Cloud Native Environment and
its tools in order to allow complete control over the Oracle Cloud Native
Environment experience.

> **_NOTE:_** This guide makes use of container image registries.  The
> example registry "myregistry.com" is used.  Replace this value with
> something appropriate for your environment.

## Creating Custom Media

There are many reasons to create custom installation media.  Perhaps there are
additional packages that need to be installed onto OCK for a particular
environment, or maybe there are complex configuration requirements that are too
challenging to manage with Ignition.  Whatever the case, `ock-forge` can be used
to generate customized installation media and operating system content.

The purpose of this guide is to show the end to end workflow from creating
custom installation media to starting a cluster.  It is not a discussion of
what those customizations can or should be.  To that end, this example will
build media using the default OCK configuration but use those standard artifacts
to illustrate the process.

`ock-forge` produces two primary outputs.  One is boot media installed to some
disk shaped object.  That object can be a phsyical disk, an ISO, a virtual disk,
or similar.  The other output is an "ostree native container".  An ostree native
container is little more than a container image that has the complete contents
of an ostree commit as though it were checked out or used as the root filesystem
for a deployed system.  The boot media is useful in that it can be used directly
to boot systems.  The ostree native container image is used to deliver updates
to OCK hosts as well as act as the seed media for other ostree interactions.

Build an OCK 1.30 image for Kubernetes 1.30 like so:
```
$ sudo modprobe nbd
$ git clone https://github.com/oracle-cne/ock-forge.git
$ cd ock-forge
$ ./ock-forge -d /dev/nbd0 -D out/1.30/boot.qcow2 -i myregistry.com/ocne/ock-ostree:1.30 -O ./out/1.30/archive.tar -C ./ock -c configs/config-1.30 -P -s https://github.com/oracle-cne/ock.git
```

The first execution of `ock-forge` compiles some tools and builds some container
images.  It takes a while.

## Using These Media

### The Disk - out/1.30/boot.qcow2

In this case, `ock-forge` created a `qcow2` formatted disk image that is
suitable for the `libvirt` provider.  It is identical to the media delivered
via `container-registry.oracle.com/olcne/ock`.  The same content is used
during `ocne image create --type oci`, which converts a `libvirt` provider
image into an `oci` provider image.

For quick tests with the `libvirt` provider, the disk can be copied into
the volume pool used by the provider.  In a default Oracle Linux 8 or 9
installation, that will be located at `/var/lib/libvirt/images`.  To use
this image as boot media without jumping through other hoops, it can simply be
copied to `/var/lib/libvirt/images/boot.qcow2-1.30`, where the suffix `- 1.30`
is the Kubernetes version.  This is only useful for quick tests.

A better use of this media is to bundle it into a container image and push it
to a registry.  That way it can be consumed by other Oracle Cloud Native
Environment tools.  The image can be bunded by placing it into an otherwise
empty container image at the path `/disk/boot.qcow2`.

```
$ podman build -t myregistry.com/ocne/ock:1.30 -f <(cat << EOF
FROM scratch
ADD --chown=107:107 boot.qcow2 /disk/boot.qcow2
EOF
) ./out/1.30

$ podman push myregistry/ocne/ock:1.30
```

This container image can then be pushed to a container registry and used by
the `ocne` CLI to do work like create other images and start clusters.  To
use this image with the `libvirt` or `oci` providers, set
`bootVolumeContainerImage: myregistry.com/ocne/ock` in your defaults file or
cluster configuration.

### The OSTree Native Container Image - out/1.30/archive.tar and myregistry.com/ocne/ock-ostree:1.30

This file is an Open Container Initiative container image archive that contains
the ostree native container image with the same bits that are installed on the
disk media.  Running instances of OCK consume these container images to gather
updates.  It is also used as the seed for `ocne image create --type ostree`.
When `ock-forge` completes, the archive is automatically loaded into your local
containers storage cache with the same registry:tag that was provided to `ock-forge`.

The integration of ostree and containers is relatively recent.  Traditionally,
ostree commits were pulled from an HTTP(S) server known as an "archive".  The
archive server be kept up to date with the latest filesystem content and
deployed systems would pull updates from them.  As a result, many of the tools
that interact with ostree data are designed to use archive servers rathar than
container images.  To address this need, `ocne` has the `ocne image create --type ostree`
subcommand.  That command takes an ostree native container image and converts
it into another container image that serves an ostree archive over https via
nginx.  This archive server container can then be leveraged by Anaconda/Kickstart
to perform automated installations of OCK onto any systemd where Oracle Linux 8
is supported.

There are a couple ways to use the archive tarball and local container image.
Here are some options
```
# Push the image
$ podman push myregistry.com/ocne/ock-ostree:1.30

# Copy the image with skopeo
$ skopeo copy containers-storage:myregistry.com/ocne/ock-ostree:1.30 docker://myregistry.com/ocne/ock-ostree:1.30

# Copy the archive with skopeop
$ skopeo copy oci-archive:out/1.30/archive.tar docker://myregistry.com/ocne/ock-ostree:1.30
```

Once the image is pushed to a registry, it can be used as a source of updates
for OCK hosts.  It can also be used as the source of data for `ocne image create --type ostree`.
To use this image with `ocne`, set `osRegistry: myregistry.com/ocne/ock-ostree`
your defaults file or cluster configuration.

## Example with the BYO Provider

There are only a few extra steps involved to go from a fully custom OCK image to
a BYO cluster.  In this example, a single node cluster is started using a base
boot media generated using `ock-forge` and Anaconda/Kickstart.  If you are
familiar with the `byo` provider guide, many of these steps will be familiar.

The overall flow goes:
- Build OCK
- Push the ostree native container image to a registry
- Use that image to generate an ostree archive server
- Use the archive server to do a kickstart install
- Use the kickstart install disk to make a cluster

### Generate the OSTree Media

Run `ock-forge` and build an OCK image
```
$ sudo modprobe nbd
$ git clone https://github.com/oracle-cne/ock-forge.git
$ cd ock-forge
$ ./ock-forge -d /dev/nbd0 -D out/1.30/boot.qcow2 -i myregistry.com/ocne/ock-ostree:1.30 -O ./out/1.30/archive.tar -C ./ock -c configs/config-1.30 -P -s https://github.com/oracle-cne/ock.git
+ [[ -z '' ]]
+ [[ -z '' ]]
+ IGNITION_PROVIDER=qemu
+ [[ -n out/1.30/boot.qcow2 ]]
++ realpath -m out/1.30/boot.qcow2
+ DISK=/root/git/ock-forge/out/1.30/boot.qcow2
+ [[ -n ./ock ]]
++ realpath -m ./ock
+ CONFIGS_DIR=/root/git/ock-forge/ock
+ [[ -n out/1.30/archive.tar ]]
++ realpath -m out/1.30/archive.tar
+ OSTREE_IMAGE_PATH=/root/git/ock-forge/out/1.30/archive.tar
+ [[ -n '' ]]
+ [[ -n '' ]]
+ pushd /root/git/ock-forge
~/git/ock-forge ~/git/ock-forge
+++ mktemp -d
++ realpath /tmp/tmp.MSWozYSp9q
+ TMPDIR=/tmp/tmp.MSWozYSp9q
+ OSTREE_IMAGE_DIR=
...
```

For the purpose of this example, the only thing we care about is the ostree
native container image.  The qcow2 image can be ignored.

### Push the Image

Once the build completes, push the image to the container registry.  Don't
get intimidated by the apparent size of the image.  It's not actually that big.
The ostree native container image will typically have around 100 layers.  Don't
let that surprise you, either.

```
# There it is
$ podman images
REPOSITORY                                              TAG         IMAGE ID      CREATED         SIZE
myregistry.com/ocne/ock-ostree                         1.30        c22c2eb46f09  35 minutes ago  10.9 GB

# Push it
$ podman push myregistry.com/ocne/ock-ostree:1.30
...
Copying blob 1136dccb4440 done   |
Copying blob bd4618c92f4c done   |
Copying blob 7b0cd540f2fc done   |
Copying blob a057aa73f4eb done   |
Copying blob 12787d84fa13 done   |
Copying config c22c2eb46f done   |
Writing manifest to image destination
```

### Configure your Environment

Edit the `~/.ocne/defaults.yaml` to point to this new image.  That way `ocne`
will use it as the source for both OCK host updates and the ostree archive
container image.

```
$ vi ~/.ocne/defaults
...
osRegistry: myregistry.com/ocne/ock-ostree
...
```

### Create the OSTree Archive Server

Now that the CLI is configured to use the custom ostree native container image,
`ocne image create --type ostree` will use that content when generating the
ostree archive container image.

```
$ ocne image create --type ostree --version 1.30 --arch amd64
INFO[2025-01-30T16:34:17Z] Preparing pod used to create image
INFO[2025-01-30T16:34:23Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok
INFO[2025-01-30T16:39:39Z] Generating container image: ok
INFO[2025-01-30T16:41:10Z] Saving container image: ok
INFO[2025-01-30T16:41:10Z] Saved image to /home/opc/.ocne/images/ock-1.30-amd64-ostree.tar
```

### Load the OSTree Archive Server

Load the ostree archive server to the local container storage cache so it can
be used to start containers.
```
$ podman load < ~/.ocne/images/ock-1.30-amd64-ostree.tar
```

### Start Building the Environment

At this point, our journey meets up with the example in the `byo` provider guide.

### Creating a Common Boot Volume

The easiest way to build multi-node clusters with virtualization is to create
a common base image and clone that for each virtual machine.  Do that so
it is easy to join new cluster nodes later on.

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

selinux --enforcing
firewall --use-system-defaults
network --bootproto=dhcp

zerombr
clearpart --all --initlabel
part /boot --fstype=xfs --label=boot --size=1024
part /boot/efi --fstype=efi --label=efi --size=512
part / --fstype=xfs --label=root --grow

services --enabled=ostree-remount

bootloader --append "rw ip=dhcp rd.neednet=1 ignition.platform.id=metal ignition.config.url=http://$OSTREE_IP:$OSTREE_PORT/ks/ignition.ign ignition.firstboot=1"

ostreesetup --nogpg --osname ock --url http://$OSTREE_IP:$OSTREE_PORT/$OSTREE_PATH --ref $OSTREE_REF

%post

%end
EOF
```

#### Serving an OSTree Archive

Start a container using the container image generated earlier in this example,
configured to serve the kickstart media above.
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
  <os firmware='efi'>
    <type arch='x86_64' machine='q35'>hvm</type>
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

### Start the Cluster

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
  <os firmware='efi'>
    <type arch='x86_64' machine='q35'>hvm</type>
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
INFO[2025-01-30T17:47:13Z] Installing flannel into kube-flannel: ok 
INFO[2025-01-30T17:47:13Z] Installing ui into ocne-system: ok 
INFO[2025-01-30T17:47:14Z] Installing ocne-catalog into ocne-system: ok 
INFO[2025-01-30T17:47:14Z] Kubernetes cluster was created successfully  
INFO[2025-01-30T17:47:34Z] Waiting for the UI to be ready: ok 

Run the following command to create an authentication token to access the UI:
    KUBECONFIG='/home/opc/.kube/kubeconfig.byocluster' kubectl create token ui -n ocne-system
Browser window opened, enter 'y' when ready to exit: y

INFO[2025-01-30T17:47:56Z] Post install information:

To access the cluster:
    use /home/opc/.kube/kubeconfig.byocluster
To access the UI, first do kubectl port-forward to allow the browser to access the UI.
Run the following command, then access the UI from the browser using via https://localhost:8443
    kubectl port-forward -n ocne-system service/ui 8443:443
Run the following command to create an authentication token to access the UI:
    kubectl create token ui -n ocne-system
```

And there is your cluster!

From here, your cluster nodes can be kept up to date continuously by
rerunning `ock-forge` when there is an update required and pushing the
new ostree native container image to the registry.  The cluster nodes will
automatically pull in any updates and mark themselves updatable.
