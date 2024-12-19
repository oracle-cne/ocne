// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upload

import (
	"fmt"
	"github.com/docker/go-units"
	"github.com/oracle-cne/ocne/pkg/cluster/driver/olvm"
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/ovirt/ovclient"
	ovdisk "github.com/oracle-cne/ocne/pkg/ovirt/rest/disk"
	ovit "github.com/oracle-cne/ocne/pkg/ovirt/rest/imagetransfer"
	ovsd "github.com/oracle-cne/ocne/pkg/ovirt/rest/storagedomain"
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// UploadOlvm uploads a boot image to an OLVM disk.
// see https://ovirt.github.io/ovirt-imageio/images.html
func UploadOlvm(o UploadOptions) error {
	// get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(o.KubeConfigPath)
	if err != nil {
		return err
	}

	oCluster := o.ClusterConfig.Providers.Olvm.OlvmCluster

	// Get OvClient
	ca, err := olvm.GetCA(&o.ClusterConfig.Providers.Olvm)
	if err != nil {
		return err
	}
	ovcli, err := ovclient.GetOVClient(kubeClient, ca, oCluster.OVirtAPI.ServerURL)
	if err != nil {
		return err
	}

	log.Infof("Starting uploaded OCK image `%s` to disk `%s` in storage domain `%s`", o.ImagePath, oCluster.OVirtOck.DiskName,
		oCluster.OVirtOck.StorageDomainName)

	fileInfo, err := getImageInfo(o.ImagePath)
	if err != nil {
		return err
	}

	// Create an empty disk in the oVirt storage domain
	disk, err := createDisk(ovcli, oCluster, fileInfo)
	if err != nil {
		return err
	}

	// Wait for disk ready for data transfer
	err = waitForDiskReady(ovcli, disk.Id)
	if err != nil {
		ovdisk.DeleteDisk(ovcli, disk.Id)
		return err
	}

	// Create imagetransfer resource
	iTran, err := createImageTransfer(ovcli, disk, fileInfo)
	if err != nil {
		ovdisk.DeleteDisk(ovcli, disk.Id)
		return err
	}

	// Wait for the imagetransfer resource to be ready to transfer
	err = waitForImageTransferPhase(ovcli, iTran.Id, ovit.PhaseTransferring)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Upload the image to the disk
	err = uploadImageAndWait(ovcli, iTran.ProxyUrl, o.ImagePath, fileInfo, disk)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Wait until the imagetransfer resource reports the total bytes transferred
	fileLenStr := fmt.Sprintf("%v", fileInfo.Size())
	err = waitForImageTransferLenMatch(ovcli, iTran.Id, fileLenStr)
	if err != nil {
		cleanup(ovcli, iTran.Id, disk.Id)
		return err
	}

	// Finalize the image transfer
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

	log.Infof("Successfully uploaded OCK image")
	return nil
}

func createDisk(ovcli *ovclient.Client, oCluster otypes.OlvmCluster, fileInfo os.FileInfo) (*ovdisk.Disk, error) {
	// Get storage name
	sd, err := ovsd.GetStorageDomain(ovcli, oCluster.OVirtOck.StorageDomainName)
	if err != nil {
		return nil, err
	}

	initialSize := fmt.Sprintf("%v", fileInfo.Size())

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
		InitialSize:     initialSize,
	}
	disk, err := ovdisk.CreateDisk(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func createImageTransfer(ovcli *ovclient.Client, disk *ovdisk.Disk, info os.FileInfo) (*ovit.ImageTransfer, error) {
	inactiveTimeoutSecs := fmt.Sprintf("%d", 180)

	// Create image transfer
	req := ovit.CreateImageTransferRequest{
		Name: fmt.Sprintf("Upload OCK image to disk %s, ID %s", disk.Name, disk.Id),
		Disk: ovit.Disk{
			Id: disk.Id,
		},
		Direction:         ovit.DirectionUpload,
		TimeoutPolicy:     ovit.TimeoutPolicy,
		InactivityTimeout: inactiveTimeoutSecs,
		Format:            ovdisk.FormatCow,
	}

	iTran, err := ovit.CreateImageTransfer(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return iTran, nil
}

func waitForImageTransferPhase(ovcli *ovclient.Client, transferID string, phase string) error {
	log.Infof("Waiting for image transfer phase %s", phase)
	const maxTries = 60
	var lastPhase string
	for i := 0; i < maxTries; i++ {
		iTran, err := ovit.GetImageTransfer(ovcli, transferID)
		if err != nil {
			return err
		}
		if iTran.Phase == phase {
			return nil
		}
		lastPhase = iTran.Phase
		time.Sleep(1 + time.Second)
	}
	err := fmt.Errorf("Timed out waiting for image transfer phase %s, current phase is %s", phase, lastPhase)
	log.Error(err)
	return err
}

func waitForImageTransferLenMatch(ovcli *ovclient.Client, transferID string, numBytesTransferred string) error {
	const maxTries = 60
	for i := 0; i < maxTries; i++ {
		iTran, err := ovit.GetImageTransfer(ovcli, transferID)
		if err != nil {
			return err
		}
		if iTran.Transferred == numBytesTransferred {
			log.Infof("imagetransfer.Transferred value %s matches the actual number of bytes transferred", iTran.Transferred)
			return nil
		}
		time.Sleep(1 + time.Second)
	}
	err := fmt.Errorf("Timed out waiting for image transfer length to match")
	log.Error(err)
	return err
}

func waitForDiskReady(ovcli *ovclient.Client, diskID string) error {
	log.Infof("Waiting for disk status to be OK")
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
	err := fmt.Errorf("Timed out waiting for disk %s status to be OK", diskID)
	log.Error(err)
	return err
}

// cleanup is a best effort
func cleanup(ovcli *ovclient.Client, transferID string, diskID string) {
	log.Infof("Cleaning up image transfer due to failure")

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

func getImageInfo(imagePath string) (os.FileInfo, error) {
	path, err := file.AbsDir(imagePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func uploadImageAndWait(ovcli *ovclient.Client, proxy_url string, imagePath string, info os.FileInfo, disk *ovdisk.Disk) error {
	haveError := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
		{
			Message: fmt.Sprintf("Uploading image %s with %v bytes to %s", imagePath, info.Size(), disk.Name),
			WaitFunction: func(i interface{}) error {
				err := uploadImage(ovcli, proxy_url, imagePath, info)
				return err
			},
		},
	})
	if haveError {
		return fmt.Errorf("Oracle image upload failed")
	}
	return nil
}

func uploadImage(ovcli *ovclient.Client, proxy_url string, imagePath string, info os.FileInfo) error {
	path, err := file.AbsDir(imagePath)
	if err != nil {
		return err
	}

	reader, err := os.Open(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = ovit.UploadFile(ovcli, proxy_url, reader, info.Size())
	if err != nil {
		return err
	}
	return nil
}
