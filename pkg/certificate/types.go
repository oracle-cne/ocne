// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"crypto/rsa"
	"crypto/x509"
	"net"
	"time"
)

// CertConfig specifies the certificate configuration
type CertConfig struct {

	// DNS names
	DNSNames []string

	// IP IpAddresses
	IPAddresses []net.IP

	// CommonName is the certificate common name
	CommonName string

	// CertOptions for location and organization information surrounding the certificate
	CertOptions

	// NotBefore time when certificate is valid
	NotBefore time.Time

	// NotAfter time when certificate is valid
	NotAfter time.Time

	// AltNames has the list of alternative names
	AltNames map[string]string
}

// CertPairWithPem contains certificates and chain in PEM format
type CertPairWithPem struct {
	// The certificate chain in PEM format.  This contains the leaf cert followed
	// by the root cert
	CertChainPEM []byte

	// The leaf cert results
	LeafCertResult *CertResult

	// The root cert results
	RootCertResult *CertResult
}

// CertResult contains the generated cert results
type CertResult struct {
	PrivateKey    *rsa.PrivateKey
	PrivateKeyPEM []byte
	Cert          *x509.Certificate
	CertPEM       []byte
}

// CertOptions contains the country, state, org, and orgUnit information of a certificate
type CertOptions struct {
	Country string
	Org     string
	OrgUnit string
	State   string
}
