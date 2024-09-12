// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stage

const (
	//TODO: Very important! Make sure user input is in the form N.N where N is a number. The input should not exceed 4 characters
	stageNodeScript = `#! /bin/bash
	set -e
    OLD_K8S_VERSION=$(chroot /hostroot sudo yq '.tag' /etc/ocne/update.yaml)
    OLD_REGISTRY=$(chroot /hostroot sudo yq '.registry' /etc/ocne/update.yaml)
    OLD_TRANSPORT=$(chroot /hostroot sudo yq '.transport' /etc/ocne/update.yaml)
    if [[ "$OLD_K8S_VERSION" == "$NEW_K8S_VERSION" ]] && [[ "$OLD_REGISTRY" == "$NEW_REGISTRY" || "$NEW_REGISTRY" == "" ]] && [[ "$OLD_TRANSPORT" == "$NEW_TRANSPORT" || "$NEW_TRANSPORT" == "" ]]
    then
       # if a previous run of this script failed before restarting the update service (or it is stopped for any reason), start it
       chroot /hostroot systemctl status ocne-update.service
       if [[ $? -ne 0 ]]; then
          chroot /hostroot systemctl start ocne-update.service
       fi
       exit 0
    fi 
	chroot /hostroot sudo sed "s/tag:.*/tag: $NEW_K8S_VERSION/" /etc/ocne/update.yaml -i
    if [[ "$NEW_REGISTRY" != "" ]]
    then 
       chroot /hostroot sudo sed "s?registry:.*?registry: $NEW_REGISTRY?" /etc/ocne/update.yaml -i
    fi 
    if [[ $NEW_TRANSPORT != "" ]] 
    then 
       chroot /hostroot sudo sed "s?transport:.*?transport: $NEW_TRANSPORT?" /etc/ocne/update.yaml -i
    fi 
	chroot /hostroot systemctl stop ocne-update.service
	KUBECONFIG=/etc/kubernetes/kubelet.conf chroot /hostroot kubectl annotate node ${NODE_NAME} ocne.oracle.com/update-available-
	chroot /hostroot systemctl start ocne-update.service
	`
)
