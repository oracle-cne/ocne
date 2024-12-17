// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"github.com/oracle-cne/ocne/pkg/cluster/driver/olvm"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	ovdisk "github.com/oracle-cne/ocne/pkg/ovirt/rest/disk"
	ovsd "github.com/oracle-cne/ocne/pkg/ovirt/rest/storagedomain"
	"k8s.io/apimachinery/pkg/types"
)

// UploadOlvm uploads a boot image to an OLVM disk.
func UploadOlvm(o UploadOptions) error {
	// get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}

	oCluster := o.ClusterConfig.Providers.Olvm.OlvmCluster

	// Get OvClient
	secretNsn := types.NamespacedName{
		Namespace: o.ClusterConfig.Providers.Olvm.Namespace,
		Name:      olvm.CredSecretName(o.ClusterConfig),
	}
	caConfigMapNsn := types.NamespacedName{
		Namespace: o.ClusterConfig.Providers.Olvm.Namespace,
		Name:      olvm.CaConfigMapName(o.ClusterConfig),
	}
	ovcli, err := ovclient.GetOVClient(kubeClient, secretNsn, caConfigMapNsn, oCluster.OVirtAPI.ServerURL)
	if err != nil {
		return err
	}

	// Get storage name
	sd, err := ovsd.GetStorageDomain(ovcli, oCluster.OVirtOck.StorageDomainName)
	if err != nil {
		return err
	}

	// Create disk
	cd := ovdisk.CreateDiskRequest{
		StorageDomainList: ovdisk.StorageDomainList{
			StorageDomains: []ovdisk.StorageDomain{
				{Id: sd.Id}},
		},
		Name:            oCluster.OVirtOck.DiskName,
		ProvisionedSize: oCluster.OVirtOck.DiskSize,
		Format:          "cow",
		Backup:          "none",
	}
	_, err = ovdisk.CreateDisk(ovcli, &cd)
	if err != nil {
		return err
	}

	// Create imagetransfer

	// Wait for imagetransfer stage

	// Upload image

	// Finish imagetransfer

	return nil

}
