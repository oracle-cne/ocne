// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

const (
	// This script deploys the new OCK, stops the update service, then clears the update annotation.
	updateNodeScript = `#! /bin/bash
set -e

chroot /hostroot /bin/bash <<"EOF"
  ostree admin deploy ock:ock
  systemctl stop ocne-update.service

  # Patch ConfigMap kubeadm-config to update the CoreDNS image tag.  Only do patch on a control-plane node
  # and if the ConfigMap has not already been patched.
  if [ -f /etc/kubernetes/admin.conf ]; then
    # If the current CoreDNS value is equal to the desired value, then no need to patch again
    KUBECONFIG=/etc/kubernetes/admin.conf kubectl get configmap -n kube-system kubeadm-config -o jsonpath='{.data.ClusterConfiguration}' > /tmp/kubeadm-config.yaml
    if [ $(yq '.dns.imageTag' < /tmp/kubeadm-config.yaml) != "${CORE_DNS_IMAGE_TAG}" ]; then
      echo Updating CoreDNS image tag to ${CORE_DNS_IMAGE_TAG}
      KUBECONFIG=/etc/kubernetes/admin.conf kubectl get configmap -n kube-system kubeadm-config -o jsonpath='{.data.ClusterConfiguration}' | yq ".dns.imageTag=\"${CORE_DNS_IMAGE_TAG}\"" > /tmp/kubeadm-config.yaml
      sed -i 's/^/    /' /tmp/kubeadm-config.yaml
      sed -i '1s/^/data:\n  ClusterConfiguration: |\n/' /tmp/kubeadm-config.yaml
      KUBECONFIG=/etc/kubernetes/admin.conf kubectl patch configmap kubeadm-config -n kube-system --patch-file /tmp/kubeadm-config.yaml
    fi
  fi

  rpm-ostree kargs --delete-if-present=ignition.firstboot=1
  KUBECONFIG=/etc/kubernetes/kubelet.conf kubectl annotate node ${NODE_NAME} ocne.oracle.com/update-available-
  (sleep 3 && shutdown -r now)&
EOF
`
)
