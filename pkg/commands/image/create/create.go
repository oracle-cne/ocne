// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"

	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util"
)

const ProviderTypeOCI = "oci"
const ProviderTypeOstree = "ostree"

const (
	podName           = "ocne-image-builder"
	cmName            = "ocne-image-builder"
	imageMountPath    = "/ocne-image-build"
	remoteFilePath    = "/tmp/boot.qcow2"
	localVMImage      = "boot.qcow2"
	tempDir           = "create-images"
	envIgnitionStanza = "IGNITION_STANZA"
	envKargs          = "KARGS_APPEND_STANZA"

	ociDefaultIgnition  = "ignition.platform.id=oci"
	qemuDefaultIgnition = "ignition.platform.id=qemu"
)

// CreateOptions are the options for the create image command
type CreateOptions struct {
	// IgnitionProvider is the provider type for ignition
	IgnitionProvider string

	// KernelArguments is any extra kernel command line arguments to append
	KernelArguments string

	// ProviderConfigPath is the path for the provider config (e.g ~/.oci/config)
	ProviderConfigPath string

	// ProviderType is the provider type (e.g. oci)
	ProviderType string

	// Architecture of the image to create ("amd64", "arm64")
	Architecture string

	// Destination
	Destination string
}

type providerFuncs struct {
	defaultIgnitionStanza string
	createConfigMap       func(string, string) *corev1.ConfigMap
	createImage           func(*copyConfig) error
}

// Create creates a qcow2 image for the specified provider type
func Create(startConfig *otypes.Config, clusterConfig *otypes.ClusterConfig, options CreateOptions) error {
	namespace := constants.OCNESystemNamespace

	kubeConfig, isEphemeral, err := start.EnsureCluster(startConfig.KubeConfig, startConfig, clusterConfig)
	if err != nil {
		return err
	}

	if isEphemeral {
		defer func() {
			err := start.StopEphemeralCluster(startConfig, clusterConfig)
			if err != nil {
				log.Errorf("Error deleting ephemeral cluster: %v", err)
			}
		}()
	}

	// Get a kubernetes client
	restConfig, kubeClient, err := client.GetKubeClient(kubeConfig)
	if err != nil {
		return err
	}

	// sanity check to make sure we can access cluster
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// Ensure the namespace exists
	err = k8s.CreateNamespaceIfNotExists(kubeClient, namespace)

	log.Info("Preparing pod used to create image")

	// create the configmap with the scripts that will run on the pod. First delete cm if exists.
	// the pod mounts this configmap
	if err := k8s.DeleteConfigmap(kubeClient, namespace, cmName); err != nil {
		return err
	}
	defer k8s.DeleteConfigmap(kubeClient, namespace, cmName)
	if err := createConfigMap(kubeClient, namespace, cmName, options.ProviderType); err != nil {
		return err
	}

	ignitionStanza := options.IgnitionProvider
	if ignitionStanza == "" {
		ignitionStanza = providers[options.ProviderType].defaultIgnitionStanza
	}

	kargsStanza, err := generateKernelArgsStanza(options.KernelArguments)
	if err != nil {
		return err
	}

	// create the pod, first delete the pod if it exists
	if err := k8s.DeletePod(kubeClient, namespace, podName); err != nil {
		return err
	}
	defer k8s.DeletePod(kubeClient, namespace, podName)
	if err := createPod(kubeClient, namespace, podName, constants.DefaultPodImage, ignitionStanza, kargsStanza); err != nil {
		return err
	}

	// wait for pod to be ready
	if err := k8s.WaitUntilPodReady(kubeClient, namespace, podName); err != nil {
		return err
	}

	// get config needed to use kubctl
	kcConfig, err := kubectl.NewKubectlConfig(restConfig, startConfig.KubeConfig, namespace, nil, true)
	if err != nil {
		return err
	}

	// create config need for copy
	cc := &copyConfig{
		KubectlConfig:            kcConfig,
		providerType:             options.ProviderType,
		bootVolumeContainerImage: clusterConfig.BootVolumeContainerImage,
		ostreeContainerImage:     fmt.Sprintf("%s:%s", clusterConfig.OsRegistry, clusterConfig.OsTag),
		remotePath:               remoteFilePath,
		kubeVersion:              clusterConfig.KubeVersion,
		imageArchitecture:        options.Architecture,
		podName:                  podName,
		httpsProxy:               startConfig.Proxy.HttpsProxy,
		httpProxy:                startConfig.Proxy.HttpProxy,
		noProxy:                  startConfig.Proxy.NoProxy,
		restConfig:               restConfig,
	}

	return createImage(cc)
}

