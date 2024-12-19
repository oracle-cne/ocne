// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"net/http"
)

// DeleteDisk deletes a disk resource.
func DeleteDisk(ovcli *ovclient.Client, diskID string) error {
	path := fmt.Sprintf("/api/disks/%s", diskID)

	// call the server to get the imagetransfer
	h := &http.Header{}
	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentJSON(h)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	_, err := ovcli.REST.Delete(path, h)
	if err != nil {
		err = fmt.Errorf("Error doing HTTP DELETE of the disk %s: %v", diskID, err)
		return err
	}

	return nil
}
