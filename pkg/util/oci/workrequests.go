// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/workrequests"

	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

type workRequestWait struct {
	workRequestId   string
	profile         string
	percentComplete float32
	status          workrequests.WorkRequestStatusEnum
	prefix          string
	mutex           sync.RWMutex
}

// GetWorkRequestStatus gives a summary of a work request.  Specifically, it
// returns the current status and completion percentage.
func GetWorkRequestStatus(workRequestId string, profile string) (workrequests.WorkRequestStatusEnum, float32, error) {
	ctx := context.Background()
	wrc, err := workrequests.NewWorkRequestClientWithConfigurationProvider(common.CustomProfileConfigProvider("", profile))
	if err != nil {
		return "", 0, err
	}

	req := workrequests.GetWorkRequestRequest{
		WorkRequestId: &workRequestId,
	}
	resp, err := wrc.GetWorkRequest(ctx, req)
	if err != nil {
		return "", 0, err
	}
	return resp.WorkRequest.Status, *resp.WorkRequest.PercentComplete, nil
}

// WaitForWorkRequest waits for a work request to complete, while pretty
// printing the progress.  If the work request fails, then an error
// is returned.
func WaitForWorkRequest(workRequestId string, prefix string, profile string) error {
	return WaitForWorkRequests(map[string]string{workRequestId: prefix}, profile)
}

func waitForWorkRequest(wIface interface{}) error {
	for {
		w, _ := wIface.(*workRequestWait)
		status, complete, err := GetWorkRequestStatus(w.workRequestId, w.profile)
		if err != nil {
			return err
		}

		w.mutex.Lock()
		w.percentComplete = complete
		w.status = status
		w.mutex.Unlock()

		switch w.status {
		case workrequests.WorkRequestStatusFailed:
			return fmt.Errorf("Work request failed")
		case workrequests.WorkRequestStatusSucceeded:
			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}

func workRequestProgress(wIface interface{}) string {
	w, _ := wIface.(*workRequestWait)
	return fmt.Sprintf("%s: %s", w.prefix, logutils.ProgressBar(w.percentComplete))
}

// WaitForWorkRequestswaits for a set of work requests, to complete,
// while pretty printing the progress.  If any work requests fail, an
// error is returned.
func WaitForWorkRequests(requests map[string]string, profile string) error {
	var waits []*logutils.Waiter
	for id, msg := range requests {
		w := workRequestWait{
			workRequestId: id,
			prefix:        msg,
			profile:       profile,
		}
		waits = append(waits, &logutils.Waiter{
			Args:            &w,
			WaitFunction:    waitForWorkRequest,
			MessageFunction: workRequestProgress,
		})
	}

	failed := logutils.WaitFor(logutils.Info, waits)
	if failed {
		return fmt.Errorf("Work request failed")
	}
	return nil
}
