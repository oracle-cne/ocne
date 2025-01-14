// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"fmt"
	"io"
	"net/http"
)

func (r REST) Get(accessToken string, path string) ([]byte, error) {
	body, err := r.getPriv(accessToken, path)
	if err != nil {
		r.tokenResetter.ClearAccessToken()
	}
	return body, err
}

func (r REST) getPriv(accessToken string, path string) ([]byte, error) {
	URL := resolveURL(r.endpointURL, path)

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("HTTP GET to %s returned a nil response", URL)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP GET to %s returned status code  %d", URL, resp.StatusCode)
	}

	defer resp.Body.Close()

	// Read the data
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
