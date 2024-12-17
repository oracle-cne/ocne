// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package storagedomain

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
)

// GetStorageDomains gets all datacenters.
func GetStorageDomains(ovcli *ovclient.Client) (*StorageDomainList, error) {
	const path = "/api/storagedomains"

	// call the server to get the datacenters
	body, err := ovcli.REST.Get(ovcli.AccessToken, path)
	if err != nil {
		err = fmt.Errorf("Error doing HTTP GET: %v", err)
		return nil, err
	}

	sdList := &StorageDomainList{}
	err = json.Unmarshal(body, sdList)
	if err != nil {
		err = fmt.Errorf("Error unmarshaling StorageDomains: %v", err)
		log.Error(err)
		return nil, err
	}

	return sdList, nil
}

// GetStorageDomain gets a storage domain by name
func GetStorageDomain(ovcli *ovclient.Client, storageDomainName string) (*StorageDomain, error) {
	sdList, err := GetStorageDomains(ovcli)
	if err != nil {
		return nil, err
	}

	for i, sd := range sdList.StorageDomains {
		if sd.Name == storageDomainName {
			return &sdList.StorageDomains[i], nil
		}
	}

	return nil, fmt.Errorf("Storage Domain %s not found", storageDomainName)
}
