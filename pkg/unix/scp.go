// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package unix

import (
	"errors"
	"fmt"
	"path/filepath"
)

// Scp defines the interface to run scp commands
type Scp interface {
	CopyFileToRemote(sourcePath string, destPath string) (string, string, error)
}

// ScpConfig specifies info needed to run scp and retry if it fails
type ScpConfig struct {
	MaxRetries int
	ConnectionConfig
}

// NewScp returns a Scp interface
func NewScp(sshURI string, sshKey string) Scp {
	return ScpConfig{
		MaxRetries: 2,
		ConnectionConfig: ConnectionConfig{
			URI:         sshURI,
			PubKeyFname: sshKey,
		},
	}
}

// CopyFileToRemote copies a file to the remote host with retries
func (r ScpConfig) CopyFileToRemote(sourcePath string, destPath string) (string, string, error) {
	maxRetries := r.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1 // default
	}
	for i := 1; i < maxRetries; i++ {
		stdout, stderr, err := r.copyFileToRemoteOnce(sourcePath, destPath)
		if err == nil {
			return stdout, stderr, nil
		}
		if i == maxRetries {
			return stdout, stderr, err
		}
	}
	return "", "", errors.New("scp failed")
}

// copyFileToRemoteOnce copies a file to the remote host
func (s ScpConfig) copyFileToRemoteOnce(sourcePath string, destPath string) (string, string, error) {
	destURI := fmt.Sprintf("%s:%s", s.URI, destPath)
	var a []string
	a = append(a, sourcePath)
	if s.PubKeyFname == "" {
		a = append(a, destURI)
	} else {
		if !filepath.IsAbs(s.PubKeyFname) {
			var err error
			s.PubKeyFname, err = filepath.Abs(s.PubKeyFname)
			if err != nil {
				return "", "", err
			}
		}
		a = append(a, "-i", s.PubKeyFname, s.URI)
	}

	e := NewCmdExecutor("scp", a...)
	err := e.Cmd.Run()
	return e.StdOutBuf.String(), e.StdErrBuf.String(), err
}
