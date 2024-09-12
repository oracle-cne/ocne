// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package unix

// RunBash runs a bash script
func RunBash(args ...string) (string, string, error) {
	e := NewCmdExecutor("bash", args...)
	err := e.Cmd.Run()
	if err != nil {
		return e.StdOutBuf.String(), e.StdErrBuf.String(), err
	}
	return e.StdOutBuf.String(), "", err
}

// RunBashChangeDirectory runs a bash script with the option of changing the directory that it operates in
func RunBashChangeDirectory(dir string, args ...string) (string, string, error) {
	e := NewCmdExecutor("bash", args...)
	e.Cmd.Dir = dir
	err := e.Cmd.Run()
	if err != nil {
		return e.StdOutBuf.String(), e.StdErrBuf.String(), err
	}
	return e.StdOutBuf.String(), "", err
}
