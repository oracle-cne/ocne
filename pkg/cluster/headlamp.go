// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"

	"github.com/oracle-cne/ocne/pkg/certificate"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateCert - create a default certificate for accessing the UI
func CreateCert(client kubernetes.Interface, namespace string) error {
	err := k8s.CreateNamespaceIfNotExists(client, namespace)
	if err != nil {
		return err
	}

	// Nothing to do if the secret already exists
	_, err = client.CoreV1().Secrets(namespace).Get(context.TODO(), constants.UISecretNameTLS, metav1.GetOptions{})
	if err == nil || !errors.IsNotFound(err) {
		return err
	}

	// Create the certificates
	certs, err := certificate.CreateHeadlampCerts(constants.UIServiceName)
	if err != nil {
		return err
	}

	// Create the TLS secret
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.UISecretNameTLS,
			Namespace: namespace,
		},
		Data: make(map[string][]byte),
		Type: v1.SecretTypeTLS,
	}
	secret.Data[constants.CertKey] = certs.LeafCertResult.CertPEM
	secret.Data[constants.PrivKey] = certs.LeafCertResult.PrivateKeyPEM

	_, err = client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	secret.ObjectMeta.Name = constants.CASecretNameTLS
	secret.Data[constants.CertKey] = certs.RootCertResult.CertPEM
	secret.Data[constants.PrivKey] = certs.RootCertResult.PrivateKeyPEM

	_, err = client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})

	return err
}
