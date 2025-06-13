// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dump

const (
	dumpScriptName       = "dump.sh"
	dumpFullScriptName   = "dump-full.sh"
	dumpSubsetScriptName = "dump-subset.sh"

	// The dump.sh does a chroot to /hostroot, runs the script to dump info into files, then
	// copies the files so that they can be downloaded by the client.
	dumpScript = `#! /bin/bash
set -e

HOST_OUT_DIR="/tmp/ocne/out"
HOST_SCRIPTS_DIR="/tmp/ocne/scripts"
HOSTROOT_OUT_DIR="/hostroot/tmp/ocne/out"
HOSTROOT_SCRIPTS_DIR="/hostroot/tmp/ocne/scripts"

# copy scripts to /hostroot so that they can be accessed once we do chroot
mkdir -p $HOSTROOT_OUT_DIR
mkdir -p $HOSTROOT_SCRIPTS_DIR
cp /ocne-scripts/..data/* $HOSTROOT_SCRIPTS_DIR

# chroot and run the script to modify the image
chroot /hostroot $HOST_SCRIPTS_DIR/dump-subset.sh
if [ "$OCNE_DUMP_FULL" = TRUE ]; then
  chroot /hostroot $HOST_SCRIPTS_DIR/dump-full.sh
fi

# copy the output files
mkdir -p $HOST_OUT_DIR
cp $HOSTROOT_OUT_DIR/* $HOST_OUT_DIR
rm -fr $HOSTROOT_OUT_DIR
rm -fr $HOSTROOT_SCRIPTS_DIR
`

	// This dumps data that is available after doing chroot /hostroot
	dumpFullScript = `#! /bin/bash
set -e

cd /tmp/ocne/out
top -b -n 1 >./top.out
systemctl -t service > ./services.out
df -Th > ./disk-df.out
du -h --threshold=100K -d 4 --exclude=/proc / | sort -rh > ./disk-du.out

# dump all the image information
podman images > ./podman-images.out
podman images --format json > ./podman-images.json
podman info --format json > ./podman-info.json

# dump the image details and fix the output to be valid json
podman images --format '{{.Id}}' | xargs -i sh -c 'podman inspect {} || true ' > ./podman-inspect-all.json 2>./podman-inspect-all-err.out
perl -0777  -pi -e 's/\](\n)\[/\,/g' ./podman-inspect-all.json

# dump the image errors with debug so we can get the image name along with the errors
podman images --format '{{.Id}}' | xargs -i sh -c 'podman inspect  --log-level=debug {} || true ' > /dev/null 2>./podman-inspect-all-err-debug.out

journalctl --sync
journalctl --list-boots > ./journal.boots.out
journalctl --disk-usage > ./journal.disk-usage.out
journalctl -S 1970-01-01 -n 2000 > ./journal.head.out
journalctl -S 1970-01-01 -b | grep -i error > ./journal.errors.out
journalctl -n 2000 > ./journal.tail.out
journalctl -u kubelet.service -m --no-hostname > ./journal.kubelet.service
journalctl -u grow-rootfs.service > ./journal.grow-rootfs.service
journalctl -u crio.service -m --no-hostname > ./journal.crio.service
journalctl -u keepalived.service -m --no-hostname > ./journal.keepalived.service
journalctl -u ocne.service -m --no-hostname > ./journal.ocne.service
journalctl -u ocne-image-cleanup.service -m --no-hostname > ./journal.ocne-image-cleanup.service
journalctl -u ocne-nginx.service -m --no-hostname > ./journal.ocne-nginx.service
journalctl -u ocne-nginx-refresh.service -m --no-hostname > ./journal.ocne-nginx-refresh.service
journalctl -u ocne-update.service -m --no-hostname > ./journal.ocne-update.service
journalctl -u ostree-remount.service -m --no-hostname > ./journal.ostree-remount.service

# exit chroot so that the image can be copied to pod /tmp.  The client will download it after the script completes
exit
`

	// This dumps a subset of data that is available after doing chroot /hostroot
	// This is all the information displayed by ocne cluster info, but also included in ocne cluster dump
	dumpSubsetScript = `#! /bin/bash
set -e

# dump a file if it exists
dump_file() {
  IN=$1
  OUT=$2
  if [ -f $IN ]; then
    cat $IN > $OUT
  fi
}

cd /tmp/ocne/out
dump_file "/etc/ocne/update.yaml" "./update.yaml"
dump_file "/etc/crictl.yaml" "./crictl.conf"
dump_file "/etc/crio/crio.conf" "./crio.conf"
dump_file "/etc/systemd/system/crio.service.d/proxy.conf" "./crio-proxy.conf"
dump_file "/etc/containers/registries.conf" "./crio-registries.conf"

if type ostree > /dev/null; then
  ostree admin status | grep -vi version | grep -vi origin > ./ostree-refs.out
fi

# exit chroot so that the image can be copied to pod /tmp.  The client will download it after the script completes
exit

`
)
