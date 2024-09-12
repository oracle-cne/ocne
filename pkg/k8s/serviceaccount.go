// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/util"
)

type waitForSA struct {
	accountName string
	namespace   string
	client      kubernetes.Interface
}

// WaitForServiceAccount waits for a service account to exist.
func WaitForServiceAccount(client kubernetes.Interface, accountName string, namespace string) error {
	sa := waitForSA{
		accountName: accountName,
		namespace:   namespace,
		client:      client,
	}
	_, _, err := util.LinearRetry(func(sai interface{}) (interface{}, bool, error) {
		sa, _ := sai.(*waitForSA)
		_, err := sa.client.CoreV1().ServiceAccounts(sa.namespace).Get(context.TODO(), sa.accountName, metav1.GetOptions{})
		return nil, false, err
	}, &sa)
	return err
}
