// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"bytes"
	"debug/elf"
	"fmt"
//	"os"
	"path/filepath"
	"strings"

	ctypes "github.com/containers/image/v5/types"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	log "github.com/sirupsen/logrus"

//	"github.com/diskfs/go-diskfs/filesystem"
//	"github.com/diskfs/go-diskfs/partition/gpt"
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/disk"
//	"github.com/oracle-cne/ocne/pkg/util/linux"
//	"github.com/oracle-cne/ocne/pkg/file"
	"github.com/oracle-cne/ocne/pkg/image"
)

const (
	// Files from the syslinux image.  These lack a
	// leading slash because they come from a tarball.
	IsoLinux = "usr/share/syslinux/isolinux.bin"
	LdLinux = "usr/share/syslinux/ldlinux.c32"
	Libcom = "usr/share/syslinux/libcom32.c32"
	Libutil = "usr/share/syslinux/libutil.c32"
	Vesamenu = "usr/share/syslinux/vesamenu.c32"

	IsoLinuxDest = "isolinux/isolinux.bin"
	LdLinuxDest = "isolinux/ldlinux.c32"
	LibcomDest = "isolinux/libcom32.c32"
	LibutilDest = "isolinux/libutil.c32"
	VesamenuDest = "isolinux/vesamenu.c32"

	// Files from the EFI partition
	BootX64 = "/EFI/BOOT/BOOTX64.EFI"
	GrubX64 = "/EFI/redhat/grubx64.efi"
	MMX64 = "/EFI/redhat/mmx64.efi"

	BootX64Dest = "EFI/BOOT/BOOTX64.EFI"
	GrubX64Dest = "EFI/BOOT/grubx64.efi"
	MMX64Dest = "EFI/BOOT/mmx64.efi"


	// Files from Boot partition
	GrubConfig = "/loader.1/entries/ostree-1-ock.conf"

	EfiGrubConfigDest = "EFI/BOOT/grub.cfg"
	GrubConfigDest = "isolinux/grub.conf"
	KernelDest = "isolinux/vmlinuz"
	InitrdDest = "isolinux/initrd.img"

	ImagesKernelDest = "images/pxeboot/vmlinuz"
	ImagesInitrdDest = "images/pxeboot/initrd.img"

	// Files from Root filesystem
	RootDest = "images/ock.img"

	DefaultDirUid = 0
	DefaultDirGid = 0
	DefaultDirMode = 0755

	DefaultFileUid = 0
	DefaultFileGid = 0
	DefaultFileMode = 0644

	// Executables that need to be copied from the OCK image
	// into the initramfs
	SkopeoPath = "usr/bin/skopeo"
	OstreePath = "usr/bin/ostree"
	RpmOstreePath = "usr/bin/rpm-ostree"
	CutPath = "usr/bin/cut"
	ExtOstreeContainerPath = "usr/libexec/libostree/ext/ostree-container"
	BwrapPath = "usr/bin/bwrap"
	CpioPath = "usr/bin/cpio"

	// Files necessary to use a container runtime
	PolicyPath = "etc/containers/policy.json"

	// Special destinations
	OstreeContainerPath = "usr/bin/ostree-container"
)

var fileMapping = map[string]string{
	IsoLinux: IsoLinuxDest,
	LdLinux: LdLinuxDest,
	Libcom: LibcomDest,
	Libutil: LibutilDest,
	Vesamenu: VesamenuDest,
	BootX64: BootX64Dest,
	GrubX64: GrubX64Dest,
	MMX64: MMX64Dest,
}

var executables = []string{
	SkopeoPath,
	OstreePath,
	RpmOstreePath,
	CutPath,
	ExtOstreeContainerPath,
	BwrapPath,
	CpioPath,
}

