// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/constants"
)

func TestCertificateGenerationConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, defaults, cluster string
		want                    bool
	}{
		{"omitted", "", "provider: byo", true},
		{"explicit default", "", "provider: byo\ncustomCertificates: false", true},
		{"defaults disabled", "customCertificates: true", "provider: byo", false},
		{"cluster disabled", "customCertificates: false", "provider: byo\ncustomCertificates: true", false},
		{"cluster reenabled", "customCertificates: true", "provider: byo\ncustomCertificates: false", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Log("Parsing defaults and cluster settings, then applying command configuration precedence")
			defaults, err := ParseConfig(tc.defaults)
			if err != nil {
				t.Fatal(err)
			}
			cluster, err := ParseClusterConfig(tc.cluster)
			if err != nil {
				t.Fatal(err)
			}
			mergedDefaults := types.MergeConfig(defaults, &types.Config{})
			mergedCluster := types.MergeClusterConfig(cluster, &types.ClusterConfig{})
			resolved := types.OverlayConfig(&mergedCluster, &mergedDefaults)
			copied := types.CopyClusterConfig(&resolved)
			if got := copied.CertificateGenerationEnabled(); got != tc.want {
				t.Fatalf("certificate generation = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestCertificateGenerationDefaultsFile(t *testing.T) {
	t.Log("Loading certificate generation from the defaults file selected by OCNE_DEFAULTS")
	path := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(path, []byte("customCertificates: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(constants.UserConfigDefaultsEnvironmentVariable, path)
	defaults, err := GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	resolved := types.OverlayConfig(&types.ClusterConfig{Provider: "byo"}, defaults)
	if resolved.CertificateGenerationEnabled() {
		t.Fatal("defaults file did not disable certificate generation")
	}
}
