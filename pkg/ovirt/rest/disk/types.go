package disk

type CreateDiskRequest struct {
	StorageDomainList StorageDomainList `json:"storage_domains"`
	Name              string            `json:"name"`
	ProvisionedSize   string            `json:"provisioned_size"`
	Format            string            `json:"format"`
	Backup            string            `json:"backup"`
}

type StorageDomainList struct {
	StorageDomains []StorageDomain `json:"storage_domain"`
}

type StorageDomain struct {
	Id string `json:"id"`
}

type Disk struct {
	ActualSize      string `json:"actual_size"`
	Alias           string `json:"alias"`
	Backup          string `json:"backup"`
	ContentType     string `json:"content_type"`
	Format          string `json:"format"`
	ImageId         string `json:"image_id"`
	PropagateErrors string `json:"propagate_errors"`
	ProvisionedSize string `json:"provisioned_size"`
	Shareable       string `json:"shareable"`
	Sparse          string `json:"sparse"`
	Status          string `json:"status"`
	StorageType     string `json:"storage_type"`
	TotalSize       string `json:"total_size"`
	WipeAfterDelete string `json:"wipe_after_delete"`
	DiskProfile     struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"disk_profile"`
	Quota struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"quota"`
	StorageDomains struct {
		StorageDomain []struct {
			Href string `json:"href"`
			Id   string `json:"id"`
		} `json:"storage_domain"`
	} `json:"storage_domains"`
	Actions struct {
		Link []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"link"`
	} `json:"actions"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Link        []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"link"`
	Href string `json:"href"`
	Id   string `json:"id"`
}
