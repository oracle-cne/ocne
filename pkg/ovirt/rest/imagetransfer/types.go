// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagetransfer

const PhaseTransferring string = "transferring"

// CreateImageTransferRequest specifies the request to the image transfer service
type CreateImageTransferRequest struct {
	Disk      `json:"disk"`
	Direction string `json:"direction"`
}

type Disk struct {
	Id string `json:"id"`
}

type ImageTransfer struct {
	Active            string `json:"active"`
	Direction         string `json:"direction"`
	Format            string `json:"format"`
	InactivityTimeout string `json:"inactivity_timeout"`
	Phase             string `json:"phase"`
	ProxyUrl          string `json:"proxy_url"`
	Shallow           string `json:"shallow"`
	TimeoutPolicy     string `json:"timeout_policy"`
	TransferUrl       string `json:"transfer_url"`
	Transferred       string `json:"transferred"`
	Host              struct {
		Href string `json:"href"`
		Id   string `json:"id"`
	} `json:"host"`
	Image struct {
		Id string `json:"id"`
	} `json:"image"`
	Actions struct {
		Link []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"link"`
	} `json:"actions"`
	Href string `json:"href"`
	Id   string `json:"id"`
}

type ImageTransferList struct {
	ImageTransfers []struct {
		ImageTransfer
	} `json:"image_transfer"`
}
