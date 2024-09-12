// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/portforward"
	"k8s.io/kubectl/pkg/cmd/util"
)

// Logger used to filter some  kubectl output from appearing on the console
var myLogger klog.Logger

// PortForwardSpec contains the information needed to do the port forward
type PortForwardSpec struct {
	// ServiceNsn is the namespace and name of the service which the target of the port forward.
	// Required
	ServiceNsn types.NamespacedName

	// LocalPort is the port on the local system (where the CLI is) where HTTP requests will be sent
	// This is normally left blank and the port-forwarding code will assign an ephemeral port, which
	// is returned to the caller when the port-forward is started. Only set it if you really
	// want to use a specific port.
	// Optional
	LocalPort string

	// RemotePort is the service port which will receive forwarded traffic.
	// Required
	RemotePort string
}

// Result is the data sent from the goroutine over the channel
type Result struct {
	// Err will be non-nil if the goroutine encountered a problem
	Err error

	// LocalPort is the local port that the port-forwarder is listening on
	LocalPort int
}

// outWriter implements io.Write for output
type outWriter struct {
	// mutex is used to lock the channel to prevent concurrent writes from goroutine, outWriter, and errWriter.  This may not
	// be needed, but just to be safe
	mutex *sync.Mutex

	// buffer is where output text will be written (i.e. similar to stdout)
	buffer bytes.Buffer

	// channel is the channel used to send the result
	channel *chan Result
}

// errWriter implements io.Write for error output
type errWriter struct {
	// mutex is used to lock the channel to prevent concurrent writes from goroutine, outWriter, and errWriter.  This may not
	// be needed, but just to be safe
	mutex *sync.Mutex

	// channel is the channel used to send the result
	channel *chan Result
}

// PortForwardToService does a port forward to a service and returns the local port
func PortForwardToService(kubeConfigPath string, serviceNsn types.NamespacedName, remotePort string) (int, error) {
	// create a KubeInfo
	restConfig, _, err := client.GetKubeClient(kubeConfigPath)
	if err != nil {
		return 0, err
	}
	path, _, err := client.GetKubeConfigLocation(kubeConfigPath)
	if err != nil {
		return 0, err
	}
	kubeInfo := client.KubeInfo{KubeconfigPath: path, RestConfig: restConfig}

	// the localPort will be assigned by the port-forward code
	spec := PortForwardSpec{
		ServiceNsn: serviceNsn,
		RemotePort: remotePort,
	}
	log.Debugf("PFS: %+v", spec)

	// start the port-forward to the service
	return StartPortForwardToService(&kubeInfo, &spec)
}

// StartPortForwardToService starts a port-forward to a service in the background, then returns the local port.
// The caller can then send data to http://localhost:<localport>
// The port-forwarding session remains until the CLI exits.
func StartPortForwardToService(kubeInfo *client.KubeInfo, forwardSpec *PortForwardSpec) (int, error) {
	config := &genericclioptions.ConfigFlags{
		KubeConfig:    &kubeInfo.KubeconfigPath,
		BearerToken:   &kubeInfo.RestConfig.BearerToken,
		Namespace:     &forwardSpec.ServiceNsn.Namespace,
		Insecure:      retBool(true),
		TLSServerName: &kubeInfo.KubeApiServerIP,
	}

	c := make(chan Result, 1)

	// goroutine starts the port forwarder with custom writers.
	// The writers will send the result to main thread
	go func(*genericclioptions.ConfigFlags, *PortForwardSpec) {
		mutex := sync.Mutex{}
		streams := genericiooptions.IOStreams{
			In: os.Stdin,
			Out: &outWriter{
				mutex:   &mutex,
				buffer:  bytes.Buffer{},
				channel: &c,
			},
			ErrOut: &errWriter{
				mutex:   &mutex,
				channel: &c,
			},
		}

		// start port-forward, this is a blocking call except when there is an error
		err := executePortForwardCmd(config, forwardSpec, streams)
		if err != nil {
			mutex.Lock()
			defer mutex.Unlock()
			c <- Result{Err: err}
			return
		}
	}(config, forwardSpec)

	// wait for goroutine to send result
	result := <-c
	return result.LocalPort, result.Err
}

// suppressKubectlConnectionError - filter some errors from kubectl from appearing on the console
func suppressKubectlConnectionError() {
	klog.SetLoggerWithOptions(myLogger, klog.ContextualLogger(true), klog.WriteKlogBuffer(func(msg []byte) {
		if !bytes.Contains(msg, []byte("connection reset")) {
			log.Errorf("%s", msg)
		}
	}))
}

// executePortForwardCmd does the equivalent of kubectl port-forward -n ocne-system service/<service-name> :<remote-port>
func executePortForwardCmd(config *genericclioptions.ConfigFlags, forwardSpec *PortForwardSpec, streams genericiooptions.IOStreams) error {
	cmd := portforward.NewCmdPortForward(util.NewFactory(config), streams)
	suppressKubectlConnectionError()

	s1 := fmt.Sprintf("service/%s", forwardSpec.ServiceNsn.Name)
	s2 := fmt.Sprintf("%s:%s", forwardSpec.LocalPort, forwardSpec.RemotePort)
	args := []string{s1, s2}
	log.Debugf("Port Forward Cmd: %+v", args)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// extractLocalPort extracts the local port used for port forwarding
func extractLocalPort(in string) string {
	s := strings.ToLower(in)
	re := regexp.MustCompile(`forwarding from \d*\.\d*\.\d*\.\d*:(\d*)\s->\s\d*`)
	a := re.FindAllStringSubmatch(s, -1)
	if len(a) != 1 {
		return ""
	}
	if len(a[0]) != 2 {
		return ""
	}
	return a[0][1]
}

// Write extracts the local port used by the port forwarder and sends it over the channel.
// The channel is only used once then cleared.
func (w *outWriter) Write(out []byte) (n int, err error) {
	// If the result has already been sent to the channel then nothing to do
	if w.channel == nil {
		return
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()

	n, err = w.buffer.Write(out)
	if err != nil {
		*w.channel <- Result{Err: err}
		// Only send once to channel
		w.channel = nil
		return
	}

	// Check if we have the port
	sPort := extractLocalPort(w.buffer.String())
	if sPort == "" {
		return
	}

	// The output contains the port, send it to the main thread
	port, err := strconv.Atoi(sPort)
	*w.channel <- Result{Err: err, LocalPort: port}
	w.channel = nil

	return n, err
}

// Write logs the error output and writes an error to the channel.
// Lock the channel just in case outWrite and errWrite are concurrent.
// The channel is only used once then cleared.
func (w *errWriter) Write(out []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	log.Errorf("Error doing port forward : %s ", string(out))

	if w.channel != nil {
		*w.channel <- Result{Err: err}
		w.channel = nil
	}
	return n, err
}

func retBool(b bool) *bool {
	return &b
}
