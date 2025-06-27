// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package join

const (
	// This script writes the new kubeadm.conf, updates the bootstrap config to "join", writes the reset-kubeadm marker file,
	// and restarts the bootstrap service
	updateNodeScript = `#! /bin/bash
set -e

chroot /hostroot mv /etc/kubernetes/kubeadm.conf /etc/kubernetes/kubeadm.conf.bak || true
chroot /hostroot bash -c "echo \"$JOIN_CONFIG\" > /etc/kubernetes/kubeadm.conf"
chroot /hostroot sed -i '/Environment=ACTION=/c\Environment=ACTION=join' /etc/systemd/system/ocne.service.d/bootstrap.conf
chroot /hostroot systemctl daemon-reload
chroot /hostroot chown keepalived_script:keepalived_script /etc/keepalived/peers  /etc/keepalived/keepalived.conf /etc/keepalived/log || true
chroot /hostroot chown -R nginx_script:nginx_script || true
chroot /hostroot chown nginx_script:nginx_script /etc/ocne/nginx/nginx.conf || true
chroot /hostroot bash -c 'for svc in $ENABLE_SERVICES; do systemctl enable $svc --now; done'
chroot /hostroot touch /etc/ocne/reset-kubeadm
chroot /hostroot systemctl restart ocne
`
)
