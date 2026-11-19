// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ignition

import (
	"slices"
	"testing"

	clustertypes "github.com/oracle-cne/ocne/pkg/cluster/types"
)

func TestCertificateGenerationPhases(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Logf("Checking initialization and join phases with certificate generation disabled=%t", disabled)
		init := GenerateKubeadmInit(&ClusterInit{SkipCertificateGeneration: disabled})
		for _, phase := range []string{"certs", "kubeconfig", "upload-certs"} {
			if slices.Contains(init.SkipPhases, phase) != disabled {
				t.Fatalf("unexpected init skip phases: %v", init.SkipPhases)
			}
		}
		for _, role := range []clustertypes.NodeRole{clustertypes.ControlPlaneRole, clustertypes.WorkerRole} {
			join := GenerateKubeadmJoin(&ClusterJoin{Role: role, SkipCertificateGeneration: disabled})
			for _, phase := range []string{"control-plane-prepare/download-certs", "control-plane-prepare/certs", "control-plane-prepare/kubeconfig"} {
				if slices.Contains(join.SkipPhases, phase) != (disabled && role == clustertypes.ControlPlaneRole) {
					t.Fatalf("unexpected join skip phases: %v", join.SkipPhases)
				}
			}
		}
	}
}

func TestCertificateGenerationIgnition(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Logf("Checking certificate files and bootstrap mode with certificate generation disabled=%t", disabled)
		ign, err := InitializeCluster(&ClusterInit{
			SkipCertificateGeneration: disabled, OsRegistry: "ostree-unverified-registry:example.com/ock", KubeVersion: "1.33.5",
			KubePKICert: "test CA", KubePKIKey: "test key", ServiceSubnet: "10.96.0.0/12",
		})
		if err != nil {
			t.Fatal(err)
		}
		paths := make(map[string]bool)
		for _, file := range ign.Storage.Files {
			paths[file.Path] = true
		}
		if paths[CaCrtFilePath] == disabled || paths[CaKeyFilePath] == disabled {
			t.Fatal("CA files do not match generation mode")
		}
		if paths["/etc/ocne/disable-certificate-generation"] != disabled {
			t.Fatal("bootstrap mode does not match generation mode")
		}
	}
}
