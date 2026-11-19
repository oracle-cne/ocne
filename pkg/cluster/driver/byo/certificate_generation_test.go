// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package byo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	conftypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
)

func TestDisabledCertificateGeneration(t *testing.T) {
	t.Log("Generating BYO initialization Ignition without PKI or a local kubeconfig")
	bd := &ByoDriver{KubeconfigPath: filepath.Join(t.TempDir(), "kubeconfig"), Config: conftypes.ClusterConfig{
		Provider: "byo", CustomCertificates: util.BoolPtr(true), LoadBalancer: "127.0.0.1",
		OsRegistry: "ostree-unverified-registry:example.com/ock", KubeVersion: "1.33.5", ServiceSubnet: "10.96.0.0/12",
		Providers:           conftypes.Providers{Byo: conftypes.ByoProvider{NetworkInterface: "eth0"}},
		ExtraIgnitionInline: `{"ignition":{"version":"3.4.0"},"storage":{"files":[{"path":"/etc/kubernetes/pki/custom.crt","contents":{"source":"data:,external"},"mode":384}]}}`,
	}}
	raw, err := bd.clusterInit()
	if err != nil {
		t.Fatal(err)
	}
	if bd.PKIInfo != nil {
		t.Fatal("generated bootstrap PKI while disabled")
	}
	if _, err := os.Stat(bd.KubeconfigPath); !os.IsNotExist(err) {
		t.Fatalf("unexpected kubeconfig result: %v", err)
	}
	var ign igntypes.Config
	if err := json.Unmarshal(raw, &ign); err != nil {
		t.Fatal(err)
	}
	external := false
	for _, file := range ign.Storage.Files {
		if file.Path == "/etc/kubernetes/pki/custom.crt" {
			external = true
			continue
		}
		if strings.HasPrefix(file.Path, "/etc/kubernetes/pki/") {
			t.Fatalf("generated PKI file %s", file.Path)
		}
	}
	if !external {
		t.Fatal("external Ignition file was lost")
	}
}
