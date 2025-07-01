// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

const (
	// This script deploys the new OCK, stops the update service, then clears the update annotation.
	updateNodeScript = `#! /bin/bash
set -e
set -o pipefail

chroot /hostroot /bin/bash <<"EOF"
  ostree admin deploy ock:ock
  systemctl stop ocne-update.service

  # Get the OSTree commits
  #
  # Figure out the various arch suffices
  PKG_ARCH=x64
  RPM_ARCH=x86_64
  if [ "$(uname -m)" = "aarch64" ]; then
    PKG_ARCH=aa64
    RPM_ARCH=aarch64
  fi

  # For each commit, get the version of shim-$PKG_ARCH
  #
  # Get all commit.version pairs
  COMMITS=$(ostree admin status | grep ' ock [0-9a-z]\+\.[0-9]\+' | sed 's/.*ock \([a-z0-9]\+\.[0-9]\+\).*/\1/')
  CURRENT_COMMIT=$(ostree admin status | grep '\* ock [0-9a-z]\+\.[0-9]\+' | sed 's/.*ock \([a-z0-9]\+\.[0-9]\+\).*/\1/')
  CURRENT_COMMIT_HASH=
  CURRENT_PKG_VER=
  PKGS=
  for commit in $COMMITS; do
    echo "Checking commit $commit"
    COMMIT_HASH=$(echo "$commit" | sed 's/\([a-z0-9]\+\)\.[0-9]\+/\1/')
    PKG_VER=$(rpm-ostree db list "$COMMIT_HASH" "shim-$PKG_ARCH" | grep "shim-$PKG_ARCH" | sed "s/^ shim-x64-\([0-9.-]*\).el8[_0-9]*.$RPM_ARCH/\1/")
    PKGS=$(echo "${PKGS}${PKG_VER} $commit|")

    if [ "$commit" = "$CURRENT_COMMIT" ]; then
      CURRENT_PKG_VER="$PKG_VER"
      CURRENT_COMMIT_HASH="$COMMIT_HASH"
    fi
  done
  PKGS=$(echo $PKGS | tr '|' '\n')
  echo Packages
  echo "$PKGS"

  # Sort the packages by version
  SORTED=$(echo "$PKGS" | sort -rV)
  echo Sorted
  echo "$SORTED"

  # Get the latest
  LATEST=$(echo "$SORTED" | head -1)
  echo "Latest: $LATEST"

  LATEST_COMMIT=$(echo "$LATEST" | cut -d' ' -f2)
  echo "Commit: $LATEST_COMMIT"

  # If the package has been updated, copy in the new content
  # while taking a backup of the existing content.
  LATEST_PKG_VER=$(echo "$LATEST" | cut -d' ' -f1)
  echo "Package versions: $LATEST_PKG_VER $CURRENT_PKG_VER"

  if [ "$CURRENT_PKG_VER" != "$LATEST_PKG_VER" ]; then
    # Back up the previous EFI content, creating a persistent backup from
    # this boot as well as one indicating the latest working version
    cp -rf /boot/efi/EFI "/boot/efi/EFI.${CURRENT_COMMIT_HASH}"
    rm -rf /boot/efi/EFI.latest_working
    cp -rf /boot/efi/EFI /boot/efi/EFI.latest_working

    # Copy over EFI content
    cp -r /ostree/deploy/ock/deploy/$LATEST_COMMIT/usr/lib/ostree-boot/efi/EFI/* /boot/efi/EFI
  fi


  rpm-ostree kargs --delete-if-present=ignition.firstboot=1
  KUBECONFIG=/etc/kubernetes/kubelet.conf kubectl annotate node ${NODE_NAME} ocne.oracle.com/update-available-
  (sleep 3 && shutdown -r now)&
EOF
`

	getUpdateInfo = "cat /hostroot/etc/ocne/update.yaml"
)
