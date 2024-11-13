// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle/oci-go-sdk/v65/common"
	"os"
	"path/filepath"
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

// Convert rsa private key to string
func privateKeyToString(privateKey *rsa.PrivateKey) (string, error) {
	bytes := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

/** Read OCI config file from -
* 1) Default path ~/.oci/config
* 2) File specified with OCI_CONFIG_FILE
*  And, the profile set with OCI_CONFIG_PROFILE, otherwise DEFAULT
 */

func readOCIConfigFromDisk() (common.ConfigurationProvider, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	ociConfigFilePath := filepath.Join(homedir, ".oci", "config")
	if pathFromEnv := os.Getenv("OCI_CONFIG_FILE"); pathFromEnv != "" {
		ociConfigFilePath = pathFromEnv
	}

	profileName := "DEFAULT"
	if ociProfile := os.Getenv("OCI_CONFIG_PROFILE"); ociProfile != "" {
		profileName = ociProfile
	}
	return common.CustomProfileConfigProvider(
		ociConfigFilePath,
		profileName,
	), nil
}

// GetOCIConfig reads the OCI config specified either via providers.oci.profile or from the disk
// Returns an object to ConfigurationProvider, otherwise an error
func GetOCIConfig(profile types.OCIProfile) (common.ConfigurationProvider, error) {
	var provider common.ConfigurationProvider
	if profile != (types.OCIProfile{}) {
		valBytes, err := os.ReadFile(profile.Key)
		if err != nil {
			return nil, fmt.Errorf("Error in reading OCI pem file: %w", err)
		}
		key := string(valBytes)
		provider = common.NewRawConfigurationProvider(
			profile.Tenancy,
			profile.User,
			profile.Region,
			profile.Fingerprint,
			key,
			&profile.Passphrase,
		)
	} else {
		provider, _ = readOCIConfigFromDisk()
	}

	return provider, nil
}

// GetConfig reads the OCI config specified either via providers.oci.profile or from the disk
// Returns an object to OciConfig, otherwise an error
func GetConfig(profile types.OCIProfile) (*OciConfig, error) {
	provider, err := GetOCIConfig(profile)
	if provider == nil {
		return nil, fmt.Errorf("Failed to read OCI configuration: %w", err)
	}

	var ret OciConfig

	if tenancy, err := provider.TenancyOCID(); err == nil {
		ret.Tenancy = tenancy
	} else {
		return nil, fmt.Errorf("failed to retrieve Tenancy OCID: %w", err)
	}

	if user, err := provider.UserOCID(); err == nil {
		ret.User = user
	} else {
		return nil, fmt.Errorf("failed to retrieve User OCID: %w", err)
	}

	if reg, err := provider.Region(); err == nil {
		ret.Region = reg
	} else {
		return nil, fmt.Errorf("failed to retrieve Region: %w", err)
	}

	if fp, err := provider.KeyFingerprint(); err == nil {
		ret.Fingerprint = fp
	} else {
		return nil, fmt.Errorf("failed to retrieve Fingerprint: %w", err)
	}

	if key, err := provider.PrivateRSAKey(); err == nil {
		ret.Key, _ = privateKeyToString(key)
	} else {
		return nil, fmt.Errorf("failed to retrieve Private Key Path: %w", err)
	}
	return &ret, nil
}
