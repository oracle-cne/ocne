// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

const (
	// This script deploys the new OCK, stops the update service, then clears the update annotation.
	updateNodeScript = `#! /bin/bash
set -e

chroot /hostroot /bin/bash <<"EOF"
  ostree admin deploy ock:ock
  systemctl stop ocne-update.service

  rpm-ostree kargs --delete-if-present=ignition.firstboot=1
  KUBECONFIG=/etc/kubernetes/kubelet.conf kubectl annotate node ${NODE_NAME} ocne.oracle.com/update-available-
  (sleep 3 && shutdown -r now)&
EOF
`
)
