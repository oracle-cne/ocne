// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package script

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
)

func RunScript(client kubernetes.Interface, kcConfig *kubectl.KubectlConfig, nodeName string, namespace string, action string, script string, envVars []corev1.EnvVar) error {
	nodeAction := action + "-" + nodeName
	podName := nodeAction + "-pod"
	cmName := nodeAction + "-cm"
	scriptName := nodeAction + ".sh"

	// create the configmap with the scripts that will run in the job, first delete cm if exists.
	// the job mounts this configmap
	if err := k8s.DeleteConfigmap(client, namespace, cmName); err != nil {
		return err
	}
	defer k8s.DeleteConfigmap(client, namespace, cmName)
	if err := k8s.CreateConfigMapWithData(client, namespace, cmName, map[string]string{scriptName: script}); err != nil {
		return err
	}

	// create the pod, first delete the pod if it exists
	if err := k8s.DeletePod(client, namespace, podName); err != nil {
		return err
	}
	p := &k8s.PodOptions{
		NodeName:      nodeName,
		Namespace:     namespace,
		PodName:       podName,
		ImageName:     constants.DefaultPodImage,
		ConfigMapName: cmName,
		HostPID:       true,
		HostNetwork:   true,
		Env:           envVars,
	}
	defer k8s.DeletePod(client, namespace, podName)
	if err := k8s.CreateScriptsPod(client, p); err != nil {
		return err
	}

	// wait for pod to be ready
	if err := k8s.WaitUntilPodReady(client, namespace, podName); err != nil {
		return err
	}

	// run the script from the pod
	if err := kubectl.RunScript(kcConfig, podName, constants.ScriptMountPath, scriptName); err != nil {
		return err
	}

	return nil
}
