// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"bufio"
	"bytes"
	"strings"

	"k8s.io/client-go/rest"
)

// ApplyResources creates resources in a cluster if the resource does not
// already exist. If the resource already exists, it is not modified.
func ApplyResources(restConfig *rest.Config, clusterResources string) error {
	resources, err := Unmarshall(bufio.NewReader(bytes.NewBufferString(clusterResources)))
	if err != nil {
		return err
	}

	for _, r := range resources {
		err = CreateResourceIfNotExist(restConfig, &r)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}
	}

	return nil
}
