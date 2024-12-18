// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

import (
	"bytes"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	"io"
	"net/http"
)

// UploadFile uploads an image to an endpoint
func UploadFile(ovcli *ovclient.Client, inUrl string, reader io.Reader, totalLen int64) error {
	const defChunkLen = 10 * 1024 * 1024
	var chunkLen int64 = defChunkLen
	var start int64
	var end int64 = -1

	remainingLen := totalLen
	for remainingLen > 0 {
		if chunkLen > remainingLen {
			chunkLen = remainingLen
		}

		// On the last request we want to close the connection and flush.
		// This will close imageio backend so we can deactivate the volume
		// on block storage. On all other requests we want to avoid
		// flushing for better performance.
		// see https://github.com/oVirt/ovirt-engine/blob/master/frontend/webadmin/modules/uicommonweb/src/main/java/org/ovirt/engine/ui/uicommonweb/models/storage/UploadImageHandler.java
		var url string
		if chunkLen == remainingLen {
			url = inUrl + "?close=y"
		} else {
			url = inUrl + "?flush=n"
		}

		// start and end are 0-based and inclusive
		// see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Range
		start = end + 1
		end = start + chunkLen - 1
		remainingLen -= chunkLen
		if err := uploadChunk(ovcli, url, reader, chunkLen, start, end, totalLen); err != nil {
			return err
		}
	}

	return nil
}

func uploadChunk(ovcli *ovclient.Client, url string, reader io.Reader, chunkLen int64, start int64, end int64, totalLen int64) error {
	// TEMP - until i figure out how to prevent http from closing the file stream
	b := make([]byte, chunkLen)
	n, err := io.ReadFull(reader, b)
	if err != nil {
		return err
	}
	if n != int(chunkLen) {
		return fmt.Errorf("upload chunk: expected %d bytes, got %d bytes", chunkLen, n)
	}

	h := &http.Header{}
	//	ovcli.REST.HeaderAcceptJSON(h)
	ovcli.REST.HeaderContentLen(h, chunkLen)
	ovcli.REST.HeaderContentOctet(h)
	ovcli.REST.HeaderContentRange(h, start, end, totalLen)
	ovcli.REST.HeaderBearerToken(h, ovcli.AccessToken)
	ovcli.REST.HeaderNoCache(h)
	_, statusCode, err := ovcli.REST.Put(url, bytes.NewReader(b), h, chunkLen)
	if err != nil {
		err = fmt.Errorf("Error calling HTTP PUT to upload a file: %v", err)
		return err
	}

	if statusCode != 200 && statusCode != 201 && statusCode != 202 {
		err = fmt.Errorf("Error calling HTTP PUT to upload a file %v", statusCode)
		return err
	}

	return nil
}
