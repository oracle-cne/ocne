// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package info

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	qcow2 "github.com/dypflying/go-qcow2lib/qcow2"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/partition/gpt"

	"github.com/oracle-cne/ocne/pkg/config/types"
	"github.com/oracle-cne/ocne/pkg/file"
	contimg "github.com/oracle-cne/ocne/pkg/image"
	"github.com/oracle-cne/ocne/pkg/util"

	log "github.com/sirupsen/logrus"
)

type InfoOptions struct {
	Architecture string
	File         string
}

type qcowFileInfo struct {
	qf *qcowFile
}

type qcowFile struct {
	disk *qcow2.BdrvChild
	size uint64
	name string
	read uint64

}

func (qf *qcowFileInfo) Name() string {
	return qf.qf.name
}

func (qf *qcowFileInfo) Size() int64 {
	if qf.qf.size != 0 {
		return int64(qf.qf.size)
	}
	size, _ := qcow2.Blk_Getlength(qf.qf.disk)
	qf.qf.size = size
	return int64(size)
}

func (qf *qcowFileInfo) Mode() fs.FileMode {
	return 0400
}

func (qf *qcowFileInfo) ModTime() time.Time {
	return time.Now()
}

func (qf *qcowFileInfo) IsDir() bool {
	return false
}


func (qf *qcowFileInfo) Sys() any {
	return nil
}

func (qf *qcowFile) Stat() (fs.FileInfo, error) {
	return &qcowFileInfo{
		qf: qf,
	}, nil
}

func (qf *qcowFile) Sys() (*os.File, error) {
	return nil, nil
}

func (qf *qcowFile) Read(out []byte) (int, error) {
	read, err := qcow2.Blk_Pread(qf.disk, qf.read, out, uint64(len(out)))
	log.Debugf("Reading %d bytes from %d: %d", len(out), qf.read, read)
	if err != nil {
		return -1, nil
	}
	qf.read = qf.read + read
	return int(read), nil
}

func (qf *qcowFile) ReadAt(out []byte, at int64) (int, error) {
	log.Debugf("Reading %d bytes at %d", len(out), at)
	read, err := qcow2.Blk_Pread(qf.disk, uint64(at), out, uint64(len(out)))
	return int(read), err
}

func (qf *qcowFile) Seek(offset int64, whence int) (int64, error) {
	newRead := int64(0)
	stat, _ := qf.Stat()
	switch whence {
		case io.SeekStart:
			newRead = offset
		case io.SeekCurrent:
			newRead = int64(qf.read) + offset
		case io.SeekEnd:
			newRead = int64(stat.Size()) - offset
	}

	if newRead < 0 {
		return newRead, fmt.Errorf("Cannot seek %d bytes from the end of a file %d bytes in size", offset, stat.Size())
	} else if newRead >= stat.Size() {
		return newRead, fmt.Errorf("Cannot seek %d bytes from the start of a file %d bytes in size", offset, stat.Size())
	}

	qf.read = uint64(newRead)

	return int64(qf.read), nil
}

func (qf *qcowFile) WriteAt(in []byte, off int64) (int, error) {
	written, err := qcow2.Blk_Pwrite(qf.disk, uint64(off), in, uint64(len(in)), 0)
	return int(written), err
}

func (qf *qcowFile) Close() error {
	qcow2.Blk_Close(qf.disk)
	return nil
}

func (qf *qcowFile) Writable() (backend.WritableFile, error) {
	return qf, nil
}

func Info(startConfig *types.Config, clusterConfig *types.ClusterConfig, options InfoOptions) error {
	var imgPath string

	if options.File == "" {
		tmpPath, err := file.CreateOcneTempDir("image-info")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpPath)

		tarStream, closer, err := contimg.EnsureBaseQcow2Image(clusterConfig.BootVolumeContainerImage, options.Architecture)
		if err != nil {
			return err
		}
		defer closer()

		imgPath := filepath.Join(tmpPath, "boot.qcow2")
		err = writeFile(tarStream, imgPath)
		if err != nil {
			return err
		}
	} else {
		imgPath = options.File
	}

	qcowImg, err := qcow2.Blk_Open(imgPath,
		map[string]any{
			qcow2.OPT_FMT: qcow2.TYPE_QCOW2_NAME,
		},
		qcow2.BDRV_O_RDWR,
	)
	if err != nil {
		return err
	}

	diskImgFile := &qcowFile{
		disk: qcowImg,
	}

	log.Infof("Info: %s", qcow2.Blk_Info(qcowImg, true, true))

	// TODO: remove
	data := make([]uint8, 512)
	_, err = qcow2.Blk_Pread(qcowImg, 0, data, 512)
	if err != nil {
		return err
	}

	log.Infof("Dump:\n%s", hex.Dump(data))

	disk, err := diskfs.OpenBackend(diskImgFile)
	if err != nil {
		return err
	}

	partTable, err := disk.GetPartitionTable()
	if err != nil {
		return err
	}

	stat, _ := diskImgFile.Stat()
	log.Infof("Image: %s", clusterConfig.BootVolumeContainerImage)
	log.Infof("Size: %s", util.HumanReadableSize(uint64(stat.Size())))
	log.Infof("Partition Table: %s", partTable.UUID())
	for i, pt := range partTable.GetPartitions() {
		gptPart, ok := pt.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("Parition %d is not a GPT partition", i)
		}
		log.Infof("\t%d", i)
		log.Infof("\t\tName: %s", gptPart.Name)
		log.Infof("\t\tGUID: %s", gptPart.GUID)
		log.Infof("\t\tType: %s", gptPart.Type)
		log.Infof("\t\tSize: %s", util.HumanReadableSize(gptPart.Size))
		log.Infof("\t\tExtents: %d to %d", gptPart.Start, gptPart.End)
	}

	return nil

}

func writeFile(reader io.Reader, filePath string) error {
	w, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		return err
	}

	return nil
}


