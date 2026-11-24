// Copyright (c) 2026, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/oracle-cne/ocne/pkg/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUICertificateGenerationDisabled(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Logf("Checking disabled generation leaves the Kubernetes API untouched (existing credentials=%t)", existing)
		client := fake.NewSimpleClientset()
		if existing {
			client = fake.NewSimpleClientset(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.UISecretNameTLS, Namespace: constants.UINamespace}})
		}
		if err := CreateCert(client, constants.UINamespace, false); err != nil {
			t.Fatal(err)
		}
		if len(client.Actions()) != 0 {
			t.Fatalf("disabled generation accessed Kubernetes: %v", client.Actions())
		}
	}
}

func TestUICertificateGenerationEnabled(t *testing.T) {
	t.Log("Generating UI credentials and verifying both installed certificate/key pairs")
	client := fake.NewSimpleClientset(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: constants.UINamespace}})
	if err := CreateCert(client, constants.UINamespace, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{constants.UISecretNameTLS, constants.CASecretNameTLS} {
		secret, err := client.CoreV1().Secrets(constants.UINamespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tls.X509KeyPair(secret.Data[constants.CertKey], secret.Data[constants.PrivKey]); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Checking a repeat installation preserves the generated credentials")
	client.ClearActions()
	if err := CreateCert(client, constants.UINamespace, true); err != nil {
		t.Fatal(err)
	}
	for _, action := range client.Actions() {
		if action.GetResource().Resource == "secrets" && action.GetVerb() != "get" {
			t.Fatalf("existing credentials were modified: %v", action)
		}
	}
}
