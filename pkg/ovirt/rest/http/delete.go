// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"fmt"
	"io"
	"net/http"
)

// Delete deletes a REST resource
func (r REST) Delete(path string, header *http.Header) (int, error) {
	statusCode, err := r.deletePriv(path, header)
	if err != nil {
		r.tokenResetter.ClearAccessToken()
	}
	return statusCode, err
}

// deletePriv deletes a REST resource
func (r REST) deletePriv(path string, header *http.Header) (int, error) {
	URL := resolveURL(r.endpointURL, path)

	req, err := http.NewRequest("DELETE", URL, nil)
	if err != nil {
		return 0, err
	}

	req.Header = *header

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, fmt.Errorf("HTTP DELETE to endpoint %s received a nil response", URL)
	}

	defer resp.Body.Close()

	// 204 Success but no data from server
	if resp.StatusCode == 204 {
		return resp.StatusCode, nil
	}

	// Always the body, it may have an error message
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if resp.StatusCode > 204 {
			return resp.StatusCode, fmt.Errorf("HTTP DELETE to %s returned status code  %d", URL, resp.StatusCode)
		}
		return resp.StatusCode, err
	}

	// Body was read ok but there is a response code error
	if resp.StatusCode > 204 {
		return resp.StatusCode, fmt.Errorf("HTTP DELETE to %s returned status code %d.  Error Message: %s", URL, resp.StatusCode, body)
	}

	return resp.StatusCode, nil
}