func defaultInitramfsContent() map[*util.CpioHeader][]byte {
	return  map[*util.CpioHeader][]byte{
		util.CpioFile("usr/lib/systemd/system/deploy-ock.service", 0755): []byte(deployOckService),
		util.CpioFile("usr/lib/systemd/system/set-config.service", 0755): []byte(setConfigService),
		util.CpioFile("usr/sbin/deploy-ock", 0755): []byte(deployOckScript),
		util.CpioFile("usr/sbin/set-config", 0755): []byte(setConfigScript),
		util.CpioFile("etc/grub.cfg", 0644): []byte(grubCfgFile),
		util.CpioSymlink("etc/systemd/system/ignition-complete.target.requires/deploy-ock.service", "/usr/lib/systemd/system/deploy-ock.service", 0644): nil,
		util.CpioSymlink("etc/systemd/system/ignition-complete.target.requires/set-config.service", "/usr/lib/systemd/system/set-config.service", 0644): nil,
	}
}

// librariesFromObjects recursively finds the dependencies
// for a set of elf objects.
func librariesFromObjects(objs map[string][]byte, fileMap map[string]*image.FileInfo, out map[string][]byte, ref ctypes.ImageReference, arch string) error {
	found := map[string]bool{}

	for path, contents := range objs {
		_, ok := out[path]
		if ok {
			continue
		}
		out[path] = contents

		log.Debugf("Reading ELF objects %s", path)
		elfObj, err := elf.NewFile(bytes.NewReader(contents))
		if err != nil {
			return err
		}

		dvns, err := elfObj.DynamicVersionNeeds()
		if err != nil {
			return err
		}

		//  All libraries for a default install are in /usr/lib64
		prefix := "usr/lib64"
		for _, dvn := range dvns {
			depPath := filepath.Join(prefix, dvn.Name)
			log.Debugf(" has dependency %s", depPath)
			_, ok := found[depPath]
			if ok {
				continue
			}

			_, ok = fileMap[depPath]
			if !ok {
				return fmt.Errorf("%s has unsatisfiable dependency %s", path, depPath)
			}

			found[depPath] = true
		}

		deps, err := elfObj.ImportedLibraries()
		if err != nil {
			return err
		}
		for _, dep := range deps {
			depPath := filepath.Join(prefix, dep)
			log.Debugf(" has dependency %s", depPath)
			_, ok := found[depPath]
			if ok {
				continue
			}

			_, ok = fileMap[depPath]
			if !ok {
				return fmt.Errorf("%s has unsatisfiable dependency %s", path, depPath)
			}

			found[depPath] = true
		}
	}

	log.Debugf("there are now %d dependencies", len(out))

	paths := []string{}
	for o, _ := range found {
		paths = append(paths, o)
	}

	if len(paths) == 0 {
		return nil
	}

	depContents, err := image.FindInImageFollowLinks(ref, arch, paths, fileMap)
	if err != nil {
		return err
	}

	return librariesFromObjects(depContents, fileMap, out, ref, arch)
}

func addFiles(files map[*util.CpioHeader][]byte, newFiles map[string][]byte) {
	for p, c := range newFiles {
		files[util.CpioFile(p, 0755)] = c
	}
}

