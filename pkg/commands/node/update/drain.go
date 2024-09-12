// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"fmt"
	"k8s.io/kubectl/pkg/cmd/drain"
	kutil "k8s.io/kubectl/pkg/cmd/util"
	"github.com/oracle-cne/ocne/pkg/k8s/kubectl"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"time"
)

// cordonAndDrainNode cordons and drains a node
func cordonAndDrainNode(o *UpdateOptions, kc *kubectl.KubectlConfig) error {
	waitMsg := fmt.Sprintf("Draining node %s", o.NodeName)

	// Default to 30 min timeout
	timeout := "30m"
	if o.Timeout != "" {
		_, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return fmt.Errorf("Invalid timeout format %s", o.Timeout)
		}
		timeout = o.Timeout
	}

	waitors := []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				cmd := drain.NewCmdDrain(kutil.NewFactory(kutil.NewMatchVersionFlags(kc.ConfigFlags)), kc.Streams)

				args := []string{}
				if o.DeleteEmptyDir {
					args = append(args, "--delete-emptydir-data")
				}
				if o.DisableEviction {
					args = append(args, "--disable-eviction")
				}
				args = append(args,
					"--force",
					"--ignore-daemonsets",
					fmt.Sprintf("--timeout=%s", timeout),
					o.NodeName)
				cmd.SetArgs(args)
				return cmd.Execute()
			},
		},
	}
	haveError := logutils.WaitFor(logutils.Info, waitors)
	if haveError == true {
		return fmt.Errorf("Error draining node %s: %v", o.NodeName, waitors[0].Error)
	}
	return nil
}

// uncordonNode un-cordons a node
func uncordonNode(o *UpdateOptions, kc *kubectl.KubectlConfig) error {
	var retErr error
	waitMsg := fmt.Sprintf("Un-cordoning node %s", o.NodeName)

	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		&logutils.Waiter{
			Message: waitMsg,
			WaitFunction: func(i interface{}) error {
				cmd := drain.NewCmdUncordon(kutil.NewFactory(kutil.NewMatchVersionFlags(kc.ConfigFlags)), kc.Streams)
				cmd.SetArgs([]string{o.NodeName})
				retErr = cmd.Execute()
				return retErr
			},
		},
	})
	if haveError == true {
		return fmt.Errorf("Timeout un-cordoning node %s", o.NodeName)
	}
	return nil
}
