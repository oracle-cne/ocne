// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package driver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

func TestCustomCertificatesProviderValidation(t *testing.T) {
	for _, provider := range []string{"byo", "libvirt", "oci", "olvm", "none"} {
		t.Run(provider, func(t *testing.T) {
			previous, registered := drivers[provider]
			t.Cleanup(func() {
				if registered {
					drivers[provider] = previous
				} else {
					delete(drivers, provider)
				}
			})
			called := false
			drivers[provider] = func(*types.Config, *types.ClusterConfig) (ClusterDriver, error) {
				called = true
				return nil, nil
			}
			for _, defaults := range []*bool{nil, util.BoolPtr(false), util.BoolPtr(true)} {
				for _, cluster := range []*bool{nil, util.BoolPtr(false), util.BoolPtr(true)} {
					t.Run(fmt.Sprintf("defaults=%v/cluster=%v", boolSetting(defaults), boolSetting(cluster)), func(t *testing.T) {
						t.Log("Checking explicit settings are rejected before provider initialization, including false and defaults-file values")
						called = false
						_, err := CreateDriver(&types.Config{CustomCertificates: defaults}, &types.ClusterConfig{Provider: provider, CustomCertificates: cluster})
						rejected := provider != "byo" && (defaults != nil || cluster != nil)
						if rejected {
							if err == nil || !strings.Contains(err.Error(), "customCertificates") || !strings.Contains(err.Error(), provider) {
								t.Fatalf("expected provider-specific customCertificates error, got %v", err)
							}
							if called {
								t.Fatal("unsupported configuration reached provider initialization")
							}
						} else if err != nil || !called {
							t.Fatalf("valid configuration did not initialize provider: %v", err)
						}
					})
				}
			}
		})
	}
}

func boolSetting(value *bool) string {
	if value == nil {
		return "omitted"
	}
	return fmt.Sprint(*value)
}
