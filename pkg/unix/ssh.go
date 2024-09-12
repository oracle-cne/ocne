// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package unix

import (
	"errors"
	"fmt"
	"path/filepath"
)

// ConnectionConfig specifies info needed to run ssh/scp
type ConnectionConfig struct {
	URI         string
	PubKeyFname string
}

// Ssh interface implements ssh functions
type Ssh interface {
	Run(args ...string) (string, string, error)
}

// SshConfig specifies config needed to run ssh/scp and retry if it fails
type SshConfig struct {
	MaxRetries int
	ConnectionConfig
	*SshTunnelConfig
}

type SshTunnelConfig struct {
	KubernetesIP   string
	KubernetesPort string
}

// NewSsh returns an Ssh interface
func NewSsh(sshURI string, sshKey string) Ssh {
	return SshConfig{
		MaxRetries: 2,
		ConnectionConfig: ConnectionConfig{
			URI:         sshURI,
			PubKeyFname: sshKey,
		},
	}
}

// NewSshTunnel returns an SshTunne interface
func NewSshTunnel(sshURI string, sshKey string, kubernetesIP string, kubernetesPort string) SshConfig {
	return SshConfig{
		MaxRetries: 2,
		ConnectionConfig: ConnectionConfig{
			URI:         sshURI,
			PubKeyFname: sshKey,
		},
		SshTunnelConfig: &SshTunnelConfig{
			KubernetesIP:   kubernetesIP,
			KubernetesPort: kubernetesPort,
		},
	}
}

// Run implements CmdExecutor.Run to run ssh with retries
func (s SshConfig) Run(args ...string) (string, string, error) {
	maxRetries := s.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1 // default
	}
	for i := 1; i < maxRetries; i++ {
		stdout, stderr, err := s.runOnce(args...)
		if err == nil {
			return stdout, stderr, nil
		}
		if i == maxRetries {
			return stdout, stderr, err
		}
	}
	return "", "", errors.New("ssh failed")
}

// runOnce runs the ssh command once
func (s SshConfig) runOnce(args ...string) (string, string, error) {
	var a []string
	if len(s.PubKeyFname) > 0 {
		if !filepath.IsAbs(s.PubKeyFname) {
			var err error
			s.PubKeyFname, err = filepath.Abs(s.PubKeyFname)
			if err != nil {
				return "", "", err
			}
		}
		a = append(a, "-i", s.PubKeyFname)
	}
	if s.SshTunnelConfig != nil {
		a = append(a, "-L", fmt.Sprintf("%s:%s:%s",
			s.SshTunnelConfig.KubernetesPort, s.SshTunnelConfig.KubernetesIP, s.SshTunnelConfig.KubernetesPort))
	}
	a = append(a, s.URI)
	a = append(a, args...)

	e := NewCmdExecutor("ssh", a...)
	err := e.Cmd.Run()
	return e.StdOutBuf.String(), e.StdErrBuf.String(), err
}
