// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// CreateHeadlampCerts creates the certs used by Headlamp when running in an OCNE Kubernetes cluster
func CreateHeadlampCerts(uiHost string) (*CertPairWithPem, error) {
	certOptions := CertOptions{
		Country: "US",
		Org:     "Oracle Cloud Native Environment",
		OrgUnit: "Oracle Cloud Native Environment",
		State:   "TX",
	}
	rootConfig := CertConfig{
		CommonName:  uiHost,
		CertOptions: certOptions,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
	}

	leafConfig := CertConfig{
		CommonName:  "Oracle Cloud Native Environment UI",
		CertOptions: certOptions,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
	}

	pemData, err := createCACertAndLeafCert(rootConfig, leafConfig)
	if err != nil {
		log.Errorf("Failed to create UI Certificates for Oracle Cloud Native Environment: %v", err)
		return nil, err
	}
	return pemData, nil
}
