// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kexec "k8s.io/kubectl/pkg/cmd/exec"
	kutil "k8s.io/kubectl/pkg/cmd/util"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	log "github.com/sirupsen/logrus"
	"os"
)

type Overrides struct {
	Node  string
	Image string
}

type Options struct {
	// KubeConfigPath is the path of the kubeconfig file
	KubeConfigPath string

	// NodeName is the name of the node
	NodeName string

	// DefaultRegistry is the registry pull images from.
	DefaultRegistry string

	// Toolbox whether to use toolbox image or not
	Toolbox bool

	// Chroot whether to chroot or not
	Chroot bool

	// Commands is the specific commands to be run
	Commands []string
}

const podPrefix = "console"

func Console(options Options) error {
	// Get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(options.KubeConfigPath)
	if err != nil {
		return err
	}

	// Get kubeConfigPath
	kubeConfigPath, _, err := client.GetKubeConfigLocation(options.KubeConfigPath)
	if err != nil {
		return err
	}

	// sanity check to make sure we can access the cluster
	if _, err = k8s.WaitUntilGetNodesSucceeds(kubeClient); err != nil {
		return err
	}

	// Check if the node is valid
	if _, err = k8s.GetNode(kubeClient, options.NodeName); err != nil {
		return err
	}

	// Create the OCNE system namespace if it does not exist
	if err = k8s.CreateNamespaceIfNotExists(kubeClient, constants.OCNESystemNamespace); err != nil {
		return err
	}

	// Delete the pod, if it exists.
	podName := fmt.Sprintf("%s-%s", podPrefix, options.NodeName)
	k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, podName)

	// start the pod
	pod, err := k8s.StartAdminPodOnNode(kubeClient, options.NodeName, constants.OCNESystemNamespace, podPrefix, options.Toolbox, options.DefaultRegistry)
	defer k8s.DeletePod(kubeClient, constants.OCNESystemNamespace, podName)
	if err != nil {
		return err
	}

	// exec the pod
	tty := "false"
	isTTY, err := util.FileIsTTY(os.Stdin)
	if err != nil {
		return err
	}
	if isTTY {
		tty = "true"
	}

	insecure := true
	configFlags := &genericclioptions.ConfigFlags{
		KubeConfig: &kubeConfigPath,
		Namespace:  util.StrPtr(constants.OCNESystemNamespace),
		Insecure:   &insecure,
	}
	// MatchVersionFlags is necessary, or we will get "GroupVersion is required when initializing a RESTClient" attaching pod
	factory := kutil.NewFactory(kutil.NewMatchVersionFlags(configFlags))
	streams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	cmd := kexec.NewCmdExec(factory, streams)
	cmd.Flags().Set("tty", tty)
	cmd.Flags().Set("stdin", "true")
	cmdArgs := []string{pod.ObjectMeta.Name, "--"}
	if options.Chroot {
		cmdArgs = append(cmdArgs, "/usr/sbin/chroot", "/hostroot")
	} else if len(options.Commands) == 0 {
		cmdArgs = append(cmdArgs, "sh")
	}
	cmdArgs = append(cmdArgs, options.Commands...)
	log.Debugf("Executing command on %s: %+v", pod.ObjectMeta.Name, cmdArgs)
	cmd.SetArgs(cmdArgs)
	if err := cmd.Execute(); err != nil {
		return err
	}

	return nil
}
