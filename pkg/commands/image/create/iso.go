// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
	otypes "github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/util"
	"github.com/oracle-cne/ocne/pkg/util/disk"
	"github.com/oracle-cne/ocne/pkg/util/linux"
	"github.com/oracle-cne/ocne/pkg/file"
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

func CreateIso(startConfig *otypes.Config, clusterConfig *otypes.ClusterConfig, options CreateOptions) error {
	// Do the work to balance time vs certainty.  Short, uncertain things go
	// first.  Long, uncertain things go next.  Short, mostly certain things
	// are after than.  Finally, long mostly certain tasks are at the end.

	tmpPath, err := file.CreateOcneTempDir(tempDir)
	if err != nil {
		return err
	}
	//defer os.RemoveAll(tmpPath)

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

	// Get the OCK image
	// Get the tarstream of the boot qcow2 image
	log.Infof("Getting local boot image for architecture: %s", options.Architecture)
	tarStream, closer, err := image.EnsureBaseQcow2Image(clusterConfig.BootVolumeContainerImage, options.Architecture)
	if err != nil {
		return err
	}
	defer closer()

	// Write the local image. e.g. ~/.ocne/tmp/create-images.xyz/boot.oci
	localImagePath := filepath.Join(tmpPath, localVMImage+".oci")
	err = writeFile(tarStream, localImagePath)
	if err != nil {
		return err
	}

	qcowImg, err := disk.OpenQcow2(localImagePath)
	if err != nil {
		return err
	}

	// Get filesystems
	var efiFs filesystem.FileSystem
	var rootFs filesystem.FileSystem
	var bootFs filesystem.FileSystem

	var rootPart *gpt.Partition

	partTable, err := qcowImg.GetPartitionTable()
	if err != nil {
		return err
	}

	for i, pt := range partTable.GetPartitions() {
		gptPart, ok := pt.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("Partition %d is not a GPT partition", i)
		}

		thefs, err := qcowImg.GetFilesystem(i)
		if err != nil {
			return err
		}

		if strings.Contains(gptPart.Name, "EFI") {
			efiFs = thefs
		} else if gptPart.Name == "root" {
			rootFs = thefs
			rootPart = gptPart
		} else if gptPart.Name == "boot" {
			bootFs = thefs
		}
	}

	if efiFs == nil {
		return fmt.Errorf("Could not find EFI filesystem")
	} else if rootFs == nil {
		return fmt.Errorf("Could not find root filesystem")
	} else if bootFs == nil {
		return fmt.Errorf("Could not find boot filesystem")
	}

	// Get the EFI files
	efiFiles, err := disk.FindFilesInFilesystem(efiFs, []string{
		BootX64,
		GrubX64,
		MMX64,
	})
	if err != nil {
	}

	log.Debugf("Found all EFI files")
	for f, c := range efiFiles {
		log.Debugf("  %s has %s", f, util.HumanReadableSize(uint64(len(c))))
	}


	// Get the grub configuration.  Once that is found, sniff around in it
	// to find a reasonable kernel and initrd to use.
	grubConf, err := disk.GetFileInFilesystem(bootFs, GrubConfig)
	if err != nil {
		return err
	}

	menuEntry, err := linux.ParseMenuEntry(string(grubConf))
	if err != nil {
		return err
	}

	// Get the ostree that is the source of the content going
	// into the squashfs
	ostreeArgs := linux.GetKernelArg(menuEntry.KernelArgs, "ostree")
	if len(ostreeArgs) == 0 {
		return fmt.Errorf("grub menu entry did not specify an ostree deployment")
	}
	ostreePath, err := disk.RealPath(rootFs, ostreeArgs[0])
	if err != nil {
		return err
	}

	kernelPath := menuEntry.Kernel
	initrdPath := menuEntry.Initrd
	log.Debugf("ostree: %s", ostreePath)
	log.Debugf("Kernel: %s", kernelPath)
	log.Debugf("Initrd: %s", initrdPath)

	bootFiles, err := disk.FindFilesInFilesystem(bootFs, []string{
		kernelPath,
		initrdPath,
	})
	if err != nil {
		return err
	}

	bootTree := &disk.File{
		IsDir: true,
	}
	bootTree.AddFile(ImagesKernelDest, bootFiles[kernelPath], DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)
	bootTree.AddFile(ImagesInitrdDest, bootFiles[initrdPath], DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)
	bootTree.AddFile(KernelDest, bootFiles[kernelPath], DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)
	bootTree.AddFile(InitrdDest, bootFiles[initrdPath], DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)

	for f, c := range efiFiles {
		bootTree.AddFile(fileMapping[f], c, DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)
	}
	for f, c := range isoFiles {
		bootTree.AddFile(fileMapping[f], c, DefaultFileUid, DefaultFileGid, DefaultFileMode, DefaultDirUid, DefaultDirGid, DefaultDirMode)
	}
	bootTree.AddISO9660TransTbl()

	// Embed an ignition file into the initrd.

	// Generate a reasonable grub configuration.

	// Dump the root filesystem into a squashfs
	tmpDir, err := os.MkdirTemp("", "ocneIso")
	if err != nil {
		return err
	}

	rootSquashDiskPath := filepath.Join(tmpDir, "rootSquash")

	// If this is an xfs partition, get the amount of data being
	// copied to use for a progress bar.  At present, this is
	// the only filesystem implementation that can do this.  Thankfully
	// it is the only realistic option for the root filesystem.
	rootXfsFs, ok := rootFs.(*disk.XfsFilesystem)
	var rootUsed uint64
	if ok {
		rootPartSize := rootPart.Size
		rootUsed = rootPartSize - rootXfsFs.Free()
		log.Debugf("Root filesystem size: %s", util.HumanReadableSize(rootUsed))
	}

	//defer os.RemoveAll(tmpDir)
	rootSquashDisk, rootSquashFs, err := disk.MakeSquashfs(rootSquashDiskPath, 8 * 1024 * 1024 * 1024)
	if err != nil {
		return err
	}

	// Copy in the checked out ostree.  This is a read-only filesystem, so
	// retaining the actual ostree commits is not relevant.  Use the total
	// used space as an estimate for how data needs to be copied.  ostree
	// checkouts are done with hard links, so there is roughly twice as
	// much apparent disk use as actual disk use.  Given that only half
	// of that apparent data is required, it's a reasonable estimate.
	err = disk.CopyFilesystem(rootFs, rootSquashFs, ostreePath, rootUsed)
	if err != nil {
		// TODO remove this
		rootSquashFs.Finalize(squashfs.FinalizeOptions{})
		return err
	}

	err = rootSquashFs.Finalize(squashfs.FinalizeOptions{})
//	err = rootSquashFs.Finalize(squashfs.FinalizeOptions{
//		Compression: &squashfs.CompressorGzip{
//			CompressionLevel: 9,
//			WindowSize: 65535,
//			Strategies: map[squashfs.GzipStrategy]bool{
//				squashfs.GzipDefault: true,
//			},
//		},
//		Xattrs: true,
//	})
	if err != nil {
		return err
	}

	err = rootSquashDisk.Close()
	if err != nil {
		return err
	}

	log.Infof("Wrote squashfs to %s", rootSquashDiskPath)

	// Stuff the whole thing into an iso
	isoDisk, isoFs, err := disk.MakeISO9660(options.Destination, 8 * 1024 * 1024 * 1024)
	if err != nil {
		return err
	}

	err = disk.CopyFiles(isoFs, bootTree.Entries, "/")
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
