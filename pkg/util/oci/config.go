// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OciConfig struct {
	Name                 string
	Fingerprint          string
	Key                  string
	Passphrase           string
	Region               string
	Tenancy              string
	UseInstancePrincipal bool
	User                 string
}

func parseOciConfig(filename string) ([]*OciConfig, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	ret := []*OciConfig{}
	var cur *OciConfig
	for _, l := range strings.Split(string(contents), "\n") {
		l = strings.Trim(l, " \t")

		// Blank lines get ignored
		if len(l) == 0 {
			continue
		}

		// If there is a section header, start a new
		// section.  If there was an old section, add
		// it to the list.
		if suf, ok := strings.CutPrefix(l, "["); ok {
			if section, ok := strings.CutSuffix(suf, "]"); ok {
				if cur != nil {
					ret = append(ret, cur)
				}
				cur = &OciConfig{
					Name:                 section,
					UseInstancePrincipal: false,
				}
			} else {
				return nil, fmt.Errorf("%s is not a valid OCI configuration file element", l)
			}
			continue
		}

		// There isn't a section header.  If this is a comment, then
		// skip
		if strings.HasPrefix(l, "#") {
			continue
		}

		// There isn't a section header.  If there is an equals
		// sign, get the key and the value and put them in
		// the right place
		elements := strings.Split(l, "=")
		if len(elements) != 2 {
			return nil, fmt.Errorf("%s is not a valid OCI configuration file element", l)
		}

		key := strings.Trim(elements[0], " \n")
		val := strings.Trim(elements[1], " \n")

		switch key {
		case "user":
			cur.User = val
		case "fingerprint":
			cur.Fingerprint = val
		case "tenancy":
			cur.Tenancy = val
		case "region":
			cur.Region = val
		case "key_file":
			valBytes, err := os.ReadFile(val)
			if err != nil {
				return nil, err
			}
			cur.Key = string(valBytes)
		case "passphrase":
			cur.Passphrase = val
		}
	}

	if cur == nil {
		return ret, nil
	}
	// The final section is never added to the list
	// because a new section is not started after it
	// Add it now
	ret = append(ret, cur)

	return ret, nil
}

func GetConfig(profile string) (*OciConfig, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sections, err := parseOciConfig(filepath.Join(homedir, ".oci", "config"))
	if err != nil {
		return nil, err
	}

	var ret *OciConfig
	for _, o := range sections {
		if o.Name == profile {
			ret = o
			break
		}
	}

	if ret == nil {
		return nil, fmt.Errorf("no default section found in OCI configuration file")
	}

	// HACK - remove this when the oci-capi chart accepts empty passphrases
	if ret.Passphrase == "" {
		ret.Passphrase = "fiddlesticks"
	}

	return ret, nil
}
