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
)

var ostreeImageDockerfile = `
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
