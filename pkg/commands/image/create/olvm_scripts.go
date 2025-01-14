// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

const (
	setOlvmProviderScriptName = "set-provider.sh"
	modifyOlvmImageScriptName = "modify-image.sh"

	// The setProvider script changes the ignition provider type inside the boot qcow2 image.
	// The script runs both before, during, and after chroot.  The modprobe needs chroot to work
	// and if qemu-nbd is not run in chroot then it cannot find the nbd device.
	// The script first copies files to /hostroot so that they can be accessed after
	// chroot is done.  The modify_image.sh script runs from chroot and does the modifications then
	// exits chroot.  Finally, the script copies the modified boot.qcow2 from /hostroot to /tmp
	// so that it can be downloaded by the client
	setOlvmProviderScript = `#! /bin/bash
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
	modifyOlvmImageScript = `#! /bin/bash
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
sed -i "s/ignition.platform.id=qemu/ignition.platform.id=openstack/g" $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf
sed -i "s/console=ttyS0/console=tty0/g" $HOST_BUILD_DIR/p2/loader/entries/ostree-1-ock.conf
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

umount $HOST_BUILD_DIR/p3

# Set the UUIDs back
#xfs_admin -U $OLD_BOOT_UUID /dev/nbd0p2
#xfs_admin -U $OLD_ROOT_UUID /dev/nbd0p3

# exit chroot so that the image can be copied to pod /tmp.  The client will download it after the script completes
exit
`
)
