// Copyright (c) 2024 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateWebhookCertificates tests that the certificates needed for OCNE are created
// GIVEN an output directory for certificates
//
//	WHEN I call CreateWebhookCertificates
//	THEN all the needed certificate artifacts are created
func TestCreateOCNECertificates(t *testing.T) {
	asserts := assert.New(t)
	options := CertOptions{
		Country: "US",
		Org:     "OCNE",
		OrgUnit: "OCNE",
		State:   "TX",
	}
	certpair, err := createKubernetesCerts("1.2.3.4", "2.3.4.5", options)
	asserts.NoError(err)

	// Verify generated certs
	asserts.True(strings.Contains(string(certpair.RootCertResult.CertPEM), "-----BEGIN CERTIFICATE-----"))
	asserts.True(strings.Contains(string(certpair.RootCertResult.PrivateKeyPEM), "-----BEGIN RSA PRIVATE KEY-----"))
	asserts.True(strings.Contains(string(certpair.LeafCertResult.CertPEM), "-----BEGIN CERTIFICATE-----"))
	asserts.True(strings.Contains(string(certpair.LeafCertResult.PrivateKeyPEM), "-----BEGIN RSA PRIVATE KEY-----"))

}
