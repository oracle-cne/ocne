// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Ensure Worker implements SPI interface
var _ HeaderAdder = &REST{}

type TokenResetter interface {
	ClearAccessToken()
}

type HeaderAdder interface {
	HeaderAcceptJSON(header *http.Header)
	HeaderContentJSON(header *http.Header)
	HeaderContentXML(header *http.Header)
	HeaderUrlEncoded(header *http.Header)
	HeaderBearerToken(header *http.Header, token string)
}

type REST struct {
	endpointURL   string
	client        *http.Client
	tokenResetter TokenResetter
}

func NewRestClient(resetter TokenResetter, endpointURL string, ca string) *REST {
	// Create Transport object
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{RootCAs: rootCertPool([]byte(ca))},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{Transport: tr}
	return &REST{
		endpointURL:   endpointURL,
		client:        client,
		tokenResetter: resetter,
	}
}

func (r REST) HeaderAcceptJSON(header *http.Header) {
	header.Set("Accept", "application/json")
}
func (r REST) HeaderContentJSON(header *http.Header) {
	header.Set("Content-Type", "application/json")
}
func (r REST) HeaderContentXML(header *http.Header) {
	header.Set("Content-Type", "application/xml")
}
func (r REST) HeaderUrlEncoded(header *http.Header) {
	header.Set("Content-Type", "application/x-www-form-urlencoded")
}
func (r REST) HeaderBearerToken(header *http.Header, token string) {
	header.Set("Authorization", "Bearer "+token)
}

func rootCertPool(caData []byte) *x509.CertPool {
	if len(caData) == 0 {
		return nil
	}
	// if we have caData, use it
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	return certPool
}

func resolveURL(endpoint string, path string) string {
	var URL string
	if strings.HasPrefix(endpoint, "https://") {
		URL = fmt.Sprintf("%s%s", endpoint, path)
	} else {
		URL = fmt.Sprintf("https://%s%s", endpoint, path)
	}
	URL = strings.TrimSuffix(URL, "/")
	return URL
}
