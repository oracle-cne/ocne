// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

// GetImageTransfer gets an imagetransfer resource.
func GetImageTransfer(ovcli *ovclient.Client, transferID string) (*ImageTransfer, error) {
	path := fmt.Sprintf("/api/imagetransfers/%s", transferID)

	// call the server to get the imagetransfer
	body, err := ovcli.REST.Get(ovcli.AccessToken, path)
	if err != nil {
		err = fmt.Errorf("Error doing HTTP GET: %v", err)
		log.Error(err)
		return nil, err
	}

	iTran := &ImageTransfer{}
	err = json.Unmarshal(body, iTran)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling ImageTransfer: %v", err)
		log.Error(err)
		return nil, err
	}

	return iTran, nil
}
