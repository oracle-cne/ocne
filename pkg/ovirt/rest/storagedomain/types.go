// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package storagedomain

type StorageDomain struct {
	Available                  string `json:"available,omitempty"`
	Backup                     string `json:"backup"`
	BlockSize                  string `json:"block_size"`
	Committed                  string `json:"committed"`
	CriticalSpaceActionBlocker string `json:"critical_space_action_blocker"`
	DiscardAfterDelete         string `json:"discard_after_delete"`
	ExternalStatus             string `json:"external_status"`
	Master                     string `json:"master"`
	Storage                    struct {
		Type        string `json:"type"`
		VolumeGroup struct {
			LogicalUnits struct {
				LogicalUnit []struct {
					DiscardMaxSize    string `json:"discard_max_size"`
					DiscardZeroesData string `json:"discard_zeroes_data"`
					LunMapping        string `json:"lun_mapping"`
					Paths             string `json:"paths"`
					ProductId         string `json:"product_id"`
					Serial            string `json:"serial"`
					Size              string `json:"size"`
					StorageDomainId   string `json:"storage_domain_id"`
					VendorId          string `json:"vendor_id"`
					VolumeGroupId     string `json:"volume_group_id"`
					Id                string `json:"id"`
				} `json:"logical_unit"`
			} `json:"logical_units"`
			Id string `json:"id"`
		} `json:"volume_group,omitempty"`
	} `json:"storage"`
	StorageFormat             string `json:"storage_format"`
	SupportsDiscard           string `json:"supports_discard"`
	SupportsDiscardZeroesData string `json:"supports_discard_zeroes_data"`
	Type                      string `json:"type"`
	Used                      string `json:"used,omitempty"`
	WarningLowSpaceIndicator  string `json:"warning_low_space_indicator"`
	WipeAfterDelete           string `json:"wipe_after_delete"`
	DataCenters               struct {
		DataCenter []struct {
			Href string `json:"href"`
			Id   string `json:"id"`
		} `json:"data_center"`
	} `json:"data_centers,omitempty"`
	Actions struct {
		Link []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"link"`
	} `json:"actions"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Comment     string `json:"comment"`
	Link        []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"link"`
	Href   string `json:"href"`
	Id     string `json:"id"`
	Status string `json:"status,omitempty"`
}

type StorageDomainList struct {
	StorageDomains []StorageDomain `json:"storage_domain"`
}
