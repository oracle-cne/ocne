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

	OckDirectory="/etc/ocne/ock"
	OckPatchDirectory="/etc/ocne/ock/patches"
)

var Files = map[string]string{
	KubeadmUpgradePath:   KubeadmUpgrade,
}
