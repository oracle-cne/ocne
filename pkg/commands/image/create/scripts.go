// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

const (
	setProviderScriptName = "set-provider.sh"
	modifyImageScriptName = "modify-image.sh"

	// The setProvider script changes the ignition provider type inside the boot qcow2 image.
	// The script runs both before, during, and after chroot.  The modprobe needs chroot to work
	// and if qemu-nbd is not run in chroot then it cannot find the nbd device.
	// The script first copies files to /hostroot so that they can be accessed after
	// chroot is done.  The modify_image.sh script runs from chroot and does the modifications then
	// exits chroot.  Finally, the script copies the modified boot.qcow2 from /hostroot to /tmp
	// so that it can be downloaded by the client
	setProviderScript = `#! /bin/bash
set -e

HOST_BUILD_DIR="/tmp/ocne-image-build"
HOSTROOT_BUILD_DIR="/hostroot/tmp/ocne-image-build"
MODIFY_IMAGE_SCRIPT=modify-image.sh

# copy scripts and image to /hostroot so that they can be accessed once we do chroot
# chroot is needed or else modprobe won't work (it cannot find libraries)
# note that once we chroot the /tmp is different from the /tmp that is seen from the pod root
mkdir -p $HOSTROOT_BUILD_DIR
cp /ocne-image-build/..data/* $HOSTROOT_BUILD_DIR
cp /tmp/boot.qcow2 $HOSTROOT_BUILD_DIR 

# chroot and run the script to modify the image
chroot /hostroot $HOST_BUILD_DIR/modify-image.sh

# copy the modified image back to /tmp so the client can download it
cp $HOSTROOT_BUILD_DIR/boot.qcow2 /tmp/boot.qcow2
`

	// This script modifies the image after chroot is done
	modifyImageScript = `#! /bin/bash
set -e
HOST_BUILD_DIR="/tmp/ocne-image-build"

# Use network device to access QCOW2 image and modify the ignition provider type using sed
modprobe nbd max_part=16
qemu-nbd --connect=/dev/nbd0 $HOST_BUILD_DIR/boot.qcow2
mkdir -p $HOST_BUILD_DIR/p1
mkdir -p $HOST_BUILD_DIR/p2
mkdir -p $HOST_BUILD_DIR/p3

partprobe

# Stash the old UUIDs
OLD_BOOT_UUID=$(blkid -o value -s UUID /dev/nbd0p2)
OLD_ROOT_UUID=$(blkid -o value -s UUID /dev/nbd0p3)

# Generate some new UUIDs for the XFS partitions
i=0
while xfs_admin -U generate /dev/nbd0p2; [[ $? -ne 0 ]]; do
	sleep 2s
	echo "Retrying UUID generation for /dev/nbd0p2"
	((++i))
	if [[ $i -eq 30 ]]; then
		echo "Error generating UUID for /dev/nbd0p2"
		exit 1
	fi
done

i=0
while xfs_admin -U generate /dev/nbd0p3; [[ $? -ne 0 ]]; do
	sleep 2s
	echo "Retrying UUID generation for /dev/nbd0p3"
	((++i))
	if [[ $i -eq 30 ]]; then
		echo "Error generating UUID for /dev/nbd0p3"
		exit 1
	fi
done

# Get the new UUIDs
BOOT_UUID=$(blkid -o value -s UUID /dev/nbd0p2)
ROOT_UUID=$(blkid -o value -s UUID /dev/nbd0p3)

# sometimes the mount fails because the kernel doesn't yet know about the partition, so retry
i=0
while mount -o rw /dev/nbd0p2 $HOST_BUILD_DIR/p2; [[ $? -ne 0 ]]; do
        sleep 2s
        echo "Retrying mount /dev/nbd0p2"
        ((++i))
        if [[ $i -eq 30 ]]; then
                echo "Error mounting /dev/nbd0p2"
                exit 1
        fi
done

# Change the ignition platform ID
sed -i "s/ignition.platform.id=qemu/ignition.platform.id=${IGNITION_PROVIDER_TYPE}/g" $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf
if [ -n "${KARGS_APPEND_STANZA}" ]; then
	sed -i "${KARGS_APPEND_STANZA}" $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf
fi

# Change the UUIDs
sed -i "s/$OLD_ROOT_UUID/$ROOT_UUID/g" $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf

# Get the ostree commit to deploy
#
# OSTREE_DEPLOY is preceeded with a '/', hence the
# lack thereof in the pattern for OSTREE_DEPLOY_DIR
OSTREE_DEPLOY=$(cat $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf | grep -o 'ostree=[^ ]*' | cut -d= -f2)
OSTREE_DEPLOY_DIR="$HOST_BUILD_DIR/p3$OSTREE_DEPLOY"

umount $HOST_BUILD_DIR/p2

# Change /etc/fstab
# sometimes the mount fails because the kernel doesn't yet know about the partition, so retry
i=0
while mount -o rw /dev/nbd0p3 $HOST_BUILD_DIR/p3; [[ $? -ne 0 ]]; do
        sleep 2s
        echo "Retrying mount /dev/nbd0p3"
        ((++i))
        if [[ $i -eq 30 ]]; then
                echo "Error mounting /dev/nbd0p3"
                exit 1
        fi
done

sed -i "s|UUID=[a-zA-z0-9-]* /[ \t]|UUID=$ROOT_UUID / |g" "$OSTREE_DEPLOY_DIR/etc/fstab"
sed -i "s|UUID=[a-zA-z0-9-]* /boot[ \t]|UUID=$BOOT_UUID /boot |g" "$OSTREE_DEPLOY_DIR/etc/fstab"

# Remove machine id
find $HOST_BUILD_DIR/p3/ostree/deploy/ock/deploy -type f -iname machine-id | xargs -I {} rm {}

# Copy in files
#cp -r "$HOST_BUILD_DIR/files/*" "$OSTREE_DEPLOY_DIR"
cp "$HOST_BUILD_DIR/oci-dhclient.sh" "$OSTREE_DEPLOY_DIR/etc/oci-dhclient.sh"
cp "$HOST_BUILD_DIR/oci.sh" "$OSTREE_DEPLOY_DIR/etc/dhcp/dhclient.d/oci.sh"
cp "$HOST_BUILD_DIR/11-dhclient" "$OSTREE_DEPLOY_DIR/etc/NetworkManager/dispatcher.d/11-dhclient"

chmod +x "$OSTREE_DEPLOY_DIR/etc/oci-dhclient.sh"
chmod +x "$OSTREE_DEPLOY_DIR/etc/dhcp/dhclient.d/oci.sh"
chmod +x "$OSTREE_DEPLOY_DIR/etc/NetworkManager/dispatcher.d/11-dhclient"

umount $HOST_BUILD_DIR/p3

# Set the UUIDs back
#xfs_admin -U $OLD_BOOT_UUID /dev/nbd0p2
#xfs_admin -U $OLD_ROOT_UUID /dev/nbd0p3

# exit chroot so that the image can be copied to pod /tmp.  The client will download it after the script completes
exit
`

	dockerfileName = "Dockerfile"

	ostreeImageDockerfile = `
FROM container-registry.oracle.com/os/oraclelinux:8-slim

ARG PODMAN_IMG
ARG ARCH
ARG OSTREE_IMG

ADD ca-trust /tmp/ca-trust
ADD tls /tmp/tls
ADD make-archive.sh /make-archive.sh
RUN sh /make-archive.sh

CMD ["nginx", "-g", "daemon off;"]
EXPOSE 80
EXPOSE 443`

	ostreeArchiveScriptName = "make-archive.sh"
	ostreeArchiveScript     = `
#! /bin/bash
set -e

# Apply the given trust store
mv /etc/pki/ca-trust /tmp/ca-trust-orig
mv /etc/pki/tls /tmp/tls-orig
mv /tmp/ca-trust /etc/pki/ca-trust
mv /tmp/tls /etc/pki/tls

ls -l /etc/pki/tls
ls -l /etc/pki/ca-trust

# Install the things
microdnf upgrade
microdnf install ostree rpm-ostree skopeo-1.14.3 nginx podman jq

# Set up the filesystem
mkdir -p /ostree/img
mkdir -p /usr/share/nginx/html/ostree
ostree --repo=/ostree/img init --mode=bare-user
ostree --repo=/usr/share/nginx/html/ostree init --mode=archive

# It is possible that the given image is either a multi-arch manifest
# or a bare image.  Differentiating between the two is annoying, but
# must be done
DIGEST=""
MANIFEST=$(podman manifest inspect $PODMAN_IMG 2>&1 || true)
if echo "$MANIFEST" | grep -v "is not a manifest list but a single image"; then
	DIGEST=$(skopeo inspect docker://$PODMAN_IMG | jq -r .Digest)
else
	DIGEST=$(echo "$MANIFEST" | jq -r ".manifests[] | select( .platform.architecture == \"$ARCH\") | .digest")
	if [ -z "$DIGEST" ]; then
		>&2 echo "The image $PODMAN_IMG does not have an image for the $ARCH architecture"
		exit 1
	fi
fi
ostree container unencapsulate --repo=/ostree/img --write-ref ock $OSTREE_IMG@$DIGEST
ostree --repo=/usr/share/nginx/html/ostree pull-local /ostree/img

# Clean up the CA trust store
rm -rf /ostree/img /etc/pki/ca-trust /etc/pki/tls
mv /tmp/ca-trust-orig /etc/pki/ca-trust
mv /tmp/tls-orig /etc/pki/tls
`

	ostreeScriptName = "make-ostree-image.sh"
	ostreeScript     = `
export https_proxy=%s
export http_proxy=%s
export no_proxy=%s
export NO_PROXY="$no_proxy"
mkdir -p /hostroot/tmp/ostree-image/build
cp -r /hostroot/etc/pki/ca-trust /hostroot/tmp/ostree-image/build/ca-trust
cp -r /hostroot/etc/pki/tls /hostroot/tmp/ostree-image/build/tls
ls -l /hostroot/etc/pki/tls 
ls -l /hostroot/tmp/ostree-image/build/tls
ls -l /hostroot/etc/pki/ca-trust 
ls -l /hostroot/tmp/ostree-image/build/ca-trust
cp /ocne-image-build/Dockerfile /hostroot/tmp/ostree-image/build/Dockerfile
cp /ocne-image-build/make-archive.sh /hostroot/tmp/ostree-image/build/make-archive.sh

mkdir /hostroot/tmp/ostree-image/cache
mkdir /hostroot/tmp/ostree-image/run
cat > /hostroot/tmp/ostree-image/storage.conf << EOF
[storage]
driver = "overlay"
runroot = "/tmp/ostree-images/run"
graphroot = "/tmp/ostree-images/cache"

[storage.options]
pull_options = {enable_partial_images = "false", use_hard_links = "false", ostree_repos=""}

[storage.options.overlay]
mountopt = "nodev,metacopy=on"
EOF

export CONTAINERS_STORAGE_CONF=/tmp/ostree-image/storage.conf
chroot /hostroot podman build --no-cache --isolation chroot -t ock-ostree:latest --build-arg OSTREE_IMG='%s' --build-arg PODMAN_IMG=%s --build-arg ARCH=%s /tmp/ostree-image/build
`

	deployOckService = `
[Unit]
Description=Deploy OCK
DefaultDependencies=false

# Go between disks being partitioned and any file or mounting
# activity.
After=ignition-disks.service
Before=ignition-mount.service
Requires=systemd-udevd.service


# A bunch of stuff needs to be interrupted if install succeeds
Before=make-rootfs.service
Before=grow-rootfs.service
Before=ignition-files.service

# Go after devices are enumerating
After=systemd-udevd.service

# Don't get stuck in a loop if things go sideways
OnFailure=emergency.target
OnFailureJobMode=isolate

# If install succeeds, shut down immediately
OnSuccess=shutdown.target
OnSuccessJobMode=isolate

# Don't run if there is already a deployed ostree
ConditionKernelCommandLine=!ostree

[Service]
Type=oneshot
RemainAfterExit=yes
MountFlags=slave
ExecStart=/usr/sbin/deploy-ock
`
	deployOckScript = `
#! /bin/bash
set -e
set -x

SYSROOT="/sysroot-alt"
BOOT="${SYSROOT}/boot"
EFI="${BOOT}/efi"

ROOT_LABEL="root"
BOOT_LABEL="boot"
ROOT_FILESYSTEM="xfs"
BOOT_FILESYSTEM="xfs"

pushd /dev/disk/by-partlabel
EFI_LABEL=$(ls | grep EFI)
popd

OSTREE="${SYSROOT}/ostree"
OSTREE_REPO="${OSTREE}/repo"
OS_NAME=ock

# Disable ignition-files.  This can write stuff to /etc and
# break selinux
systemctl disable ignition-files.service

# Mount the root, boot, and efi partitions so
# they can be installed to
mkdir -p "$SYSROOT"
mount "/dev/disk/by-partlabel/$ROOT_LABEL" "$SYSROOT"
mkdir -p "$BOOT"
mount "/dev/disk/by-partlabel/$BOOT_LABEL" "$BOOT"
mkdir -p "$EFI"
mount "/dev/disk/by-partlabel/$EFI_LABEL" "$EFI"

mkdir -p /media
mount /dev/cdrom /media

# make a temp directory so that ostree is writing to
# disk instead of memory
rm -rf /var/tmp
mkdir -p "$SYSROOT/tmp"
ln -s "$SYSROOT/tmp" /var/tmp

# Get disk details to fill things like grub entries and fstab
ROOT_PATH="${SYSROOT}"
BOOT_PATH="${ROOT_PATH}/boot"
EFI_PATH="${BOOT_PATH}/efi"
ROOT_DETAILS=$(findmnt --output SOURCE,FSTYPE,FS-OPTIONS -n --target "${ROOT_PATH}")
BOOT_DETAILS=$(findmnt --output SOURCE,TARGET -n --target "${BOOT_PATH}")
EFI_DETAILS=$(findmnt --output SOURCE,TARGET -n --target "${EFI_PATH}")
ROOT_DEVICE=$(echo "$ROOT_DETAILS" | cut -d' ' -f1)
BOOT_DEVICE=$(echo "$BOOT_DETAILS" | cut -d' ' -f1)
EFI_DEVICE=$(echo "$EFI_DETAILS" | cut -d' ' -f1)

ROOT_UUID=$(blkid -o value -s UUID "$ROOT_DEVICE")
BOOT_UUID=$(blkid -o value -s UUID "$BOOT_DEVICE")
EFI_UUID=$(blkid -o value -s UUID "$EFI_DEVICE")

# Deploy the ostree
ostree admin init-fs --modern "$SYSROOT"
ostree config --repo "$OSTREE_REPO" set sysroot.readonly true
ostree admin os-init "$OS_NAME" --sysroot "$SYSROOT"

IMAGE=
if ! IMAGE="$(grep -o -e 'ostree-source=[^ ]*' /proc/cmdline)"; then
	IMAGE="ostree-unverified-image:oci-archive:/media/ostree.tar"
fi

# A policy.json is required to unencapulsate the container.  There is
# an issue with creating files under directories with additional cpios
# so copy one from a known location.
mkdir -p /etc/containers
cp /etc/policy.json /etc/containers/policy.json
ostree container unencapsulate --repo="$OSTREE_REPO" --write-ref "$OS_NAME" "$IMAGE"

ostree admin deploy --sysroot "$SYSROOT" --os "$OS_NAME" \
	--karg-proc-cmdline \
	--karg ignition.firstboot=1 \
	--karg root=UUID=${ROOT_UUID} \
	"$OS_NAME"

# Set up the fstab based on the current mount points
# - get current mount points
# - get UUID from device
# - get fs type and options from mount table


# Configure /etc/fstab for later ignition steps as well as the
# boot process

COMMIT=$(ostree log --repo="$OSTREE_REPO" ${OS_NAME} | grep commit | cut -d' ' -f2)
DEPLOY_DIR="${OSTREE}/deploy/${OS_NAME}/deploy/${COMMIT}.0"
cat > "${DEPLOY_DIR}/etc/fstab" << EOF
UUID=$ROOT_UUID / $ROOT_FILESYSTEM defaults 0 0
UUID=$BOOT_UUID /boot $BOOT_FILESYSTEM defaults,sync 0 0
UUID=$EFI_UUID /boot/efi vfat defaults,uid=0,gid=0,umask=077,shortname=winnt 0 2
EOF

cp -rn "${DEPLOY_DIR}/usr/lib/ostree-boot/efi" "${BOOT}"
cp -rn "${DEPLOY_DIR}/usr/lib/ostree-boot/grub2" "${BOOT}"
cp /etc/grub.cfg "${EFI}/EFI/redhat/grub.cfg"

# Add the ignition to the initramfs for the new install
TMP_INITRD="/etc/initrd.tmp"
FULL_INITRD="/etc/initrd.full.tmp"
REAL_INITRD=${BOOT}/ostree/${OS_NAME}-*/initramfs-*.img
pushd /
echo "config.ign" | cpio -oc | gzip -c > "$TMP_INITRD"
popd
cp $REAL_INITRD "$FULL_INITRD"
cat "$FULL_INITRD" "$TMP_INITRD" > $REAL_INITRD
rm -f "$TMP_INITRD" "$FULL_INITRD"

umount -R "$SYSROOT"
`

	setConfigService = `
[Unit]
Description=Set Ignition Configuration
DefaultDependencies=false

# Go before any ignition files are fetched so that the
# chosen one is actually used.
Before=ignition-fetch.service
Before=ignition-fetch-offline.service


# Don't get stuck in a loop if things go sideways
OnFailure=emergency.target
OnFailureJobMode=isolate

# Don't run if there is already a deployed ostree
ConditionKernelCommandLine=ock.config

[Service]
Type=oneshot
RemainAfterExit=yes
MountFlags=slave
ExecStart=/usr/sbin/set-config
`
	setConfigScript = `
#! /bin/bash
set -e
set -x

# Get the config from the kernel command line
CONF=$(cat /proc/cmdline | grep -o 'ock.config=[^ ]*' | cut -d= -f2)
if [ -z "$CONF" ]; then
	# A configuration was not specified.  Odd, but not impossible.
	echo "No configuration found"
	exit 0
fi

# The configuration file does not exist.  Fail.
if [ ! -f "/$CONF" ]; then
	echo "The configuration files /$CONFG does not exist"
	exit 1
fi

cp "/$CONF" /config.ign
`

	grubCfgFile = `
# This file is copied from https://github.com/coreos/coreos-assembler/blob/0eb25d1c718c88414c0b9aedd19dc56c09afbda8/src/grub.cfg
# Changes:
#   - Dropped Ignition glue, that can be injected into platform.cfg
# petitboot doesn't support -e and doesn't support an empty path part
if [ -d (md/md-boot)/grub2 ]; then
  # fcct currently creates /boot RAID with superblock 1.0, which allows
  # component partitions to be read directly as filesystems.  This is
  # necessary because transposefs doesn't yet rerun grub2-install on BIOS,
  # so GRUB still expects /boot to be a partition on the first disk.
  #
  # There are two consequences:
  # 1. On BIOS and UEFI, the search command might pick an individual RAID
  #    component, but we want it to use the full RAID in case there are bad
  #    sectors etc.  The undocumented --hint option is supposed to support
  #    this sort of override, but it doesn't seem to work, so we set $boot
  #    directly.
  # 2. On BIOS, the "normal" module has already been loaded from an
  #    individual RAID component, and $prefix still points there.  We want
  #    future module loads to come from the RAID, so we reset $prefix.
  #    (On UEFI, the stub grub.cfg has already set $prefix properly.)
  set boot=md/md-boot
  set prefix=($boot)/grub2
else
  if [ -f ${config_directory}/bootuuid.cfg ]; then
    source ${config_directory}/bootuuid.cfg
  fi
  if [ -n "${BOOT_UUID}" ]; then
    search --fs-uuid "${BOOT_UUID}" --set boot --no-floppy
  else
    search --label boot --set boot --no-floppy
  fi
fi
set root=$boot

if [ -f ${config_directory}/grubenv ]; then
  load_env -f ${config_directory}/grubenv
elif [ -s $prefix/grubenv ]; then
  load_env
fi

if [ -f $prefix/console.cfg ]; then
  # Source in any GRUB console settings if provided by the user/platform
  source $prefix/console.cfg
fi

if [ x"${feature_menuentry_id}" = xy ]; then
  menuentry_id_option="--id"
else
  menuentry_id_option=""
fi

function load_video {
  if [ x$feature_all_video_module = xy ]; then
    insmod all_video
  else
    insmod efi_gop
    insmod efi_uga
    insmod ieee1275_fb
    insmod vbe
    insmod vga
    insmod video_bochs
    insmod video_cirrus
  fi
}

# Other package code will be injected from here
if [ -e (md/md-boot) ]; then
  # The search command might pick a RAID component rather than the RAID,
  # since the /boot RAID currently uses superblock 1.0.  See the comment in
  # the main grub.cfg.
  set prefix=md/md-boot
else
  if [ -f ${config_directory}/bootuuid.cfg ]; then
    source ${config_directory}/bootuuid.cfg
  fi
  if [ -n "${BOOT_UUID}" ]; then
    search --fs-uuid "${BOOT_UUID}" --set prefix --no-floppy
  else
    search --label boot --set prefix --no-floppy
  fi
fi
if [ -d ($prefix)/grub2 ]; then
  set prefix=($prefix)/grub2
  configfile $prefix/grub.cfg
else
  set prefix=($prefix)/boot/grub2
  configfile $prefix/grub.cfg
fi
boot

if [ x$feature_timeout_style = xy ] ; then
  set timeout_style=menu
  set timeout=1
# Fallback normal timeout code in case the timeout_style feature is
# unavailable.
else
  set timeout=1
fi

# Import user defined configuration
# tracker: https://github.com/coreos/fedora-coreos-tracker/issues/805
if [ -f $prefix/user.cfg ]; then
  source $prefix/user.cfg
fi

blscfg
`
)
