// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubectl

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
	"k8s.io/kubectl/pkg/cmd/cp"
	kutil "k8s.io/kubectl/pkg/cmd/util"
)

type FilePath struct {
	RemotePath string
	LocalPath  string
}

// CopyConfig contains information used to copy to and from the pod
type CopyConfig struct {
	*KubectlConfig
	FilePaths []FilePath
	PodName   string
}

// CopyFilesToPod copies one or more files to a pod. This function stops if an error occurs
// copying any of the files, and the remaining files are not copied.
func CopyFilesToPod(c *CopyConfig, waitMsg string) error {
	defer func() {
		if !util.IsNil(c.StreamOutWriter) {
			c.StreamOutWriter.Close()
		}
		if !util.IsNil(c.StreamErrWriter) {
			c.StreamErrWriter.Close()
		}
	}()

	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				fatalErr := false
				kutil.BehaviorOnFatal(func(s string, c int) {
					fatalErr = true
				})

				//	kubectl cp <localfilePath> <namespace>/<pod>:<filePath>
				for _, paths := range c.FilePaths {
					cmd := cp.NewCmdCp(kutil.NewFactory(kutil.NewMatchVersionFlags(c.ConfigFlags)), c.Streams)
					s2 := fmt.Sprintf("%s/%s:%s", c.Namespace, c.PodName, paths.RemotePath)
					args := []string{"--retries=5", "--no-preserve", paths.LocalPath, s2}
					cmd.SetArgs(args)
					logutils.LogDebugAndErrors(&logutils.Stream{
						Reader: c.StreamOut,
						Filter: nil,
						Arg:    nil,
					},
						&logutils.Stream{
							Reader: c.StreamErr,
							Filter: filter,
							Arg:    c.IgnoreErrors,
						})
					if err := cmd.Execute(); err != nil {
						return err
					}
					if fatalErr {
						return fmt.Errorf("Error running script: %s", c.ErrBuf.String())
					}
				}
				return nil
			},
		},
	})
	if haveError == true {
		return fmt.Errorf("Error copying file to pod %s/%s", c.Namespace, c.PodName)
	}
	return nil
}

// CopyFilesFromPod copies one or more remote files in a pod to local files. This function stops if an error occurs
// copying any of the files, and the remaining files are not copied.
func CopyFilesFromPod(c *CopyConfig, waitMsg string) error {
	haveError := logutils.WaitForWithLevel(logutils.Debug, log.DebugLevel, []*logutils.Waiter{
		{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				fatalErr := false
				fatalStr := ""
				kutil.BehaviorOnFatal(func(s string, c int) {
					fatalErr = true
					fatalStr = s
				})

				// kubectl cp <namespace>/<pod>:<filePath> <localfilePath>
				for _, paths := range c.FilePaths {
					cmd := cp.NewCmdCp(kutil.NewFactory(kutil.NewMatchVersionFlags(c.ConfigFlags)), c.Streams)
					s1 := fmt.Sprintf("%s/%s:%s", c.Namespace, c.PodName, paths.RemotePath)
					args := []string{"--retries=5", s1, paths.LocalPath}
					cmd.SetArgs(args)
					if err := cmd.Execute(); err != nil {
						return err
					}
					if fatalErr {
						return fmt.Errorf("%s", fatalStr)
					}
				}
				return nil
			},
		},
	})
	if haveError {
		return fmt.Errorf("Error copying file from pod %s/%s", c.Namespace, c.PodName)
	}
	return nil
}
