// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ovclient

import (
	"fmt"
	"github.com/oracle-cne/ocne/pkg/k8s"
	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	cacrtKey = "ca.crt"
)

// GetOvirtCA validates the CA configmap and returns the CA
// The configmap is optional since the CA is only needed for self-signed certs or certs that are not
// in the local trust store.
func GetOvirtCA(cli kubernetes.Interface, nsn types.NamespacedName) (string, error) {
	// CA configmap is optional
	if nsn.Name == "" {
		return "", nil
	}
	cm, err := k8s.GetConfigmap(cli, nsn.Namespace, nsn.Name)
	if err != nil {
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
