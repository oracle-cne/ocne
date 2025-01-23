// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

const (
	KubeadmUpgradePath = "/etc/ocne/ocne-kubeadm-upgrade.sh"
	KubeadmUpgrade = `#! /bin/bash
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
exit 0
`

	ImageCleanupPostPath = "/etc/ocne/ocne-image-cleanup-post.sh"
	ImageCleanupPost = `#! /bin/bash
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
set -e
set -x
K="container-registry.oracle.com/olcne/kube-proxy"
C="container-registry.oracle.com/olcne/coredns"
F='{{.Repository}}:{{.Tag}}'
ls -l /var/lib/containers/storage
podman images
BAD_TAGS=$(podman images 2>&1 1>/dev/null | sed 's/.* of image \([a-z0-9]*\) not found .*/\1/p' | sort | uniq)
for bad in $BAD_TAGS; do
  podman rmi -f "$bad"
done
KPC=$(podman images --format "$F" --filter "reference=$K:current")
if [ -z "$KPC" ]; then
  KP=$(podman images --format "$F" --filter "reference=$K" --sort created | head -1)
  podman tag "$KP" "$K:current"
fi
CDC=$(podman images --format "$F" --filter "reference=$C:current")
if [ -z "$CDC" ]; then
  CD=$(podman images --format "$F" --filter "reference=$C" --sort created | head -1)
  podman tag "$CD" "$C:current"
fi
`

	ImageCleanupPostServiceName="ocne-image-cleanup-post.service"
	ImageCleanupPostServicePath="/etc/systemd/system/ocne-image-cleanup-post.service"
	ImageCleanupPostService=`[Unit]
After=crio.service
After=ocne-image-cleanup.service
Before=kubelet.service
Wants=network-online.target

[Service]
ExecStart=/etc/ocne/ocne-image-cleanup-post.sh

[Install]
WantedBy=multi-user.target
`

	OckDirectory="/etc/ocne/ock"
	OckPatchDirectory="/etc/ocne/ock/patches"
)

var Files = map[string]string{
	KubeadmUpgradePath:   KubeadmUpgrade,
	ImageCleanupPostPath: ImageCleanupPost,
	ImageCleanupPostServicePath: ImageCleanupPostService,
}