// createPod creates a pod that mounts the config map with the same name as the pod
func createPod(client kubernetes.Interface, namespace string, name string, imageName string, ignitionStanza string, kargs string) error {
	privileged := true
	builderVolumeName := "builder"
	hostVolumeName := "host-root"
	hostPathType := corev1.HostPathDirectory
	var accessMode int32 = 0o0500
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:    util.Int64Ptr(0),
				RunAsGroup:   util.Int64Ptr(0),
				RunAsNonRoot: util.BoolPtr(false),
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    name,
					Image:   imageName,
					Command: []string{"sleep", "10d"},
					Env: []corev1.EnvVar{
						{Name: envIgnitionStanza, Value: ignitionStanza},
						{Name: envKargs, Value: kargs},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      builderVolumeName,
							MountPath: imageMountPath,
						},
						{
							Name:      hostVolumeName,
							MountPath: "/hostroot",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: builderVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: name,
							},
							DefaultMode: &accessMode,
						},
					},
				},
				{
					Name: hostVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
							Type: &hostPathType,
						},
					},
				},
			},
		},
	}

	_, err := client.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	return err
}

func createOciConfigMap(namespace string, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Immutable: nil,
		Data: map[string]string{
			setProviderScriptName: setProviderScript,
			modifyImageScriptName: modifyImageScript,
			OciDhcpScriptPath:     OciDhcpScript,
			OciDhclientScriptPath: OciDhclientScript,
			OciDhclientPath:       OciDhclient,
		},
	}
}

func createOciImage(cc *copyConfig) error {
	// copy the qcow2 image from the local system to the pod
	kubectl.SetPipes(cc.KubectlConfig)
	err := uploadImage(cc)
	if err != nil {
		return err
	}

	// run the script in the pod to change the provider in the qcow2 image
	kubectl.SetPipes(cc.KubectlConfig)
	if err := kubectl.RunScript(cc.KubectlConfig, cc.podName, imageMountPath, setProviderScriptName); err != nil {
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

var providers = map[string]providerFuncs{
	ProviderTypeOCI: providerFuncs{
		defaultIgnitionStanza: ociDefaultIgnition,
		createConfigMap:       createOciConfigMap,
		createImage:           createOciImage,
	},
	ProviderTypeOstree: providerFuncs{
		defaultIgnitionStanza: qemuDefaultIgnition,
		createConfigMap:       createOstreeConfigMap,
		createImage:           createOstreeImage,
	},
}

// createConfigMap that has the scripts to be run by the pod
func createConfigMap(client kubernetes.Interface, namespace string, name string, provider string) error {
	pf, ok := providers[provider]
	if !ok {
		return fmt.Errorf("%s is not a supported provider", provider)
	}

	cm := pf.createConfigMap(namespace, name)
	err := k8s.CreateConfigmap(client, cm)
	return err
}

func createImage(cc *copyConfig) error {
	pf, ok := providers[cc.providerType]
	if !ok {
		return fmt.Errorf("%s is not a supported provider", cc.providerType)
	}

	return pf.createImage(cc)
}

// generateKernelArgsStanza creates a sed command that appends
// a string to the end of the kernel command line in a grub
// entry
func generateKernelArgsStanza(args string) (string, error) {
	// Sed supports arbitrary expression separators for
	// replacement expressions.  The input string can have
	// any set of characters in it.  Find a character that is
	// not in the string and use that.  The reason for doing
	// this is that there is no need to try escaping characters
	// in the input string.  The downside is that it cannot
	// support the case where the input string contains all
	// printable characters.  That seems unlikely.

	// If there are no args, then there is no stanza
	if args == "" {
		return "", nil
	}

	// This is the range of printable characters, more or less.
	var bang byte = 33
	var tilde byte = 126

	sep := ""
	for sepByte := bang; sepByte <= tilde; sepByte = sepByte + 1 {
		sepCandidate := string([]byte{sepByte})

		if !strings.Contains(args, sepCandidate) {
			sep = sepCandidate
			break
		}
	}

	// No acceptable character was found.  It's hard to imagine this
	// actually happening, but stranger things do.
	if sep == "" {
		return "", fmt.Errorf("extra kernel arguments use all printable characters")
	}

	// Build a sed command that appends something to the end of the options
	// line.  Assume that the separator is '|' and the input string is
	// "foo bar baz".  The result will be:
	//
	// /^options / s|$| foo bar baz|
	return fmt.Sprintf("/^options / s%s$%s %s%s", sep, sep, args, sep), nil
}
