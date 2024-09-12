// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubectl

import (
	"bytes"
	"fmt"
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	outil "github.com/oracle-cne/ocne/pkg/util"
	"os"
	"strings"
)

// KubectlConfig contains information used access a pod via kubectl
type KubectlConfig struct {
	ConfigFlags     *genericclioptions.ConfigFlags
	Streams         genericiooptions.IOStreams
	StreamOut       io.Reader
	StreamErr       io.Reader
	StreamOutWriter io.WriteCloser
	StreamErrWriter io.WriteCloser
	Namespace       string
	ErrBuf          *bytes.Buffer
	IgnoreErrors    []string
}

type kErrWriter struct {
	ErrBuf *bytes.Buffer
	ignore []string
}

// NewKubectlConfig gets the configuration needed to use kubectl
func NewKubectlConfig(restConfig *rest.Config, kubeConfigPath string, namespace string, ignoreErrors []string, usePipes bool) (*KubectlConfig, error) {
	// Create ConfigFlags so that kubectl cmd package can be used
	kubeConfigPath, _, err := client.GetKubeConfigLocation(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	kubeInfo := client.KubeInfo{KubeconfigPath: kubeConfigPath, RestConfig: restConfig}
	configFlags := &genericclioptions.ConfigFlags{
		KubeConfig:    &kubeInfo.KubeconfigPath,
		BearerToken:   &kubeInfo.RestConfig.BearerToken,
		Namespace:     &namespace,
		Insecure:      outil.BoolPtr(true),
		TLSServerName: &kubeInfo.KubeApiServerIP,
	}

	out := io.Discard

	b := &bytes.Buffer{}
	var errOut io.Writer
	errOut = &kErrWriter{
		ignore: ignoreErrors,
		ErrBuf: b,
	}

	kc := KubectlConfig{
		ErrBuf:      b,
		ConfigFlags: configFlags,
		Streams: genericiooptions.IOStreams{
			In:     os.Stdin,
			Out:    out,
			ErrOut: errOut,
		},
		StreamOut:       nil,
		StreamErr:       nil,
		StreamOutWriter: nil,
		StreamErrWriter: nil,
		Namespace:       namespace,
		IgnoreErrors:    ignoreErrors,
	}

	if usePipes {
		SetPipes(&kc)
	}
	return &kc, nil
}

// SetPipes connects the out and error streams of a KubectlConfig
// to write to pipes.
func SetPipes(kc *KubectlConfig) {
	kc.StreamOut, kc.StreamOutWriter = io.Pipe()
	kc.StreamErr, kc.StreamErrWriter = io.Pipe()
	kc.Streams.Out = kc.StreamOutWriter
	kc.Streams.ErrOut = kc.StreamErrWriter
}

// SetLastAppliedConfigurationAnnotation applies the kubectl.kubernetes.io/last-applied-configuration annotation
// in order to calculate correct 3-way merges between object configuration file/configuration file,
// live object configuration/live configuration and declarative configuration writer/declarative writer
func SetLastAppliedConfigurationAnnotation(obj runtime.Object) error {
	err := util.CreateOrUpdateAnnotation(true, obj, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return fmt.Errorf("error while applying %s annotation on the "+
			"object: %v", v1.LastAppliedConfigAnnotation, err)
	}
	return nil
}

// Write logs the error output ignoring any errors as needed.
func (w *kErrWriter) Write(out []byte) (n int, err error) {
	o := string(out)
	for _, e := range w.ignore {
		if strings.Contains(string(out), e) {
			o = strings.ReplaceAll(o, e, "")
		}
	}
	if len(strings.TrimSpace(o)) > 0 {
		w.ErrBuf.Write([]byte(o))
	}

	return len(out), nil
}
