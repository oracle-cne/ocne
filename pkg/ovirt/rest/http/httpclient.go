// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
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

func NewRestClient(resetter TokenResetter, endpointURL string, caMap map[string]string, insecureSkipTLSVerify bool) *REST {
	// Create Transport object
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{RootCAs: rootCertPool(caMap), InsecureSkipVerify: insecureSkipTLSVerify},
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

func (r REST) HeaderContentOctet(header *http.Header) {
	header.Set("Content-Type", "application/octet-stream")
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

func (r REST) HeaderContentLen(header *http.Header, len int64) {
	lenstr := fmt.Sprintf("%v", len)
	header.Set("Content-Length", lenstr)
}

func (r REST) HeaderContentRange(header *http.Header, start int64, end int64, totalLen int64) {
	s := fmt.Sprintf("bytes %v-%v/%v", start, end, totalLen)
	header.Set("Content-Range", s)
}

func (r REST) HeaderNoCache(header *http.Header) {
	header.Set("Cache-Control'", "no-cache")
	header.Set("Pragma", "no-cache")
}

func rootCertPool(caMap map[string]string) *x509.CertPool {
	if len(caMap) == 0 {
		return nil
	}

	// if we have caData, use it
	certPool := x509.NewCertPool()

	// Create CA cert pool
	for _, cert := range caMap {
		certPool.AppendCertsFromPEM([]byte(cert))
	}
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
