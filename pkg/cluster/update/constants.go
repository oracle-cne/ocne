// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

const (
	KubeadmUpgradePath = "/etc/ocne/ocne-kubeadm-upgrade.sh"
	KubeadmUpgrade = `#! /bin/bash
#
# Copyright (c) 2025, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
exit 0
`

	OckDirectory="/etc/ocne/ock"
	OckPatchDirectory="/etc/ocne/ock/patches"

	CheckNodeUpdate = `#! /bin/bash
shopt -s extglob

BOOT_REF=$(chroot /hostroot ostree admin status | grep '^\*' | cut -d' ' -f2)
OCK_REFS=$(chroot /hostroot ostree refs | grep '^ock')

BOOT_COMMIT_DATE=$(chroot /hostroot ostree log "$BOOT_REF" | grep '^Date:')
BOOT_COMMIT_DATE="${BOOT_COMMIT_DATE##Date:+( )}"

UPDATE_COMMIT_DATE=""
if echo "$OCK_REFS" | grep -q -e 'ock:ock'; then
	UPDATE_COMMIT_DATE=$(chroot /hostroot ostree log ock:ock | grep '^Date:')
	UPDATE_COMMIT_DATE="${UPDATE_COMMIT_DATE##Date:+( )}"
fi

echo '{}' | chroot /hostroot jq ".boot_timestamp = \"$BOOT_COMMIT_DATE\" | .update_timestamp = \"$UPDATE_COMMIT_DATE\""
`
)

var Files = map[string]string{
	KubeadmUpgradePath:   KubeadmUpgrade,
}
