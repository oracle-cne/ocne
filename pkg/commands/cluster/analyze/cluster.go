// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

func analyzeCluster(p *analyzeParams) error {
	if err := analyzeClusterNodes(p); err != nil {
		return err
	}

	if err := analyzeEvents(p); err != nil {
		return err
	}
	return nil
}
