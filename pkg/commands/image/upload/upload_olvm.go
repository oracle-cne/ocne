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
	"github.com/oracle-cne/ocne/pkg/util/logutils"
	"github.com/oracle-cne/ocne/pkg/util/oci"
	"os"
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

	// Create imagetransfer
	iTran, err := createImageTransfer(ovcli, disk.Id)
	if err != nil {
		return err
	}
	defer ovit.DeleteImageTransfer(ovcli, iTran.Id)

	// Wait for imagetransfer stage transferring
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
	_, err = ovdisk.CreateDisk(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func createImageTransfer(ovcli *ovclient.Client, diskID string) (*ovit.ImageTransfer, error) {
	// Create image transfer
	req := ovit.CreateImageTransferRequest{
		Disk: ovit.Disk{
			Id: diskID,
		},
		Direction: "upload",
	}

	_, err := ovit.CreateImageTransfer(ovcli, &req)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func waitForImageTransferReady(ovcli *ovclient.Client, transferID string) error {
	log.Infof("Waiting for image transfer to become ready")
	const maxTries = 60
	for i := range maxTries {
		iTran, err := ovit.GetImageTransfer(ovcli, transferID)
		if err != nil {
			return err
		}
		if iTran.Phase == ovit.PhaseTransferring {
			return nil
		}
		time.Sleep(1 + time.Second)
	}

	//failed := logutils.WaitFor(logutils.Info, []*logutils.Waiter{
	//	&logutils.Waiter{
	//		Args:    &options,
	//		Message: "Uploading image to object storage",
	//		WaitFunction: func(uIface interface{}) error {
	//			uo, _ := uIface.(*UploadOptions)
	//			return oci.UploadObject(uo.BucketName, options.filename, uo.size, uo.file, nil)
	//		},
	//	},
	//})
	//if failed {
	//	return "", "", fmt.Errorf("Failed to upload image to object storage")
	//}
	return nil
}

//func isImageTransferReady(ovcli *ovclient.Client, diskID string) (bool, error) {
//
//
//}

func uploadingImage(ovcli *ovclient.Client, proxy_url string) error {
	log.Infof("Uploading image to %s", proxy_url)
	return nil
}
