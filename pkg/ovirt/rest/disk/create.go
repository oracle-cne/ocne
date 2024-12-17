// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"k8s.io/apimachinery/pkg/util/json"

	"net/http"
)

// CreateDisk creates a disk in a storage domain. An ock image is subsequently uploaded to that image.
func CreateDisk(ovcli *ovclient.Client, req *CreateDiskRequest) (*Disk, error) {
	const path = "/api/disks"

	jsonPayload, err := json.Marshal(req)
	if err != nil {
		err := fmt.Errorf("Error marshalling CreateDiskRequest %s: %v", req.Name, err)
		return nil, err
	}

	// call the server to create the VM
	h := &http.Header{}
	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentJSON(h)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	body, statusCode, err := ovcli.REST.Post(path, bytes.NewReader(jsonPayload), h)
	if err != nil {
		err = fmt.Errorf("Error calling HTTP POST to create an oVirt disk: %v", err)
		return nil, err
	}

	if statusCode != 200 && statusCode != 201 && statusCode != 202 {
		err = fmt.Errorf("Error calling HTTP POST to create an oVirt disk returned status code %v", statusCode)
		return nil, err
	}

	disk := &Disk{}
	err = json.Unmarshal(body, disk)
	if err != nil {
		err = fmt.Errorf("Error unmarshalling create oVirt disk response: %v", err)
		return nil, err
	}

	return disk, nil
}
