// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package storagedomain

type StorageDomain struct {
	Local             string `json:"local"`
	QuotaMode         string `json:"quota_mode"`
	Status            string `json:"status"`
	StorageFormat     string `json:"storage_format"`
	SupportedVersions struct {
		Version []struct {
			Major string `json:"major"`
			Minor string `json:"minor"`
		} `json:"version"`
	} `json:"supported_versions"`
	Version struct {
		Major string `json:"major"`
		Minor string `json:"minor"`
	} `json:"version"`
	MacPool struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"mac_pool"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Href        string `json:"href"`
	Id          string `json:"id"`
}

type StorageDomainList struct {
	StorageDomains []StorageDomain `json:"storage_domain"`
}
