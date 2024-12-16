// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ovclient

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cacrtKey = "ca.crt"
)

// GetOvirtCA validates the CA configmap and returns the CA
// The configmap is optional since the CA is only needed for self-signed certs or certs that are not
// in the local trust store.
func GetOvirtCA(cli ctrlclient.Client, nsn types.NamespacedName) (string, error) {
	// CA configmap is optional
	if nsn.Name == "" {
		return "", nil
	}

	cm := &corev1.ConfigMap{}
	if err := cli.Get(context.TODO(), nsn, cm); err != nil {
		if k8serrors.IsNotFound(err) {
			err = fmt.Errorf("The oVirt CA configmap %v is missing", nsn)
			log.Error(err.Error())
			return "", err
		}
		err = fmt.Errorf("Failed to the oVirt CA configmap %v: %v", nsn, err)
		log.Error(err.Error())
		return "", err
	}

	if cm.Data == nil {
		err := fmt.Errorf("CA configmap %v is missing data field", nsn)
		log.Error(err.Error())
		return "", err
	}

	ca := cm.Data[cacrtKey]
	if ca == "" {
		err := fmt.Errorf("CA configmap %v is missing ca.crt field", nsn)
		log.Error(err.Error())
		return "", err
	}
	return ca, nil
}
