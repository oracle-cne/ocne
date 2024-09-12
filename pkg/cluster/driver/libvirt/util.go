// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	libvirt "github.com/digitalocean/go-libvirt"
	log "github.com/sirupsen/logrus"

	"github.com/oracle-cne/ocne/pkg/cluster/types"
	configtypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/k8s"
	"github.com/oracle-cne/ocne/pkg/k8s/client"
	"github.com/oracle-cne/ocne/pkg/util"
)

// getDomainName generates the name of a libvirt domain based on
// the node role and number of nodes
func getDomainName(clusterName string, role types.NodeRole, num int) string {
	return fmt.Sprintf("%s-%s-%d", clusterName, role, num)
}

// getResourceNames gives back a set of common resource names
func getResourceNames(libvirtDomainName string) (string, string) {
	libvirtResourcePrefix := libvirtDomainName
	libvirtVolumeName := fmt.Sprintf("%s.qcow2", libvirtResourcePrefix)
	ignitionVolumeName := fmt.Sprintf("%s-init.ign", libvirtDomainName)

	return libvirtVolumeName, ignitionVolumeName
}

// isDomainFromCluster takes in a domain name and a name of a cluster and returns if the
func isDomainFromCluster(libvirtDomainName string, clusterName string) bool {
	splitDomainName := strings.Split(libvirtDomainName, "-")
	// All OCNE domains will either be clusterName-control-plane-integer or clusterName-worker-integer
	if len(splitDomainName) < 3 {
		return false
	}
	// Checks the last element to determine if it is an integer, as all domains spun up by OCNE will end with an integer
	lastElement := splitDomainName[len(splitDomainName)-1]
	_, err := strconv.Atoi(lastElement)
	if err != nil {
		return false
	}
	potentialName := ""
	// This checks to see if the domain is of the form clusterName-worker-integer and attempts to reconstruct the clusterName
	if splitDomainName[len(splitDomainName)-2] == "worker" {
		for i := 0; i <= len(splitDomainName)-3; i++ {
			potentialName = potentialName + splitDomainName[i] + "-"
		}
		// This checks to see if the domain is of the form clusterName-control-plane-integer and attempts to reconstruct the clusterName
	} else if len(splitDomainName) > 3 && splitDomainName[len(splitDomainName)-2] == "plane" && splitDomainName[len(splitDomainName)-3] == "control" {
		for i := 0; i <= len(splitDomainName)-4; i++ {
			potentialName = potentialName + splitDomainName[i] + "-"
		}
	} else {
		return false
	}
	// This removes the trailing - from the potential name parsed from the domain and checks to see if it matches the actual name
	potentialName = strings.TrimSuffix(potentialName, "-")
	return clusterName == potentialName
}

