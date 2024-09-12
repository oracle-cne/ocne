// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path"
	"path/filepath"
)

// CertLocation specifies the on disk locations of the certs
type CertLocation struct {
	// Directory is the directory where the cert files will be written
	Directory string

	// RootCertFilename is the root cert filename
	RootCertFilename string

	// RootPrivateKeyFilename is the root private key filename
	RootPrivateKeyFilename string

	// LeafCertFilename is the leaf cert filename
	LeafCertFilename string

	// LeafPrivateKeyFilename is the leaf private key filename
	LeafPrivateKeyFilename string
}

// createCACertAndLeafCert creates a self-signed CA cert and leaf cert, then returns the generated certs and PEM data
func createCACertAndLeafCert(rootConfig CertConfig, leafConfig CertConfig) (*CertPairWithPem, error) {
	// Create the object that will be loaded with the PEM data
	pem := CertPairWithPem{}

	// Create the root cert
	rResult, err := createRootCert(rootConfig)
	if err != nil {
		return nil, err
	}
	pem.RootCertResult = rResult

	// Create the leaf cert
	iResult, err := createLeafCert(leafConfig, rResult)
	if err != nil {
		return nil, err
	}
	pem.LeafCertResult = iResult

	// Write the chain
	b := bytes.Buffer{}
	b.Write(pem.LeafCertResult.CertPEM)
	b.Write(pem.RootCertResult.CertPEM)
	pem.CertChainPEM = b.Bytes()

	return &pem, nil
}

// Create the root certificate
func createRootCert(config CertConfig) (*CertResult, error) {
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	certRequest := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Country:      []string{config.Country},
			Province:     []string{config.State},
			Organization: []string{config.Org},
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	return createCert(certRequest, certRequest, nil)
}

// Create the leaf certificate
func createLeafCert(config CertConfig, rootResult *CertResult) (*CertResult, error) {
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	certRequest := &x509.Certificate{
		DNSNames:     config.DNSNames,
		IPAddresses:  config.IPAddresses,
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Country:      []string{config.Country},
			Province:     []string{config.State},
			Organization: []string{config.Org},
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	return createCert(certRequest, rootResult.Cert, rootResult.PrivateKey)
}

// Create the certificate. The parent certificate will be used to sign the new cert
func createCert(certRequest *x509.Certificate, parentCert *x509.Certificate, parentPrivKey *rsa.PrivateKey) (*CertResult, error) {
	// create private key for this cert
	privKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// For root cert there is no parent key
	if parentPrivKey == nil {
		parentPrivKey = privKey
	}

	// PEM encode the private key
	privKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	// create the CA certificate using the partial cert as input
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, certRequest, parentCert, &privKey.PublicKey, parentPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the new cert
	certPEM := new(bytes.Buffer)
	_ = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Reload the cert so that we get fields like the Subject Key ID that are generated
	newCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	return &CertResult{
		PrivateKey:    privKey,
		PrivateKeyPEM: privKeyPEM.Bytes(),
		CertPEM:       certPEM.Bytes(),
		Cert:          newCert,
	}, nil
}

// newSerialNumber returns a new random serial number suitable for use in a certificate.
func newSerialNumber() (*big.Int, error) {
	// A serial number can be up to 20 octets in size.
	return cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 8*20))
}

// writeCertPair write the root and leaf certs to a file
func writeCertPair(pair *CertPairWithPem, loc CertLocation) error {
	dir, err := filepath.Abs(loc.Directory)
	if err != nil {
		return err
	}

	// Write the certs and keys, this will overwrite any existing cert files
	err = os.WriteFile(path.Join(dir, loc.RootCertFilename), pair.RootCertResult.CertPEM, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(dir, loc.RootPrivateKeyFilename), pair.RootCertResult.PrivateKeyPEM, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(dir, loc.LeafCertFilename), pair.LeafCertResult.CertPEM, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(dir, loc.LeafPrivateKeyFilename), pair.LeafCertResult.PrivateKeyPEM, 0644)
	if err != nil {
		return err
	}

	return nil
}
