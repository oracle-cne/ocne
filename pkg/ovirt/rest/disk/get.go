// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"k8s.io/apimachinery/pkg/util/json"
)

// GetDisk gets an disk resource.
func GetDisk(ovcli *ovclient.Client, transferID string) (*Disk, error) {
	path := fmt.Sprintf("/api/disks/%s", transferID)

	// call the server to get the disk
	body, err := ovcli.REST.Get(ovcli.AccessToken, path)
	if err != nil {
		err = fmt.Errorf("Error doing HTTP GET: %v", err)
		return nil, err
	}

	disk := &Disk{}
	err = json.Unmarshal(body, disk)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling Disk: %v", err)
		return nil, err
	}

	return disk, nil
}
