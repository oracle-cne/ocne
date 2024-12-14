// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createOlvmImage(cc *copyConfig) error {
	// copy the qcow2 image from the local system to the pod
	kubectl.SetPipes(cc.KubectlConfig)
	err := uploadImage(cc)
	if err != nil {
		return err
	}

	// run the script in the pod to change the provider in the qcow2 image
	kubectl.SetPipes(cc.KubectlConfig)
	if err := kubectl.RunScript(cc.KubectlConfig, cc.podName, imageMountPath, setOlvmProviderScriptName); err != nil {
		return err
	}

	// copy boot image from pod to local system
	kubectl.SetPipes(cc.KubectlConfig)
	localBootImagePath, err := downloadImage(cc)
	if err != nil {
		return err
	}

	log.Infof("New boot image was created successfully at %s", localBootImagePath)
	return nil
}

func createOlvmConfigMap(namespace string, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Immutable: nil,
		Data: map[string]string{
			setOlvmProviderScriptName: setOlvmProviderScript,
			modifyOlvmImageScriptName: modifyOlvmImageScript,
		},
	}
}
