// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubectl

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	kexec "k8s.io/kubectl/pkg/cmd/exec"
	kutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

func filter(s string, ignoresIface interface{}) bool {
	ignores, _ := ignoresIface.([]string)
	for _, i := range ignores {
		log.Debugf("Checking ignore list for: %s", s)
		if strings.Contains(s, i) {
			return false
		}
	}
	return true
}

func RunCommand(kc *KubectlConfig, podName string, cmdArgs ...string) error {
	// kubectl exec ...
	cmd := kexec.NewCmdExec(kutil.NewFactory(kutil.NewMatchVersionFlags(kc.ConfigFlags)), kc.Streams)
	args := []string{podName, "--"}
	args = append(args, cmdArgs...)

	cmd.SetArgs(args)

	// must override BehaviorOnFatal or else kubectl just exits the process if the script returns non-zero
	fatalErr := false
	var fatalErrError error
	kutil.BehaviorOnFatal(func(s string, c int) {
		// Filter out any ignored errors
		if !filter(s, kc.IgnoreErrors) {
			log.Debugf("Ignoring fatal error: %s", s)
			return
		}
		fatalErrError = fmt.Errorf("Error running script: %s", s)
		fatalErr = true
	})

	// execute the script
	logutils.LogDebugAndErrors(&logutils.Stream{
		Reader: kc.StreamOut,
		Filter: nil,
		Arg:    nil,
	},
		&logutils.Stream{
			Reader: kc.StreamErr,
			Filter: filter,
			Arg:    kc.IgnoreErrors,
		})
	err := cmd.Execute()
	if !util.IsNil(kc.StreamOutWriter) {
		kc.StreamOutWriter.Close()
	}
	if !util.IsNil(kc.StreamErrWriter) {
		kc.StreamErrWriter.Close()
	}
	if err != nil || fatalErr {
		var retErr error
		if kc.ErrBuf != nil && len(kc.ErrBuf.String()) > 0 {
			retErr = fmt.Errorf("Error running script: %s", kc.ErrBuf.String())
		} else if err != nil {
			retErr = err
		} else if fatalErrError != nil {
			retErr = fatalErrError
		} else {
			retErr = fmt.Errorf("Error running script")
		}
		return retErr
	}

	return nil
}

// RunScript runs a script on a pod
func RunScript(kc *KubectlConfig, podName string, scriptDir string, scriptName string) error {
	return RunCommand(kc, podName, "sh", "-c", fmt.Sprintf("export \"PATH=%s:$PATH\"; cd %s; %s", scriptDir, scriptDir, scriptName))
}
