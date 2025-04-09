// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"sigs.k8s.io/yaml"
)

// rmBase is the base of a nested merge with a list
const rmBase = `
name: base
host:
  ip: 1.2.3.4
  name: foo
platform:
  vendor: company1
  os:
    name: linux
    patches:
    - version: 0.5.0
      date: 01/01/2020
`

// rmOverlay is the overlay of a nested merge with a list
const rmOverlay = `
host:
  name: bar
platform:
  os:
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

// rmMerged is the result of a nested merge
const rmMerged = `
name: base
host:
  ip: 1.2.3.4
  name: bar
platform:
  vendor: company1
  os:
    name: linux
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

func TestMergeOverrides(t *testing.T) {
	// Convert the base set of overrides into a map
	base := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(rmBase), &base)
	assert.NoError(t, err)

	// Covert the overlay set of overrides into a map
	overlay := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(rmOverlay), &overlay)
	assert.NoError(t, err)

	// Do recursive merge
	err = MergeMaps(base, overlay)
	assert.NoError(t, err)

	// Compare with the expected result
	mergedYaml, err := yaml.Marshal(base)
	assert.NoError(t, err)
	assert.YAMLEq(t, rmMerged, string(mergedYaml))
}
