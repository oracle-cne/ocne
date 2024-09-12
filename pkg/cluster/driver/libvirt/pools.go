// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package libvirt

import (
	"encoding/xml"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/digitalocean/go-libvirt"

	"github.com/oracle-cne/ocne/pkg/constants"
	"github.com/oracle-cne/ocne/pkg/util"
)

type StoragePoolPermissions struct {
	Mode string `xml:"mode"`
}

type StoragePoolTarget struct {
	Path        string                 `xml:"path"`
	Permissions StoragePoolPermissions `xml:"permissions"`
}

type StoragePoolDir struct {
	Path string `xml:"path,attr"`
}

type StoragePoolSource struct {
	//Dir StoragePoolDir `xml:"dir"`
}

type StoragePool struct {
	XMLName xml.Name          `xml:"pool"`
	Name    string            `xml:"name"`
	Type    string            `xml:"type,attr"`
	Target  StoragePoolTarget `xml:"target"`
	Source  StoragePoolSource `xml:"source"`
}

func storagePoolToXml(pool *StoragePool) (string, error) {
	retBytes, err := xml.Marshal(pool)
	if err != nil {
		return "", err
	}
	return string(retBytes), nil
}

func xmlToStoragePool(xmlDesc string) (*StoragePool, error) {
	ret := StoragePool{}
	err := xml.Unmarshal([]byte(xmlDesc), &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func refreshStoragePools(l *libvirt.Libvirt) error {
	pools, _, err := l.ConnectListAllStoragePools(int32(1), libvirt.ConnectListStoragePoolsActive)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		if err = l.StoragePoolRefresh(pool, uint32(0)); err != nil {
			return err
		}
	}
	return nil
}

// getDefaultPoolPath returns a path that can be used for
// storage based on the connection URI.  If the URI points
// to a session (read: unprivileeged) and is local, return
// a path that is within the users home directory.  Otherwise,
// use the typical path user /var/lib/libvirt.
func getDefaultPoolPath(uri *url.URL) (string, bool, error) {
	poolPath := constants.StoragePoolPath
	_, isLocal, err := util.ResolveURIToIP(uri)
	if err != nil {
		return "", isLocal, err
	}

	// If the session is local and truly a user-session, use a path within
	// the users home directory for images.  On Oracle Linux, using
	// qemu:///session for any user resolves to qemu:///system.  The only
	// meaningful use case for local directories is a local session where
	// the host is Mac.
	if strings.Contains(uri.Path, "session") && isLocal && runtime.GOOS == "darwin" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", isLocal, nil
		}
		poolPath = filepath.Join(homedir, constants.UserStoragePoolPath)
	}
	return poolPath, isLocal, nil
}

// GetStoragePoolPath gives back the path for a storage pool.  If the storage
// pool is not backed by a path, an empty string is returned along with an
// error.
func GetStoragePoolPath(l *libvirt.Libvirt, pool *libvirt.StoragePool) (string, error) {
	poolDesc, err := l.StoragePoolGetXMLDesc(*pool, 0)
	if err != nil {
		return "", err
	}

	p, err := xmlToStoragePool(poolDesc)
	if err != nil {
		return "", nil
	}

	return p.Target.Path, nil
}

// StartStoragePool starts a storage pool if it is not already started.
func StartStoragePool(l *libvirt.Libvirt, pool *libvirt.StoragePool) error {
	// If the pool is already running, just bail.
	state, _, _, _, err := l.StoragePoolGetInfo(*pool)
	if err != nil {
		return err
	} else if libvirt.StoragePoolState(state) == libvirt.StoragePoolRunning {
		return nil
	}

	// Otherwise, start the pool.
	if err = l.StoragePoolCreate(*pool, libvirt.StoragePoolCreateNormal); err != nil {
		return err
	}
	err = l.StoragePoolSetAutostart(*pool, int32(1))
	if err != nil {
		return err
	}
	err = refreshStoragePools(l)
	if err != nil {
		return err
	}
	return nil
}

