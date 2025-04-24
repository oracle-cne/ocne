// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package olvm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/oracle-cne/ocne/pkg/cluster/template/common"
	"github.com/oracle-cne/ocne/pkg/commands/cluster/start"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	log "github.com/sirupsen/logrus"
	capiclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

// Delete deletes the CAPI resources in the bootstrap cluster which results in the CAPI cluster being deleted.
func (cad *OlvmDriver) Delete() error {
	log.Debugf("Entering Delete for CAPI cluster %s", cad.ClusterConfig.Name)
	cad.Deleted = true
	if cad.FromTemplate {
		cdi, err := common.GetTemplate(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
		cad.ClusterResources = cdi
	}

	// Get the namespace.  This is done by finding the metadata
	// for the Cluster resource.
	clusterObj, err := cad.getClusterObject()
	if err != nil {
		return err
	}

	// No need to check if the label exists again.  The filter function
	// already verified that.
	cad.ResourceNamespace = clusterObj.GetNamespace()
	clusterName := clusterObj.GetName()

	// If this is a self-managed cluster, pivot back into the bootstrap cluster.
	if cad.ClusterConfig.Providers.Oci.SelfManaged {
		log.Infof("Migrating Cluster API resources to bootstrap cluster")
		capiClient, err := capiclient.New(context.TODO(), "")
		if err != nil {
			return nil
		}

		err = capiClient.Move(context.TODO(), capiclient.MoveOptions{
			FromKubeconfig: capiclient.Kubeconfig{Path: cad.KubeConfig, Context: ""},
			ToKubeconfig:   capiclient.Kubeconfig{Path: cad.BootstrapKubeConfig, Context: ""},
			Namespace:      cad.ResourceNamespace,
			DryRun:         false,
		})
		if err != nil {
			return err
		}
	}

	err = cad.deleteCluster(clusterName, clusterObj.GetNamespace())
	if err != nil {
		return err
	}

	// delete the capi cluster kubeconfig
	if err = os.Remove(cad.KubeConfig); err != nil {
		// log the error but keep going
		log.Errorf("Error deleting capi kubeconfig file %s: %v",
			cad.KubeConfig, err)
	}

	_, clientIface, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	secretName := cad.credSecretName()
	if err = k8s.DeleteSecret(clientIface, cad.ClusterConfig.Providers.Olvm.Namespace, secretName); err != nil {
		return fmt.Errorf("Error deleting oVirt credential secret %s/%s: %v",
			cad.ClusterConfig.Providers.Olvm.Namespace, secretName, err)
	}

	cmName := cad.caConfigMapName()
	if err = k8s.DeleteConfigmap(clientIface, cad.ClusterConfig.Providers.Olvm.Namespace, cmName); err != nil {
		return fmt.Errorf("Error deleting oVirt CA configmap %s/%s: %v",
			cad.ClusterConfig.Providers.Olvm.Namespace, secretName, err)
	}

	return nil
}

// Close stops the ephemeral cluster as needed.
func (cad *OlvmDriver) Close() error {
	// There needs to be some logic to figure out when a cluster
	// is done being deleted.  It is not reasonable to develop
	// this against the OCI CAPI provider because it is unreliable
	// when deleting clusters.  For now, leave the ephemeral one
	// behind so that deletion can continue in the background.
	if cad.Deleted {
		return nil
	}

	if cad.Ephemeral && cad.ClusterConfig.Providers.Oci.SelfManaged {
		err := start.StopEphemeralCluster(cad.Config, cad.ClusterConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteCluster deletes the CAPI cluster resource.
func (cad *OlvmDriver) deleteCluster(clusterName string, clusterNs string) error {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	log.Infof("Deleting Cluster %s/%s", clusterNs, clusterName)
	err = k8s.DeleteResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, clusterNs)
	if err != nil {
		return err
	}

	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: "Waiting for deletion",
			WaitFunction: func(i interface{}) error {
				return cad.waitForClusterDeletion(clusterName, clusterNs)
			},
		},
	})

	if haveError {
		return fmt.Errorf("Error deleting cluster")
	}
	return nil
}

// waitForClusterDeletion waits for the cluster to be deleted.
func (cad *OlvmDriver) waitForClusterDeletion(clusterName string, clusterNs string) error {
	restConfig, _, err := client.GetKubeClient(cad.BootstrapKubeConfig)
	if err != nil {
		return err
	}

	_, _, err = util.LinearRetryTimeout(func(i interface{}) (interface{}, bool, error) {
		u, err := k8s.GetResourceByIdentifier(restConfig, "cluster.x-k8s.io", "v1beta1", "Cluster", clusterName, clusterNs)
		if u != nil {
			log.Debugf("Found cluster %s/%s with UID %s", clusterNs, clusterName, u.GetUID())
		} else {
			log.Debugf("Resource for cluster %s/%s was nil", clusterNs, clusterName)
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, false, nil
			}

			return nil, false, err
		}

		return nil, false, fmt.Errorf("Cluster %s/%s is not yet deleted", clusterNs, clusterName)

	}, nil, 20*time.Minute)
	return err
}
