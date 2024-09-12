// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package unix

import (
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os/exec"
)

// ChanData is the data passed over the channel.  Type and fields must be upper case
type ChanData struct {
	// Err contain error formatted with stderr or nil
	Err error

	// Cmd is needed to kill process.  Don't access any other fields from main thread
	Cmd *exec.Cmd
}

// StartSshTunnel starts a tunnel in the background.  Return exec.Cmd so the caller can stop the tunnel
func StartSshTunnel(streams genericclioptions.IOStreams, sshURI string, sshKey string, kubernetesIP string, kubernetesPort string) (*exec.Cmd, error) {
	c := make(chan ChanData, 1)

	// goroutine starts tunnel and sends result to main thread
	go func() {
		// start the tunnel but don't wait
		e, err := startTunnel(streams, sshURI, sshKey, kubernetesIP, kubernetesPort)
		c <- ChanData{
			Err: err,
			Cmd: e.Cmd,
		}
		if err != nil {
			return
		}

		// wait for the tunnel to terminate.  This should only return when the tunnel gets stopped
		e.Cmd.Wait()
	}()

	// Wait for goroutine to send data
	result := <-c
	return result.Cmd, result.Err

}

func startTunnel(streams genericclioptions.IOStreams, sshURI string, sshKey string, kubernetesIP string, kubernetesPort string) (*CmdExecutor, error) {
	tunnel := fmt.Sprintf("%s:%s:%s", kubernetesPort, kubernetesIP, kubernetesPort)
	e := NewCmdExecutor("ssh", "-tt", "-L", tunnel, sshURI)
	err := e.Cmd.Start()
	if err != nil {
		err = fmt.Errorf("Failed running ssh tunnel: %v %v", err, e.StdErrBuf)
	}
	return e, err
}

func GetTunnelCommand(sshURI string, kubernetesIP string, kubernetesPort string) string {
	return fmt.Sprintf("ssh -L %s:%s:%s %s", kubernetesPort, kubernetesIP, kubernetesPort, sshURI)
}

func StopTunnel(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil && cmd.Process.Pid > 0 {
		cmd.Process.Kill()
	}
}
