// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	igntypes "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/oracle-cne/ocne/pkg/cluster/kubepki"
	clustertypes "github.com/oracle-cne/ocne/pkg/cluster/types"
	conftypes "github.com/oracle-cne/ocne/pkg/config/types"
)

func TestDefaultCertificateGeneration(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key")
	for _, path := range []string{certPath, keyPath} {
		if err := os.WriteFile(path, []byte("test bootstrap credentials"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ld := &LibvirtDriver{KubeAPIServerIP: "127.0.0.1", KubeVersion: "1.33.5",
		PKIInfo: &kubepki.PKIInfo{CACertPath: certPath, CAKeyPath: keyPath}, Config: conftypes.ClusterConfig{
			Provider: "libvirt", OsRegistry: "ostree-unverified-registry:example.com/ock",
			KubeVersion: "1.33.5", ServiceSubnet: "10.96.0.0/12",
		}}
	for _, join := range []bool{false, true} {
		t.Logf("Checking libvirt retains generated certificate bootstrap (join=%t)", join)
		raw, err := ld.generateIgnition("example-control-plane-1", clustertypes.ControlPlaneRole, join, "token", []string{"hash"}, false, true)
		if err != nil {
			t.Fatal(err)
		}
		var ign igntypes.Config
		if err := json.Unmarshal(raw, &ign); err != nil {
			t.Fatal(err)
		}
		paths := make(map[string]bool)
		for _, file := range ign.Storage.Files {
			paths[file.Path] = true
		}
		if paths["/etc/ocne/disable-certificate-generation"] {
			t.Fatal("libvirt disabled certificate generation")
		}
		if !join && (!paths["/etc/kubernetes/pki/ca.crt"] || !paths["/etc/kubernetes/pki/ca.key"]) {
			t.Fatal("initial libvirt node is missing generated CA credentials")
		}
	}
}
