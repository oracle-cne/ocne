// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

// Takes a path to a compartment name in the format
// compartment1/compartment2/compartment3 and locates
// the OCID of the indicated compartment by following
// the path.
func GetCompartmentId(compartmentName string, profile string) (string, error) {
	// If the name is already an OCID, just hand it back
	if strings.HasPrefix(compartmentName, "ocid1.compartment") {
		return compartmentName, nil
	}

	elements := strings.Split(compartmentName, "/")

	if len(elements) < 1 {
		return "", fmt.Errorf("\"%s\" is not a valid compartment name", compartmentName)
	}

	dcp := common.CustomProfileConfigProvider("", profile)
	idp, err := identity.NewIdentityClientWithConfigurationProvider(dcp)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	// The topmost compartment is the tenancy id.  Let's start there.
	id, err := dcp.TenancyOCID()
	if err != nil {
		return "", err
	}

	for _, name := range elements {
		// Get a list of compartments within this compartment
		lcr, err := idp.ListCompartments(ctx, identity.ListCompartmentsRequest{
			CompartmentId: &id,
		})
		if err != nil {
			return "", err
		}

		// Look for the correct element in the list of responses
		id = ""
		for _, c := range lcr.Items {
			if *c.Name == name {
				id = *c.Id
				break
			}
		}

		if id == "" {
			return "", fmt.Errorf("could not find compartment named %s in %s", name, compartmentName)
		}
	}

	return id, nil
}