// FindStoragePool gives back a reasonable place to locate
// volumes. First it checks to see if a storage pool is specified. If specified,
// then it proceeds to use it. Otherwise, it checks to see if a pool that matches the default
// configuration already exists. If so, that one is used.  If not, it
// will scan all pools for something that looks acceptable.  "Acceptable"
// is intentionally not defined.  If an acceptable option does not exist,
// no volume is returned
func FindStoragePool(l *libvirt.Libvirt, storagePool, poolPath string) (*libvirt.StoragePool, error) {
	var poolName string
	if storagePool != "" {
		poolName = storagePool
	} else {
		// Use the default.
		poolName = constants.StoragePool
	}
	pool, err := l.StoragePoolLookupByName(poolName)
	if err == nil {
		err = StartStoragePool(l, &pool)
		if err != nil {
			return nil, err
		}
		return &pool, nil
	} else if err != nil && !checkLibvirtError(err, libvirt.ErrNoStoragePool) {
		return nil, err
	}

	// The typical pool was not found.  Iterate through the pools to find one that
	// looks usable.  In this case, that means that it uses a filesystem to store
	// volumes rather than some other backend.
	pools, _, err := l.ConnectListAllStoragePools(int32(1), libvirt.ConnectListStoragePoolsActive)
	if err != nil {
		return nil, err
	}

	path := ""
	for _, p := range pools {
		thisPoolPath, err := GetStoragePoolPath(l, &p)
		if err != nil {
			return nil, err
		}

		if thisPoolPath == poolPath {
			path = poolPath
			pool = p
			break
		}
	}

	if path != "" {
		return &pool, nil
	}
	return nil, nil
}

// FindOrCreateStoragePool gives back a reasonable place to locate
// volumes. First it checks to see if a storage pool is specified. If specified,
// then it proceeds to use it. Otherwise, it checks to see if a pool that matches the default
// configuration already exists.  If so, that one is used.  If not, it
// will scan all pools for something that looks acceptable.  "Acceptable"
// is intentionally not defined.  If an acceptable option does not exist,
// a new pool will be created with useful defaults.
func FindOrCreateStoragePool(l *libvirt.Libvirt, uri *url.URL, storagePool string) (*libvirt.StoragePool, error) {
	// Assume a particular path for the pool.  This isn't great, but it
	// avoids a couple compatibility issues between Linux and Mac.
	// Specifically, libvirt from brew creates storage pools with different
	// names that what would be expected from OL.  This is going to have to
	// improved at some point, but this is fine-ish for now.
	poolPath, isLocal, err := getDefaultPoolPath(uri)
	if err != nil {
		return nil, err
	}

	// If this is a local pool, create the directory for the user.  libvirt
	// won't do this automatically.  os.MkdirAll only makes directories
	// that don't exist, so there's no reason to check ahead of time.
	//
	// On remote OL sysystems there is no need to create the directory.
	// /var/lib/libvirt/images is packaged in the virt:kvm_utils3 module
	// and will already exist by default.
	if isLocal {
		err = os.MkdirAll(poolPath, 0750)
		if err != nil {
			return nil, err
		}
	}

	pool, err := FindStoragePool(l, storagePool, poolPath)
	if err != nil {
		return nil, err
	} else if pool != nil {
		return pool, nil
	}

	// No viable pool was found.  Make one.
	newPool, err := CreateStoragePool(l, constants.StoragePool, poolPath)
	if err != nil {
		return nil, err
	}
	return newPool, nil
}

// CreateStoagePool creates a storage pool and returns the libvirt object
func CreateStoragePool(l *libvirt.Libvirt, name string, path string) (*libvirt.StoragePool, error) {
	newPool := StoragePool{
		Name:   name,
		Type:   "dir",
		Source: StoragePoolSource{
			//Dir: StoragePoolDir {
			//	Path: path,
			//},
		},
		Target: StoragePoolTarget{
			Path: path,
			Permissions: StoragePoolPermissions{
				Mode: "0750",
			},
		},
	}
	xmlStringToWrite, err := storagePoolToXml(&newPool)
	if err != nil {
		return nil, err
	}

	pool, err := l.StoragePoolDefineXML(xmlStringToWrite, uint32(0))
	if err != nil {
		return nil, err
	}
	err = StartStoragePool(l, &pool)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

// deleteVolume ensures that a volume does not exist in a pool.  If it's not
// there, then this function returns without error.  If it is, it will be
// delete from the pool.
func deleteVolume(l *libvirt.Libvirt, pool *libvirt.StoragePool, volName string) error {
	vol, err := l.StorageVolLookupByName(*pool, volName)
	if checkLibvirtError(err, libvirt.ErrNoStorageVol) {
		return nil
	} else if err != nil {
		return err
	}

	return l.StorageVolDelete(vol, libvirt.StorageVolDeleteNormal)
}
