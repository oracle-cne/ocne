// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package unix

import (
	"bytes"
	"os/exec"
)

type CmdExecutor struct {
	*exec.Cmd
	StdOutBuf bytes.Buffer
	StdErrBuf bytes.Buffer
}

// NewCmdExecutor creates a new CmdExecutor
func NewCmdExecutor(cmdName string, args ...string) *CmdExecutor {
	e := CmdExecutor{
		Cmd:       exec.Command(cmdName, args...),
		StdOutBuf: bytes.Buffer{},
		StdErrBuf: bytes.Buffer{},
	}
	e.Cmd.Stdout = &e.StdOutBuf
	e.Cmd.Stderr = &e.StdErrBuf
	return &e
}
