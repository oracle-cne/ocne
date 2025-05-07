// Copyright (c) 2024, 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ovclient

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/oracle-cne/ocne/pkg/cluster/driver/olvm"
	ovhttp "github.com/oracle-cne/ocne/pkg/ovirt/rest/http"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
)

type Credentials struct {
	// Username is the OAuth2 username
	Username string

	// Password is the OAuth2 user password
	Password string

	// Scope is the OAuth2 scope
	Scope string

	// CA is the oVirt CA
	CA map[string]string
}

type AuthData struct {
	AccessToken string `json:"access_token"`
}

type Client struct {
	// AccessToken is the REST bearer token
	AccessToken string

	// ApiServerURL is the endpoint to the oVirt API server
	ApiServerURL string

	// REST is the HTTP REST client used for the oVirt REST API
	*ovhttp.REST

	*Credentials

	InsecureSkipTLSVerify bool
}

// GetOVClient gets an ovClient
func GetOVClient(cli kubernetes.Interface, caMap map[string]string, apiServerURL string, insecureSkipTLSVerify bool) (*Client, error) {
	// validate the secret that has the oVirt REST creds
	creds, err := getCredentials(cli)
	if err != nil {
		return nil, err
	}

	creds.CA = caMap

	// Get an oVirt client
	ovcli, err := ensureOvClient(creds, apiServerURL, insecureSkipTLSVerify)
	if err != nil {
		return nil, err
	}

	return ovcli, nil
}

// ensureOvClient ensures that we have an access token that can be used with the oVirt REST API.
func ensureOvClient(creds *Credentials, apiServerURL string, insecureSkipTLSVerify bool) (*Client, error) {
	// Create a new client and validate that it works
	ovcli := &Client{
		ApiServerURL:          apiServerURL,
		Credentials:           creds,
		InsecureSkipTLSVerify: insecureSkipTLSVerify,
	}

	if err := ovcli.ensureAccessToken(); err != nil {
		return nil, err
	}

	// Validate the token by getting system information
	const path = "/api"

	// call the server to get the datacenters just to verify the REST API works.
	body, err := ovcli.REST.Get(ovcli.AccessToken, path)
	if err != nil {
		err = fmt.Errorf("Error calling HTTP GET for URL %s: %v", ovcli.ApiServerURL, err)
		log.Error(err)
		return nil, err
	}
	if len(body) == 0 {
		err = fmt.Errorf("No system data found at %v", ovcli.ApiServerURL)
		log.Error(err)
		return nil, err
	}

	return ovcli, nil
}

// ensureAccessToken ensures that the access token exists
func (o *Client) ensureAccessToken() error {
	const tokenPath = "/sso/oauth/token"

	if o.AccessToken != "" {
		return nil
	}

	o.REST = ovhttp.NewRestClient(o, o.ApiServerURL, o.CA, o.InsecureSkipTLSVerify)

	// create the payload to send using POST
	d := url.Values{}
	d.Set("username", o.Credentials.Username)
	d.Set("password", o.Credentials.Password)
	d.Set("scope", o.Credentials.Scope)
	d.Set("grant_type", "password")

	// call the server to get the access token
	h := &http.Header{}
	o.REST.HeaderUrlEncoded(h)
	o.REST.HeaderAcceptJSON(h)
	// path string, payload io.Reader, header *rest.Header
	body, _, err := o.REST.Post(tokenPath, strings.NewReader(d.Encode()), h)
	if err != nil {
		err = fmt.Errorf("Error doing HTTP POST to oVirt server: %v", err)
		log.Error(err)
		return err
	}
	if len(body) == 0 {
		err = fmt.Errorf("Error doing HTTP POST to create access token.  No body returned by server: %v", err)
		log.Error(err)
		return err
	}

	// extract access token from results
	ad := &AuthData{}
	if err = json.Unmarshal(body, ad); err != nil {
		err = fmt.Errorf("Error UnMarshalling JSON for credentials: %v", err)
		log.Error(err)
		return err
	}

	o.AccessToken = ad.AccessToken
	if o.AccessToken == "" {
		err = fmt.Errorf("Access token missing from body returned by POST")
		log.Error(err)
		return err
	}

	return nil
}

// ClearAccessToken is called when there is a REST HTTP error
func (o *Client) ClearAccessToken() {
	o.AccessToken = ""
}

func getCredentials(cli kubernetes.Interface) (*Credentials, error) {
	c := Credentials{}

	c.Username = os.Getenv(olvm.EnvUsername)
	if c.Username == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM username", olvm.EnvUsername)
	}
	c.Password = os.Getenv(olvm.EnvPassword)
	if c.Password == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM password", olvm.EnvPassword)
	}
	c.Scope = os.Getenv(olvm.EnvScope)
	if c.Scope == "" {
		return nil, fmt.Errorf("Missing environment variable %s used to specify OLVM scope", olvm.EnvScope)
	}

	return &c, nil
}
