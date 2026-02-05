// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"github.com/oracle-cne/ocne/pkg/cluster/ignition"
)

const (

	CheckNodeUpdate = `#! /bin/bash
shopt -s extglob

BOOT_REF=$(chroot /hostroot ostree admin status | grep '^\*' | cut -d' ' -f2)
OCK_REFS=$(chroot /hostroot ostree refs | grep '^ock')

BOOT_COMMIT_DATE=$(chroot /hostroot ostree log "$BOOT_REF" | grep '^Date:')
BOOT_COMMIT_DATE="${BOOT_COMMIT_DATE##Date:+( )}"

UPDATE_COMMIT_DATE="$BOOT_COMMIT_DATE"
if echo "$OCK_REFS" | grep -q -e 'ock:ock'; then
	UPDATE_COMMIT_DATE=$(chroot /hostroot ostree log ock:ock | grep '^Date:')
	UPDATE_COMMIT_DATE="${UPDATE_COMMIT_DATE##Date:+( )}"
fi

echo '{}' | chroot /hostroot jq ".boot_timestamp = \"$BOOT_COMMIT_DATE\" | .update_timestamp = \"$UPDATE_COMMIT_DATE\""
`

	GetControlPlaneEndpointNic = `#! /bin/bash
DEV=$(ip route get %s | cut -d' ' -f3 | head -1)
ip -br addr show dev "$DEV" | tr -s ' '
`

	GetControlPlaneVipDetails = `#! /bin/bash
systemctl is-active keepalived.service
systemctl is-enabled keepalived.service
systemctl is-active ocne-nginx.service
systemctl is-enabled ocne-nginx.service
`

	GetKeepalivedConf = `#! /bin/bash
cat /hostroot/etc/keepalived/keepalived.conf || true
`

	UpdateVipConfiguration = `#! /bin/bash
rm /hostroot/etc/keepalived/kubeconfig
rm /hostroot/etc/ocne/nginx-refresh/kubeconfig

{{ range $idx, $u := .Units }}
chroot /hostroot systemctl stop {{ $u }}
{{end}}
{{ range $p, $c := .Files }}
echo "{{ $c }}" | base64 -d > "/hostroot{{ $p }}"
{{ end}}
chroot /hostroot systemctl daemon-reload
{{ range $idx, $u := .UnitsToEnable }}
chroot /hostroot systemctl enable {{ $u }}
{{ end }}
{{ range $idx, $u := .UnitsToStart }}
chroot /hostroot systemctl start {{ $u }}
{{ end }}
exit 0
`
)

var Files = map[string]string{
	ignition.KubeadmUpgradePath:   ignition.KubeadmUpgrade,
}
