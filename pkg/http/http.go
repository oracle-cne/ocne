// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPGet performs an HTTP get to the chart service via the port-forward on the local host
// An optional helmChartFileName can be passed in to download that file
func HTTPGet(uri string) ([]byte, error) {
	cli := &http.Client{}
	resp, err := cli.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("Failed HTTP GET to the catalog service: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed HTTP GET to the catalog service. Status code %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}
