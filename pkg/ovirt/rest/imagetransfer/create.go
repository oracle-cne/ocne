// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

import (
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"

	"net/http"
)

// CreateImageTransferRequest creates an image transfer resource needed to upload an image to a disk
func CreateImageTransfer(ovcli *ovclient.Client, req *CreateImageTransferRequest) (*ImageTransfer, error) {
	const path = "/api/imagetransfers"

	jsonPayload, err := json.Marshal(req)
	if err != nil {
		log.Errorf("Error marshalling CreateImageTransferRequest for disk %s: %v", req.Disk.Id, err)
		return nil, err
	}

	// call the server to create the ImageTransfer request
	h := &http.Header{}
	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentJSON(h)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	body, statusCode, err := ovcli.REST.Post(path, bytes.NewReader(jsonPayload), h)
	if err != nil {
		err = fmt.Errorf("Error calling HTTP POST to create an ImageTransfer resource: %v", err)
		return nil, err
	}

	if statusCode != 200 && statusCode != 201 && statusCode != 202 {
		err = fmt.Errorf("Error calling HTTP POST to create an ImageTransfer resource %v", statusCode)
		return nil, err
	}

	imTran := &ImageTransfer{}
	err = json.Unmarshal(body, imTran)
	if err != nil {
		err = fmt.Errorf("Error unmarshalling create ImageTransfer respnose: %v", err)
		log.Error(err)
		return nil, err
	}

	return imTran, nil
}
