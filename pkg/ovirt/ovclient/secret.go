// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ovclient

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/k8s"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	usernameKey = "username"
	passwordKey = "password"
	scopeKey    = "scope"
)

func GetCredentials(cli kubernetes.Interface, nsn types.NamespacedName) (*corev1.Secret, *Credentials, error) {
	secret, err := GetCredSecret(cli, nsn)
	if err != nil {
		return nil, nil, err
	}

	c := getCreds(secret)
	if c.Username == "" {
		log.Errorf("Secret %s/%s is missing username", secret.Namespace, secret.Name)
	}
	if c.Password == "" {
		log.Errorf("Secret %s/%s is missing password", secret.Namespace, secret.Name)
	}
	if c.Scope == "" {
		log.Errorf("Secret %s/%s is missing scope", secret.Namespace, secret.Name)
	}
	return secret, c, nil
}

// GetCredSecret gets the credential secret
func GetCredSecret(cli kubernetes.Interface, nsn types.NamespacedName) (*corev1.Secret, error) {
	secret, err := k8s.GetSecret(cli, nsn.Namespace, nsn.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = fmt.Errorf("The credential Secret %v is missing", nsn)
			log.Error(err.Error())
			return nil, err
		}
		err = fmt.Errorf("Failed to fetch secret %v: %v", nsn, err)
		log.Error(err.Error())
		return nil, err
	}

	return secret, nil
}

// getCreds gets the credentials from the secret
func getCreds(secret *corev1.Secret) *Credentials {
	c := &Credentials{}
	if secret.Data == nil {
		return c
	}
	c.Username = string(secret.Data[usernameKey])
	c.Password = string(secret.Data[passwordKey])
	c.Scope = string(secret.Data[scopeKey])
	return c
}

func getPassword(secret *corev1.Secret) string {
	if secret.Data == nil {
		return ""
	}
	return string(secret.Data[passwordKey])
}
