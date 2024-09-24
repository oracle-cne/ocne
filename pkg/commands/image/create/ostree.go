// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

func createOstreeImage(cc *copyConfig) error {
	ignores := []string{
		"System has not been booted",
		"Failed to connect to bus",
		"archive: skipping",
		"Trying to pull",
		"Getting image source signatures",
		"Copying blob",
		"Copying config",
		"Writing manifest",
	}
	cc.KubectlConfig.IgnoreErrors = ignores
	log.Debugf("Using container image %s with architecture %s", cc.ostreeContainerImage, cc.imageArchitecture)

	// Convert the given container image to something that can be used
	// by the script.  Notable, it hacks the tag off the ostree image
	// and hacks the transport off to generate the podman image.
	ostreeTransport, registry, tag, err := image.ParseOstreeReference(cc.ostreeContainerImage)
	if err != nil {
		return err
	}
	if tag == "" {
		return fmt.Errorf("ostree image %s requires a tag", cc.ostreeContainerImage)
	}
	containerImage := fmt.Sprintf("%s:%s", ostreeTransport, registry)
	podmanImage := fmt.Sprintf("%s:%s", registry, tag)

	log.Debugf("The podman image is %s", podmanImage)
	script := fmt.Sprintf(ostreeScript, cc.httpsProxy, cc.httpProxy, cc.noProxy, containerImage, podmanImage, cc.imageArchitecture)

	waitors := []*logutils.Waiter{
		&logutils.Waiter{
			Message: "Generating container image",
			WaitFunction: func(i interface{}) error {
				return kubectl.RunScript(cc.KubectlConfig, cc.podName, imageMountPath, script)
			},
		},
	}

	haveError := logutils.WaitFor(logutils.Info, waitors)
	if haveError {
		return fmt.Errorf("Error generating container image: %v", waitors[0].Error)
	}

	kc := cc.KubectlConfig
	ckc, err := kubectl.NewKubectlConfig(cc.restConfig, *kc.ConfigFlags.KubeConfig, kc.Namespace, nil, false)
	if err != nil {
		return err
	}

	// Write the output of the command straight to a file
	imgDir, err := file.EnsureOcneImagesDir()
	if err != nil {
		return err
	}
	imgPath := filepath.Join(imgDir, fmt.Sprintf("ock-%s-%s-ostree.tar", cc.kubeVersion, cc.imageArchitecture))
	f, err := os.OpenFile(imgPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil
	}

	ckc.Streams.Out = f

	waitors = []*logutils.Waiter{
		&logutils.Waiter{
			Message: "Saving container image",
			WaitFunction: func(i interface{}) error {
				return kubectl.RunCommand(ckc, cc.podName, "sh", "-c", "export CONTAINERS_STORAGE_CONF=/tmp/ostree-image/storage.conf; chroot /hostroot podman save --format=oci-archive ock-ostree:latest")
			},
		},
	}
	haveError = logutils.WaitFor(logutils.Info, waitors)
	f.Close()
	if haveError {
		return fmt.Errorf("Error saving container image: %v", waitors[0].Error)
	}

	// Delete the image from the host
	// This is fast.  No reason to put it inside a waiter.
	rkc, err := kubectl.NewKubectlConfig(cc.restConfig, *kc.ConfigFlags.KubeConfig, kc.Namespace, nil, true)
	err = kubectl.RunCommand(rkc, cc.podName, "rm", "-rf", "/hostroot/tmp/ostree-image")
	if err != nil {
		return err
	}

	log.Infof("Saved image to %s", imgPath)
	return nil
}

func createOstreeConfigMap(namespace string, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Immutable: nil,
		Data: map[string]string{
			dockerfileName:          ostreeImageDockerfile,
			ostreeScriptName:        ostreeScript,
			ostreeArchiveScriptName: ostreeArchiveScript,
		},
	}
}
