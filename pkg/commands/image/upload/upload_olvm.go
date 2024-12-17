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
		ovdisk.DeleteDisk(ovcli, disk.Id)
		return err
	}

	// Create imagetransfer
	iTran, err := createImageTransfer(ovcli, disk)
	if err != nil {
		ovdisk.DeleteDisk(ovcli, disk.Id)
		return err
	}

	// Wait for imagetransfer to be ready to transfer
	err = waitForImageTransferPhase(ovcli, iTran.Id, ovit.PhaseTransferring)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Upload the image to the disk
	err = uploadImage(ovcli, iTran.Id)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Finish imagetransfer
	err = ovit.DoImageTransferAction(ovcli, iTran.Id, ovit.ActionFinalize)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Wait until the transfer session was successfully closed, and the targeted image was verified and ready to be used
	err = waitForImageTransferPhase(ovcli, iTran.Id, ovit.PhaseFinished)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	log.Infof("Successfully uploaded OCK image %s to disk %s", o.ImagePath, disk.Name)
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
		Format:          ovdisk.FormatCow,
		Backup:          ovdisk.BackupNone,
	}
	disk, err := ovdisk.CreateDisk(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func createImageTransfer(ovcli *ovclient.Client, disk *ovdisk.Disk) (*ovit.ImageTransfer, error) {
	// Create image transfer
	req := ovit.CreateImageTransferRequest{
		Name: fmt.Sprintf("Upload OCK image to disk %s, ID %s", disk.Name, disk.Id),
		Disk: ovit.Disk{
			Id: disk.Id,
		},
		Direction:     ovit.DirectionUpload,
		TimeoutPolicy: ovit.TimeoutPolicy,
	}

	iTran, err := ovit.CreateImageTransfer(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return iTran, nil
}

func waitForImageTransferPhase(ovcli *ovclient.Client, transferID string, phase string) error {
	log.Infof("Waiting for image transfer to become ready")
	const maxTries = 60
	for i := 0; i < maxTries; i++ {
		iTran, err := ovit.GetImageTransfer(ovcli, transferID)
		if err != nil {
			return err
		}
		if iTran.Phase == phase {
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

func uploadImage(ovcli *ovclient.Client, proxy_url string) error {
	log.Infof("Uploading image to %s", proxy_url)
	return nil
}

// cleanup is a best effort
func cleanup(ovcli *ovclient.Client, transferID string, diskID string) {
	iTran, err := ovit.GetImageTransfer(ovcli, transferID)
	if err != nil || iTran == nil {
		ovdisk.DeleteDisk(ovcli, diskID)
		return
	}
	ovit.DoImageTransferAction(ovcli, transferID, ovit.ActionCancel)
	err = waitForImageTransferPhase(ovcli, transferID, ovit.PhaseCancelled)
	if err != nil {
		return
	}
	ovit.DeleteImageTransfer(ovcli, transferID)
	ovdisk.DeleteDisk(ovcli, diskID)
}
