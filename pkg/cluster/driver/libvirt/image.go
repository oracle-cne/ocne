// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"bytes"
	"io"
	"text/template"

	libvirt "github.com/digitalocean/go-libvirt"

	"github.com/oracle-cne/ocne/pkg/image"
)

// checkLibvirtError looks at an error code from the libvirt API and
// determines if it is a particular error.
func checkLibvirtError(err error, ec libvirt.ErrorNumber) bool {
	if err == nil {
		return false
	}

	e, ok := err.(libvirt.Error)
	if !ok {
		return false
	}

	return e.Code == uint32(ec)
}

// TransferBaseImage uses the libvirt API to upload the boot.qcow2 from a
// container image to the target libvirtd instance, locating in the given
// Pool and assigning in the given name.
func TransferBaseImage(l *libvirt.Libvirt, imageName string, pool *libvirt.StoragePool, volumeName string, arch string) error {
	tarStream, closer, err := image.EnsureBaseQcow2Image(imageName, arch)
	if err != nil {
		return err
	}
	defer closer()

	// TODO: Come up with some logic that tests if re-upload is necessary.
	//       It is disabled for now to avoid uploading 2ish gigs of data
	//       every time someone wants to spin up a cluster.
	return TransferToPool(l, tarStream, pool, volumeName, "qcow2", false, 0)
}

// TransferToPool uses the libvirt API to upload a stream to the target
// libvirtd instance, locating it in the given pool and assigning it the
// given name.  This function does not create the pool if it does not
// already exist.
//
// If reupload is false, the image is not re-transferred if it already
// exists on the target libvirt instance.
//
// If size is non-zero, the volume is resized after upload to the
// given value.
func TransferToPool(l *libvirt.Libvirt, in io.Reader, pool *libvirt.StoragePool, volumeName string, volType string, reupload bool, size uint64) error {
	// Render the volume XML.  This is used to create the volume.  While it
	// is less efficient to do even if the volume already exists, it makes the
	// code a bit easier to read by removing a level of indentation.  Also,
	// the cost of this is trivial compared to the cost of actually uploading
	// and image or waiting for a cluster to start.
	volumeInformation := Volume{
		Name: volumeName,
		Size: size,
		Type: volType,
	}
	tmpl, err := template.New("volume-template").Parse(volumeTemplateNoBacking)
	if err != nil {
		return err
	}
	var templateBuffer bytes.Buffer
	err = tmpl.Execute(&templateBuffer, volumeInformation)
	if err != nil {
		return err
	}
	xmlStringToWrite := templateBuffer.String()

	// Libvirt has some issues resizing files when uploading.  There is probably
	// a good solution available.  For now, though, just delete the volume and
	// recreate it.
	vol, err := l.StorageVolLookupByName(*pool, volumeName)
	if err == nil {
		err = l.StorageVolDelete(vol, 0)
	} else if checkLibvirtError(err, libvirt.ErrNoStorageVol) {
		err = nil
	}
	if err != nil {
		return err
	}

	// Try and create the storage volume.  If the volume already exists,
	// ignore the error.  The goal of this block is to ensure that a volume
	// exists.  The way that happens is immaterial.
	vol, err = l.StorageVolCreateXML(*pool, xmlStringToWrite, 0)
	if checkLibvirtError(err, libvirt.ErrStorageVolExist) {
		// If the caller does not want to reupload the image
		// then we're done here.
		if !reupload {
			return nil
		}

		// Fetch the volume that already exists.  The full
		// details are necessary for StorageVolUpload later
		// on in this function.
		vol, err = l.StorageVolLookupByName(*pool, volumeName)
	}

	if err != nil {
		return err
	}

	// Upload the image.
	return l.StorageVolUpload(vol, in, 0, size, libvirt.StorageVolUploadSparseStream)
}

// DoesBaseImageExist returns if the base image has been previously downloaded or manually placed in the default images pool
func DoesBaseImageExist(l *libvirt.Libvirt, pool *libvirt.StoragePool, bootVolumeName string) bool {
	potentialBaseImageVolume, err := l.StorageVolLookupByName(*pool, bootVolumeName)
	if err != nil {
		return false
	}
	return potentialBaseImageVolume.Name == bootVolumeName && potentialBaseImageVolume.Pool == pool.Name
}
