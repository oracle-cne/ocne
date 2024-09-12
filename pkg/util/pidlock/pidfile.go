// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pidlock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/file"
)

const (
	PidFile = "ocne.pid"
)

// WaitFor attempts to take a lock that prevents multiple access
// across processes.
func WaitFor(timeout time.Duration) error {
	configDir, err := file.EnsureOcneDir()
	if err != nil {
		return err
	}
	pidFilePath := filepath.Join(configDir, PidFile)

	// Wait until it is possible to create the pidfile.  If it is
	// possible, then the pidfile must not exist and this process
	// owns it.  If it is not possible, and the reason is that the
	// file already exists, it is necessary to wait.
	var f *os.File
	until := time.Now().Add(timeout)
	for {
		f, err = os.OpenFile(pidFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.ModeExclusive)
		if err != nil {
			if os.IsExist(err) {
				if time.Now().After(until) {
					return fmt.Errorf("Could not get pidfile lock at %s", pidFilePath)
				}

				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}

		break
	}

	// Write the pid for this process into the file so it knows
	// that it owns it.
	f.Truncate(0)
	f.Write([]byte(strconv.Itoa(os.Getpid())))
	f.Chmod(0600)
	f.Close()

	return nil
}

// Drop releases a pid lock.
func Drop() error {
	pid := os.Getpid()

	hd, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	pidFilePath := filepath.Join(hd, constants.UserConfigDir, PidFile)

	f, err := os.Open(pidFilePath)
	if err != nil {
		return err
	}

	contents := make([]byte, 32)
	n, err := f.Read(contents)
	if err != nil {
		return err
	}

	contents = contents[:n]

	filePid, err := strconv.Atoi(string(contents))
	if err != nil {
		return err
	}

	if filePid != pid {
		return fmt.Errorf("Tried to drop a pid lock for process %d.  This process is %d", filePid, pid)
	}
	f.Close()

	err = os.Remove(pidFilePath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return nil
	}
	return nil
}
