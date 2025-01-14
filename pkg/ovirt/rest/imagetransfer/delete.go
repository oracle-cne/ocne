// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"net/http"
)

// DeleteImageTransfer deletes an imagetransfer resource.
func DeleteImageTransfer(ovcli *ovclient.Client, transferID string) error {
	path := fmt.Sprintf("/api/imagetransfers/%s", transferID)

	// call the server to get the imagetransfer
	h := &http.Header{}
	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentJSON(h)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	_, err := ovcli.REST.Delete(path, h)
	if err != nil {
		return err
	}

	return nil
}
