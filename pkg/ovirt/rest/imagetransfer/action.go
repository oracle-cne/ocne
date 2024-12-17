// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

import (
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"net/http"
)

// DoImageTransferAction post an action to the image transfer resource
func DoImageTransferAction(ovcli *ovclient.Client, transferID string, action string) error {
	path := fmt.Sprintf("/api/imagetransfers/%s/%s", transferID, action)

	// call the server to update the ImageTransfer
	h := &http.Header{}
	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentJSON(h)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	_, statusCode, err := ovcli.REST.Post(path, bytes.NewReader(nil), h)
	if err != nil {
		err = fmt.Errorf("Error calling HTTP POST to update an ImageTransfer resource: %v", err)
		return err
	}

	if statusCode != 200 && statusCode != 201 && statusCode != 202 {
		err = fmt.Errorf("Error calling HTTP POST to update an ImageTransfer resource %v", statusCode)
		return err
	}

	return nil
}
