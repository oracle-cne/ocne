// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

// ListRegions - list regions
func ListRegions(profile string) ([]string, error) {
	dcp := common.CustomProfileConfigProvider("", profile)
	idp, err := identity.NewIdentityClientWithConfigurationProvider(dcp)
	if err != nil {
		return []string{}, err
	}

	response, err := idp.ListRegions(context.Background())
	if err != nil {
		return []string{}, err
	}

	regionNames := []string{}
	for _, region := range response.Items {
		regionNames = append(regionNames, *region.Name)
	}

	return regionNames, nil
}
