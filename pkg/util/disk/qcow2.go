// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package disk

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	qcow2 "github.com/dypflying/go-qcow2lib/qcow2"
	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	log "github.com/sirupsen/logrus"
)

type qcow2FileInfo struct {
	qd *Qcow2Disk
}

type Qcow2Disk struct {
	disk *qcow2.BdrvChild
	diskfs *disk.Disk
	size uint64
	name string
	read uint64
}

func OpenQcow2(path string) (*Qcow2Disk, error) {
	img, err := qcow2.Blk_Open(path,
		map[string]any{
			qcow2.OPT_FMT: qcow2.TYPE_QCOW2_NAME,
		},
		qcow2.BDRV_O_RDWR,
	)
	if err != nil {
		return nil, err
	}

	ret :=  &Qcow2Disk{
		disk: img,
	}

	diskfsDisk, err := diskfs.OpenBackend(ret)
	if err != nil {
		return nil, err
	}

	ret.diskfs = diskfsDisk

	return ret, nil
}

func (qf *qcow2FileInfo) Name() string {
	return qf.qd.name
}

func (qf *qcow2FileInfo) Size() int64 {
	if qf.qd.size != 0 {
		return int64(qf.qd.size)
	}
	size, _ := qcow2.Blk_Getlength(qf.qd.disk)
	qf.qd.size = size
	return int64(size)
}

func (qf *qcow2FileInfo) Mode() fs.FileMode {
	return 0400
}

func (qf *qcow2FileInfo) ModTime() time.Time {
	return time.Now()
}

func (qf *qcow2FileInfo) IsDir() bool {
	return false
}


func (qf *qcow2FileInfo) Sys() any {
	return nil
}

func (qd *Qcow2Disk) Stat() (fs.FileInfo, error) {
	return &qcow2FileInfo{
		qd: qd,
	}, nil
}

func (qd *Qcow2Disk) Sys() (*os.File, error) {
	return nil, nil
}

func (qd *Qcow2Disk) Read(out []byte) (int, error) {
	log.Tracef("Reading %d bytes from %d", len(out), qd.read)
	read, err := qcow2.Blk_Pread(qd.disk, qd.read, out, uint64(len(out)))
	log.Tracef("  read %d", read)
	if err != nil {
		return -1, nil
	}
	qd.read = qd.read + read
	return int(read), nil
}

func (qd *Qcow2Disk) ReadAt(out []byte, at int64) (int, error) {
	log.Tracef("Reading %d bytes at %d", len(out), at)
	read, err := qcow2.Blk_Pread(qd.disk, uint64(at), out, uint64(len(out)))
	log.Tracef("  read %d", read)
	return int(read), err
}

func (qd *Qcow2Disk) Seek(offset int64, whence int) (int64, error) {
	newRead := int64(0)
	stat, _ := qd.Stat()
	switch whence {
		case io.SeekStart:
			newRead = offset
		case io.SeekCurrent:
			newRead = int64(qd.read) + offset
		case io.SeekEnd:
			newRead = int64(stat.Size()) - offset
	}

	if newRead < 0 {
		return newRead, fmt.Errorf("Cannot seek %d bytes from the end of a file %d bytes in size", offset, stat.Size())
	} else if newRead >= stat.Size() {
		return newRead, fmt.Errorf("Cannot seek %d bytes from the start of a file %d bytes in size", offset, stat.Size())
	}

	qd.read = uint64(newRead)

	return int64(qd.read), nil
}

func (qd *Qcow2Disk) WriteAt(in []byte, off int64) (int, error) {
	written, err := qcow2.Blk_Pwrite(qd.disk, uint64(off), in, uint64(len(in)), 0)
	return int(written), err
}

func (qd *Qcow2Disk) Close() error {
	qcow2.Blk_Close(qd.disk)
	return nil
}

func (qd *Qcow2Disk) Writable() (backend.WritableFile, error) {
	return qd, nil
}

func (qd *Qcow2Disk) Info() string {
	return qcow2.Blk_Info(qd.disk, true, true)
}

func (qd *Qcow2Disk) GetDisk() *disk.Disk {
	return qd.diskfs
}

func (qd *Qcow2Disk) GetPartitionTable() (partition.Table, error) {
	return qd.diskfs.GetPartitionTable()
}

func (qd *Qcow2Disk) GetFilesystem(part int) (filesystem.FileSystem, error) {
	// Everything in go-diskfs is zero indexed except this for some reason?
	fileSystem, err := qd.diskfs.GetFilesystem(part+1)
	if err != nil {
		log.Debugf("Could not open disk as go-diskfs supporte filesystem: %v", err)
	} else {
		return fileSystem, nil
	}

	table, err := qd.diskfs.GetPartitionTable()
	if err != nil {
		return nil, err
	}

	pts := table.GetPartitions()

	if part >= len(pts) {
		return nil, fmt.Errorf("asked for partition %d but only %d exist", part, len(pts))
	}

	pt := pts[part]

	gptPart, ok := pt.(*gpt.Partition)
	if !ok {
		return nil, fmt.Errorf("partition %d is not a GPT partition", part)
	}

	pr := &partitionReader{
		qd: qd,
		start: gptPart.Start * 512,
		end: gptPart.End * 512,
	}

	fileSystem, err = GetXfsFilesystem(pr, int64(gptPart.Size))
	if err != nil {
		return nil, err
	}

	return fileSystem, nil
}

type partitionReader struct {
	qd *Qcow2Disk
	start uint64
	end uint64
}

func (pr *partitionReader) ReadAt(out []byte, offset int64) (int, error) {
	return pr.qd.ReadAt(out, int64(pr.start) + offset)
}