// isClusterUp checks to see if a cluster is up.  If the expectation is that
// cluster is on its way up, wait for a while.  If the expectation is that
// it was already up, just check once.
func isClusterUp(kubeconfig string, wait bool) (bool, error) {
	// Get a kubernetes client
	_, kubeClient, err := client.GetKubeClient(kubeconfig)
	if err != nil {
		return false, err
	}

	if wait {
		_, err = k8s.WaitUntilGetNodesSucceeds(kubeClient)
	} else {
		_, err = k8s.GetNodeList(kubeClient)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// setDomainRunningIfExists finds a domain by name.  If it finds one,
// it will try to put it into the 'running' state.  If it is stopped,
// it will start it.  If it is already running, it will be left alone.
// Returns true if the domain is trying to run, otherwise it returns
// false.
func setDomainRunningIfExists(l *libvirt.Libvirt, name string) (bool, bool, error) {
	dom, err := l.DomainLookupByName(name)
	if checkLibvirtError(err, libvirt.ErrNoDomain) {
		return false, false, nil
	} else if err != nil {
		return false, false, err
	}

	state, _, _, _, _, err := l.DomainGetInfo(dom)
	if err != nil {
		return false, false, err
	}

	if libvirt.DomainState(state) == libvirt.DomainRunning {
		return true, true, nil
	}

	err = l.DomainCreate(dom)
	if err != nil {
		return false, false, err
	}

	log.Debugf("Started domain %s", name)
	return true, false, nil
}

// removeDomain does all things required to undefine a domain.  If
// the domain is running, it will be stopped.  Stopped domains are
// undefined
func removeDomain(l *libvirt.Libvirt, name string, arch string) error {
	dom, err := l.DomainLookupByName(name)

	// If the domain does not exist, job done.
	if checkLibvirtError(err, libvirt.ErrNoDomain) {
		return nil
	} else if err != nil {
		return err
	}

	// Stop the VM.  The intent is to undefine the domain
	// so it's not important that there is a graceful shutdown.
	// It's also not important that is was running.  So if it
	// can't be stopped due to being in a non-running state,
	// it is possible to move on to undefining.
	err = l.DomainDestroy(dom)
	if err != nil && !checkLibvirtError(err, libvirt.ErrOperationInvalid) {
		return err
	}

	// Finally, remove the domain
	err = l.DomainUndefineFlags(dom, libvirt.DomainUndefineNvram)
	return err
}

// createBaseImageVolumeFromImagesPool dynamically creates a volume using a template for a libvirt VM
func createVolumeFromImagesPool(node configtypes.Node, l *libvirt.Libvirt, pool *libvirt.StoragePool, ocneVolumeName string, bootVolumeName string) ([]string, error) {
	listOfNewlyCreatedVolumes := []string{}
	poolPath, err := GetStoragePoolPath(l, pool)
	if err != nil {
		return nil, err
	}
	resourceUnits, size, err := createLibvirtCapacityInfo(node.Storage)
	if err != nil {
		return nil, err
	}
	volumeInformation := Volume{
		filepath.Join(poolPath, ocneVolumeName),
		ocneVolumeName,
		filepath.Join(poolPath, ocneVolumeName),
		filepath.Join(poolPath, bootVolumeName),
		uint64(size),
		resourceUnits,
		"qcow2",
	}
	tmpl, err := template.New("volume-template").Parse(volumeTemplate)
	if err != nil {
		return listOfNewlyCreatedVolumes, err
	}
	var templateBuffer bytes.Buffer
	err = tmpl.Execute(&templateBuffer, volumeInformation)
	if err != nil {
		return listOfNewlyCreatedVolumes, err
	}
	xmlStringToWrite := templateBuffer.String()
	storageVolume, err := l.StorageVolCreateXML(*pool, xmlStringToWrite, 0)
	if err != nil {
		return listOfNewlyCreatedVolumes, err
	}
	listOfNewlyCreatedVolumes = append(listOfNewlyCreatedVolumes, storageVolume.Key)

	return listOfNewlyCreatedVolumes, nil
}

// createDomainFromTemplate dynamically creates a domain from a template and defines and spins up a libvirt VM
func createDomainFromTemplate(node configtypes.Node, l *libvirt.Libvirt, domainInformation *Domain) error {
	domainInformation.CPUs = node.CPUs
	resourceUnits, size, err := createLibvirtCapacityInfo(node.Memory)
	if err != nil {
		return err
	}
	domainInformation.Memory = size
	domainInformation.MemoryCapacityUnit = resourceUnits
	domainInformation.CPUArch, err = getLibvirtCPUArchitecture(l)
	if err != nil {
		return err
	}
	xmlStringToWrite, err := util.TemplateToString(domainTemplate, domainInformation)
	if err != nil {
		return err
	}

	dom, err := l.DomainDefineXML(xmlStringToWrite)
	l.DomainCreate(dom)
	return err
}

// getVolumePath returns the path to the file that backs a volume in
// a storage pool
func getVolumePath(l *libvirt.Libvirt, pool *libvirt.StoragePool, volumeName string) (string, error) {
	vol, err := l.StorageVolLookupByName(*pool, volumeName)
	if err != nil {
		return "", err
	}

	volXml, err := l.StorageVolGetXMLDesc(vol, 0)
	if err != nil {
		return "", err
	}

	var volDesc struct {
		Target struct {
			Path string `xml:"path"`
		} `xml:"target"`
	}

	err = xml.Unmarshal([]byte(volXml), &volDesc)
	return volDesc.Target.Path, err
}

// uploadInitialIgnitionFileToOL8Instance takes the local Ignition file generated from the machine and uses scp to transport the file to the remote instance
// It then runs a copy command through ssh to place the file in the /var/lib/libvirt/images directory
func uploadInitialIgnitionFile(l *libvirt.Libvirt, ignitionBytes []byte, pool *libvirt.StoragePool, volumeName string) error {
	size := uint64(len(ignitionBytes))

	return TransferToPool(l, bytes.NewReader(ignitionBytes), pool, volumeName, "raw", true, size)
}

// createLibvirtCapacityInfo takes in a kubernetes style resource string and returns as a string the resource unit
// and the  number of those resource units to be allocated, such as (GB, 10), along with an error
func createLibvirtCapacityInfo(kubernetesResourceString string) (string, int, error) {
	var position int
	kubernetesResourceString = kubernetesResourceString + "B"
	for i, char := range kubernetesResourceString {
		if unicode.IsLetter(char) {
			position = i
			break
		}
	}
	sizeString := kubernetesResourceString[0:position]
	unit := kubernetesResourceString[position:]
	size, err := strconv.Atoi(sizeString)
	return unit, size, err

}

func getLibvirtCPUArchitecture(l *libvirt.Libvirt) (string, error) {
	fullXML, err := l.Capabilities()
	if err != nil {
		return "", err
	}

	var Capabilities struct {
		HostStruct struct {
			CPUStruct struct {
				Arch string `xml:"arch"`
			} `xml:"cpu"`
		} `xml:"host"`
	}
	err = xml.Unmarshal(fullXML, &Capabilities)
	if err != nil {
		return "", err
	}
	return Capabilities.HostStruct.CPUStruct.Arch, nil
}
