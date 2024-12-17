// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"fmt"
	"github.com/docker/go-units"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/olvm"
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	ovdisk "github.com/oracle-cne/ocne/pkg/ovirt/rest/disk"
	ovit "github.com/oracle-cne/ocne/pkg/ovirt/rest/imagetransfer"
	ovsd "github.com/oracle-cne/ocne/pkg/ovirt/rest/storagedomain"
	"time"

	log "github.com/sirupsen/logrus"
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

	// Create an empty disk in the oVirt storage domain
	disk, err := createDisk(ovcli, oCluster)
	if err != nil {
		return err
	}

	// Wait for disk ready for data transfer
	err = waitForDiskReady(ovcli, disk.Id)
	if err != nil {
		return err
	}

	// Create imagetransfer
	iTran, err := createImageTransfer(ovcli, disk.Id)
	if err != nil {
		return err
	}
	defer ovit.DeleteImageTransfer(ovcli, iTran.Id)

	// Wait for imagetransfer to be ready to transfer
	err = waitForImageTransferReady(ovcli, iTran.Id)
	if err != nil {
		return err
	}

	// Upload image

	// Finish imagetransfer

	// TODO add doc telling user to create a VM Template
	return nil

}

func createDisk(ovcli *ovclient.Client, oCluster otypes.OlvmCluster) (*ovdisk.Disk, error) {
	// Get storage name
	sd, err := ovsd.GetStorageDomain(ovcli, oCluster.OVirtOck.StorageDomainName)
	if err != nil {
		return nil, err
	}

	// convert disk size to bytes
	diskSizeBytes, err := units.RAMInBytes(oCluster.OVirtOck.DiskSize)
	if err != nil {
		err = fmt.Errorf("Error, DiskSize value %s is an invalid format", oCluster.OVirtOck.DiskSize)
		log.Error(err)
		return nil, err
	}
	diskSizeBytesStr := fmt.Sprintf("%v", diskSizeBytes)

	// Create disk
	req := ovdisk.CreateDiskRequest{
		StorageDomainList: ovdisk.StorageDomainList{
			StorageDomains: []ovdisk.StorageDomain{
				{Id: sd.Id}},
		},
		Name:            oCluster.OVirtOck.DiskName,
		ProvisionedSize: diskSizeBytesStr,
		Format:          "cow",
		Backup:          "none",
	}
	disk, err := ovdisk.CreateDisk(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func createImageTransfer(ovcli *ovclient.Client, diskID string) (*ovit.ImageTransfer, error) {
	// Create image transfer
	req := ovit.CreateImageTransferRequest{
		Disk: ovit.Disk{
			Id: diskID,
		},
		Direction: "upload",
	}

	iTran, err := ovit.CreateImageTransfer(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return iTran, nil
}

func waitForImageTransferReady(ovcli *ovclient.Client, transferID string) error {
	log.Infof("Waiting for image transfer to become ready")
	const maxTries = 60
	for i := 0; i < maxTries; i++ {
		iTran, err := ovit.GetImageTransfer(ovcli, transferID)
		if err != nil {
			return err
		}
		if iTran.Phase == ovit.PhaseTransferring {
			return nil
		}
		time.Sleep(1 + time.Second)
	}
	return fmt.Errorf("Timed out waiting for image transfer to become ready")
}

func waitForDiskReady(ovcli *ovclient.Client, diskID string) error {
	log.Infof("Waiting for image transfer to become ready")
	const maxTries = 60
	for i := 0; i < maxTries; i++ {
		disk, err := ovdisk.GetDisk(ovcli, diskID)
		if err != nil {
			return err
		}
		if disk.Status == ovdisk.StatusOK {
			return nil
		}
		time.Sleep(1 + time.Second)
	}
	return fmt.Errorf("Timed out waiting for disk %s to become ok", diskID)
}

//func isImageTransferReady(ovcli *ovclient.Client, diskID string) (bool, error) {
//
//
//}

func uploadingImage(ovcli *ovclient.Client, proxy_url string) error {
	log.Infof("Uploading image to %s", proxy_url)
	return nil
}
