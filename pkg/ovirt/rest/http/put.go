// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"fmt"
	"io"
	"net/http"
)

// PutAbsURL puts an HTTP request and returns the body, status code, and an error
func (r REST) PutAbsURL(URL string, payload io.Reader, header *http.Header, contentLen int64) ([]byte, int, error) {
	body, statusCode, err := r.putPriv(URL, payload, header, contentLen)
	if err != nil {
		r.tokenResetter.ClearAccessToken()
	}
	return body, statusCode, err
}

// putPriv puts an HTTP request and returns the body, status code, and an error
func (r REST) putPriv(URL string, payload io.Reader, header *http.Header, contentLen int64) ([]byte, int, error) {
	req, err := http.NewRequest("PUT", URL, payload)
	if err != nil {
		return nil, 0, err
	}

	req.Header = *header
	req.ContentLength = contentLen

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp == nil {
		return nil, 0, fmt.Errorf("HTTP PUT to %s returned a nil response", URL)
	}

	defer resp.Body.Close()

	// 204 Success but no data from server
	if resp.StatusCode == 204 {
		return nil, 204, nil
	}

	// Always the body, it may have an error message
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if resp.StatusCode > 204 {
			return body, resp.StatusCode, fmt.Errorf("HTTP PUT to %s returned status code  %d", URL, resp.StatusCode)
		}
		return nil, resp.StatusCode, err
	}

	// Body was read ok but there is a response code error
	if resp.StatusCode > 204 {
		return body, resp.StatusCode, fmt.Errorf("HTTP PUT to %s returned status code %d.  Error Message: %s", URL, resp.StatusCode, body)
	}

	return body, resp.StatusCode, nil
}
