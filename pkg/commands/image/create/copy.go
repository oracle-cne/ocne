// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/cp"

	kutil "k8s.io/kubectl/pkg/cmd/util"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

// copyConfig contains information used to copy to and from the pod
type copyConfig struct {
	*kubectl.KubectlConfig
	providerType             string
	bootVolumeContainerImage string
	ostreeContainerImage     string
	remotePath               string
	imageArchitecture        string
	kubeVersion              string
	podName                  string
	httpsProxy               string
	httpProxy                string
	noProxy                  string
	restConfig               *rest.Config
}

// uploadImage gets the local boot image and uploads it to the pod
// if the boot image doesn't exist in the local image cache then it will be downloaded
func uploadImage(cc *copyConfig) error {
	tmpPath, err := file.CreateOcneTempDir(tempDir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpPath)

	// Get the tarstream of the boot qcow2 image
	log.Infof("Getting local boot image for architecture: %s", cc.imageArchitecture)
	tarStream, closer, err := image.EnsureBaseQcow2Image(cc.bootVolumeContainerImage, cc.imageArchitecture)
	if err != nil {
		return err
	}
	defer closer()

	// Write the local image. e.g. ~/.ocne/tmp/create-images.xyz/boot.oci
	localImagePath := filepath.Join(tmpPath, localVMImage+cc.providerType)
	err = writeFile(tarStream, localImagePath)
	if err != nil {
		return err
	}

	// copy the qcow2 image from the local system to the pod
	waitMsg := fmt.Sprintf("Uploading boot image to pod %s/%s", cc.Namespace, cc.podName)
	if err := copyFileToPod(cc, localImagePath, waitMsg); err != nil {
		return err
	}

	return nil
}

// download the boot image from the pod to the local file system
func downloadImage(cc *copyConfig) (string, error) {
	// copy the qcow2 image from the pod to the local system
	localFilePath, err := DefaultImagePath(cc.providerType, cc.kubeVersion, cc.imageArchitecture)
	if err != nil {
		return "", err
	}

	waitMsg := fmt.Sprintf("Downloading boot image from pod %s/%s", cc.Namespace, cc.podName)
	if err := copyFileFromPod(cc, localFilePath, waitMsg); err != nil {
		return "", err
	}

	return localFilePath, nil
}

// copyFileToPod copies the image file the pod
func copyFileToPod(c *copyConfig, localPath string, waitMsg string) error {
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				//	kubectl cp <localfilePath> <namespace>/<pod>:<filePath>
				cmd := cp.NewCmdCp(kutil.NewFactory(kutil.NewMatchVersionFlags(c.ConfigFlags)), c.Streams)
				s2 := fmt.Sprintf("%s/%s:%s", c.Namespace, c.podName, c.remotePath)
				args := []string{"--retries=5", localPath, s2}
				cmd.SetArgs(args)
				return cmd.Execute()
			},
		},
	})
	if haveError == true {
		return fmt.Errorf("Timeout copying file to pod %s/%s", c.Namespace, c.podName)
	}
	return nil
}

// copyFileFromPod copies a remote file in a pod to a local file
func copyFileFromPod(c *copyConfig, localPath string, waitMsg string) error {
	kc := c.KubectlConfig
	ckc, err := kubectl.NewKubectlConfig(c.restConfig, *kc.ConfigFlags.KubeConfig, kc.Namespace, nil, false)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	ckc.Streams.Out = f

	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				return kubectl.RunCommand(ckc, c.podName, "sh", "-c", fmt.Sprintf("cat %s", c.remotePath))
			},
		},
	})

	f.Close()

	if haveError == true {
		return fmt.Errorf("Error copying file to pod %s/%s", c.Namespace, c.podName)
	}
	return nil
}

func writeFile(reader io.Reader, filePath string) error {
	w, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}

	return nil
}

func DefaultImagePath(providerID string, kubeVersion string, arch string) (string, error) {
	localDir, err := file.EnsureOcneImagesDir()
	if err != nil {
		return "", err
	}

	// Write the local image. e.g. ~/.ocne/images/boot.qcow2-1.28-amd64.oci
	localImagePath := fmt.Sprintf("%s-%s-%s.%s", localVMImage, kubeVersion, arch, providerID)
	return filepath.Join(localDir, localImagePath), nil
}
