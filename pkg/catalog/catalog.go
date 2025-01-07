// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package catalog

import (
	"fmt"

	"github.com/oracle-cne/ocne/pkg/config/types"

	"k8s.io/client-go/kubernetes"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
)

// WaitForInternalCatalogInstall waits for the internal catalog to be installed
func WaitForInternalCatalogInstall(kubeClient kubernetes.Interface, logFn func(string)) error {
	haveError := logutils.WaitFor(logFn, []*logutils.Waiter{
		{
			Message: "Waiting for Oracle Catalog to start",
			WaitFunction: func(i interface{}) error {
				err := k8s.WaitForDeployment(kubeClient, constants.CatalogNamespace, constants.CatalogServiceName, 1)
				return err
			},
		},
	})
	if haveError {
		return fmt.Errorf("Oracle Catalog failed to start")
	}
	return nil
}

// NewCommunityCatalog returns the definition for adding the community catalog
func NewCommunityCatalog() types.Catalog {
	protocol := ArtifacthubProtocol
	uri := constants.CommunityCatalogURI
	name := constants.CommunityCatalogName
	namespace := constants.CommunityCatalogNamespace
	return types.Catalog{
		Protocol:  &protocol,
		URI:       &uri,
		Name:      &name,
		Namespace: &namespace,
	}
}
