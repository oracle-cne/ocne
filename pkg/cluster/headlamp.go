// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"

	"github.com/oracle-cne/ocne/pkg/certificate"
	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateCert - create a default certificate for accessing the UI
func CreateCert(client kubernetes.Interface, namespace string, generate bool) error {
	if !generate {
		log.Debug("UI certificate generation is disabled; using externally provisioned UI TLS Secret")
		return nil
	}
	log.Debug("Ensuring UI namespace exists before checking for installed TLS credentials")
	err := k8s.CreateNamespaceIfNotExists(client, namespace)
	if err != nil {
		return err
	}

	// Nothing to do if the secret already exists
	_, err = client.CoreV1().Secrets(constants.OCNESystemNamespace).Get(context.TODO(), constants.UISecretNameTLS, metav1.GetOptions{})
	if err == nil || !errors.IsNotFound(err) {
		log.Debugf("UI TLS Secret lookup complete; no generation required (error=%v)", err)
		return err
	}

	// Create the certificates
	log.Debug("Generating UI CA and server certificates")
	certs, err := certificate.CreateHeadlampCerts(constants.UIServiceName)
	if err != nil {
		return err
	}

	// Create the TLS secret
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.UISecretNameTLS,
			Namespace: constants.OCNESystemNamespace,
		},
		Data: make(map[string][]byte),
		Type: v1.SecretTypeTLS,
	}
	secret.Data[constants.CertKey] = certs.LeafCertResult.CertPEM
	secret.Data[constants.PrivKey] = certs.LeafCertResult.PrivateKeyPEM

	log.Debug("Creating UI server TLS Secret")
	_, err = client.CoreV1().Secrets(constants.OCNESystemNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	secret.ObjectMeta.Name = constants.CASecretNameTLS
	secret.Data[constants.CertKey] = certs.RootCertResult.CertPEM
	secret.Data[constants.PrivKey] = certs.RootCertResult.PrivateKeyPEM

	log.Debug("Creating UI CA TLS Secret")
	_, err = client.CoreV1().Secrets(constants.OCNESystemNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})

	return err
}
