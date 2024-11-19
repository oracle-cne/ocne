// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"io"
)

type analyzeParams struct {
	// rootDir is the directory that has dumped resources
	rootDir string

	// clusterDir is the directory that has dumped Kubernetes cluster resources
	clusterDir string

	// nameSpacesDir is the directory that has dumped Kubernetes namespaces
	nameSpacesDir string

	// clusterWideDir is the directory that has dumped Kubernetes cluster-wide resources
	clusterWideDir string

	// nodesDir is the directory that has dumped node resources
	nodesDir string

	// verbose controls displaying of analyze details like events
	verbose bool

	// writer that writes the analyzer output
	writer io.Writer

	// isJSON if the Kubernetes resources being analyzed are in JSON format
	isJSON bool
}
