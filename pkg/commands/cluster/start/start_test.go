// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"testing"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAutoStartUIWithCustomCertificates(t *testing.T) {
	t.Log("Checking custom UI certificates allow startup to finish without a UI deployment or TLS Secrets")
	clusterConfig := &types.ClusterConfig{CustomUICertificates: util.BoolPtr(true)}
	client := fake.NewSimpleClientset()
	err := autoStartUI(client, "missing-kubeconfig", !clusterConfig.UICertificateGenerationEnabled(), func(message string) {
		t.Fatalf("UI startup must not wait for readiness: %s", message)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.Actions()) != 0 {
		t.Fatalf("custom UI startup accessed Kubernetes: %v", client.Actions())
	}
}
