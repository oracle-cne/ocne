// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"fmt"
	"net"
	"net/url"
)

// ResolveURIToIP extracts a reasonable candidate for
// an IP address from the given URI.  If the URI can
// reasonably be said to point to the host, the
// second return value is "true", otherwise it is "false".
func ResolveURIToIP(uri *url.URL) (string, bool, error) {
	if len(uri.Hostname()) == 0 {
		return "127.0.0.1", true, nil
	}

	hostIPs, err := net.LookupHost(uri.Hostname())
	if err != nil {
		return "", false, err
	}
	if len(hostIPs) == 0 {
		return "", false, fmt.Errorf("Host %s has no IPs", uri.Hostname())
	}

	localAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", false, err
	}

	// Unpack the local addresses into a map for faster lookups
	addrMap := map[string]bool{}
	for _, l := range localAddrs {
		addrMap[l.String()] = true
	}

	// Check the URI IPs against the local addresses.
	// If a match is found, then this URI is assumed
	// to refer to the local host.
	isLocal := false
	addr := hostIPs[0]
	for _, h := range hostIPs {
		if _, ok := addrMap[h]; ok {
			isLocal = true
			addr = h
			break
		}
	}

	return addr, isLocal, nil
}