func CreateIso(startConfig *otypes.Config, clusterConfig *otypes.ClusterConfig, options CreateOptions) error {
	// Do the work to balance time vs certainty.  Short, uncertain things go
	// first.  Long, uncertain things go next.  Short, mostly certain things
	// are after that.  Finally, long mostly certain tasks are at the end.
	osRegistry, err := image.MakeFullOstreeReference(clusterConfig.OsRegistry, clusterConfig.OsTag)
	if err != nil {
		return err
	}

	ostreeRegistry, err := image.MakeReferenceFromOstree(osRegistry)
	if err != nil {
		return err
	}

	//  Get the syslinux image.
	log.Infof("Getting syslinux container image for architecture: %s", options.Architecture)
	syslinuxRef, err := image.GetOrPull(clusterConfig.Providers.Byo.Iso.UtilityImage, options.Architecture)
	if err != nil {
		return err
	}

	// Get all the files out of the syslinux image
	isoFiles, err := image.FindInImage(syslinuxRef, options.Architecture, []string{
		IsoLinux,
		LdLinux,
		Libcom,
		Libutil,
		Vesamenu,
	})
	if err != nil {
		return err
	}

	log.Debugf("Found all syslinux files")
	for f, c := range isoFiles {
		log.Debugf("  %s contains %s", f, util.HumanReadableSize(uint64(len(c))))
	}

	// Get the OCK OSTree image
	log.Debugf("Checking ostree registry %s", ostreeRegistry)
	log.Infof("Getting ostree image for architecture: %s", options.Architecture)
	ostreeRef, err := image.GetOrPull(ostreeRegistry, options.Architecture)
	if err != nil {
		return err
	}

	log.Debugf("Finished copying ostree image")

	// There is lots to find in the ostree image that can be spread
	// all over the 100 or so layers that it has.  Keep a cache of
	// where files live to avoid having to re-read layers over and
	// over again.
	fileMap, err := image.GetFileLayerMap(ostreeRef, options.Architecture)
	if err != nil {
		return err
	}

	log.Debugf("have %d files", len(fileMap))

	// A kernel and initramfs are required to actually boot the image. Go
	// find them.  The kernel version changes regularly and is not annotated
	// anywhere useful, so it is necessary to go rummaging around.
	//
	// They live in "/usr/lib/modules/<version>/<file>"
	kernelPrefix := "usr/lib/modules/"
	kernelSuffix := "/vmlinuz"
	initramfsSuffix := "/initramfs.img"
	kernelPath := ""
	initramfsPath := ""
	for p, _ := range fileMap {
		if !strings.HasPrefix(p, kernelPrefix) {
			continue
		}

		if strings.HasSuffix(p, kernelSuffix) {
			kernelPath = p
		} else if strings.HasSuffix(p, initramfsSuffix) {
			initramfsPath = p
		}

		// Stop early if both have been found.  There's
		// a lot of files in the pile.
		if initramfsPath != "" && kernelPath != "" {
			break
		}
	}

	if kernelPath == "" {
		return fmt.Errorf("could not find kernel in %s", ostreeRegistry)
	}
	if initramfsPath == "" {
		return fmt.Errorf("could not find initramfs in %s", ostreeRegistry)
	}
	log.Debugf("Kernel is %s", kernelPath)
	log.Debugf("Initramfs is %s", initramfsPath)

	// Get some of the required files out of ostree images.  Some more are
	// needed to satisify the dependencies of dynamically linked
	// executables, but that set of files won't be know until they are
	// discovered by resolving all the linkages in the elf objects.
	ostreeFiles := append([]string{}, kernelPath, initramfsPath)
	ostreeFiles = append(ostreeFiles, executables...)
	ostreeContents, err := image.FindInImageFollowLinks(ostreeRef, options.Architecture, ostreeFiles, fileMap)
	if err != nil {
		return err
	}

	for p, c := range ostreeContents {
		log.Debugf("%s has %d bytes", p, len(c))
	}

	execMap := map[string][]byte{}
	for _, e := range executables {
		execMap[e] = ostreeContents[e]
	}
	depsMap := map[string][]byte{}
	err = librariesFromObjects(execMap, fileMap, depsMap, ostreeRef, options.Architecture)
	if err != nil {
		return err
	}

	newInitramfsContent := defaultInitramfsContent()
	addFiles(newInitramfsContent, depsMap)
	newInitramfsCpio, err := util.MakeCpio(newInitramfsContent, true)
	if err != nil {
		return err
	}

	log.Debugf("New initramfs cpio compressed size: %s", util.HumanReadableSize(uint64(len(newInitramfsCpio))))

	// Stuff the whole thing into an iso
	isoDisk, isoFs, err := disk.MakeISO9660(options.Destination, 8 * 1024 * 1024 * 1024)
	if err != nil {
		return err
	}

	err = isoFs.Finalize(iso9660.FinalizeOptions{
		VolumeIdentifier: "ock",
		ElTorito: &iso9660.ElTorito{
			BootCatalog: "isolinux/boot.cat",
			Entries: []*iso9660.ElToritoEntry{
				{
					Platform:  iso9660.BIOS,
					Emulation: iso9660.NoEmulation,
					BootFile:  IsoLinuxDest,
					BootTable: true,
					LoadSize:  4,
				},
				{
					Platform:  iso9660.EFI,
					Emulation: iso9660.NoEmulation,
					BootFile:  BootX64Dest,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	err = isoDisk.Close()
	if err != nil {
		return err
	}

	log.Infof("Wrote image to %s", options.Destination)

	return nil
}
