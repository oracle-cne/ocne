package qcow2

/*
Copyright (c) 2023 Yunpeng Deng
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

import (
	"bytes"
	"compress/flate"
	"container/list"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

func newQcow2Driver() *BlockDriver {
	return &BlockDriver{
		FormatName:           "qcow2",
		IsFormat:             true,
		SupportBacking:       true,
		bdrv_close:           qcow2_close,
		bdrv_create:          qcow2_create,
		bdrv_open:            qcow2_open,
		bdrv_flush_to_os:     qcow2_flush_to_os,
		bdrv_pwritev_part:    qcow2_pwritev_part,
		bdrv_preadv_part:     qcow2_preadv_part,
		bdrv_block_status:    qcow2_block_status,
		bdrv_pwrite_zeroes:   qcow2_pwrite_zeroes,
		bdrv_copy_range_from: qcow2_copy_range_from,
		bdrv_copy_range_to:   qcow2_copy_range_to,
		bdrv_pdiscard:        qcow2_pdiscard,
	}
}

func qcow2_close(bs *BlockDriverState) {
	if bs == nil {
		return
	}
	s := bs.opaque.(*BDRVQcow2State)
	qcow2_cache_flush(bs, s.L2TableCache)
	qcow2_cache_flush(bs, s.RefcountBlockCache)
	s.L1Table = nil
	qcow2_cache_destroy(s.L2TableCache)
	qcow2_cache_destroy(s.RefcountBlockCache)
	if bs.current != nil {
		bdrv_close(bs.current.bs)
	}
	if bs.backing != nil {
		bdrv_close(bs.backing.bs)
	}
	if has_data_file(bs) {
		bdrv_close(s.DataFile.bs)
		s.DataFile = nil
	}
}

func qcow2_create(filename string, options map[string]any) error {

	var err error
	var size uint64
	var backingFile string
	var child *BdrvChild
	var enableSc bool
	var dataFile string
	var backingFileFmt string

	//check file name
	if filename == "" {
		return Err_IncompleteParameters
	}

	//check file size
	if val, ok := options[OPT_SIZE]; !ok {
		return Err_IncompleteParameters
	} else {
		size = interface2uint64(val)
	}

	//check backing file
	if val, ok := options[OPT_BACKING]; ok {
		backingFile = val.(string)
	}

	//check enable subcluster
	if val, ok := options[OPT_SUBCLUSTER]; ok {
		enableSc = val.(bool)
	}

	//data file
	if val, ok := options[OPT_DATAFILE]; ok {
		dataFile = val.(string)
	}

	//backing file format
	if val, ok := options[OPT_BACKING_FILE_FMT]; ok {
		backingFileFmt = val.(string)
	}

	//now open the child
	if child, err = bdrv_open_child(filename, "raw", options, BDRV_O_CREATE|BDRV_O_RDWR); err != nil {
		return err
	} else {
		bdrv_set_perm(child, PERM_ALL)
	}

	//round up the size to align with the sector size(512)
	size = round_up(size, DEFAULT_SECTOR_SIZE)

	//calculate the l1size based on the cluster size
	size2 := round_up(size, DEFAULT_CLUSTER_SIZE)
	l1Size := round_up(size2, 1<<(DEFAULT_CLUSTER_BITS+DEFAULT_CLUSTER_BITS-3)) >> (DEFAULT_CLUSTER_BITS + DEFAULT_CLUSTER_BITS - 3)

	//initiate default header
	header := &QCowHeader{
		Magic:                 binary.BigEndian.Uint32(QCOW_MAGIC),
		Version:               QCOW2_VERSION3,
		BackingFileOffset:     uint64(0),
		BackingFileSize:       uint32(0),
		ClusterBits:           uint32(DEFAULT_CLUSTER_BITS),
		Size:                  uint64(size),
		CryptMethod:           uint32(QCOW2_CRYPT_METHOD),
		L1Size:                uint32(l1Size),
		L1TableOffset:         uint64(L1_TABLE_OFFSET),
		RefcountTableOffset:   uint64(REFCOUNT_TABLE_OFFSET),
		RefcountTableClusters: uint32(DEFAULT_REFCOUNT_TABLE_CLUSTERS),
		NbSnapshots:           uint32(0),
		SnapshotsOffset:       uint64(0),
		IncompatibleFeatures:  uint64(0),
		CompatibleFeatures:    uint64(0),
		AutoclearFeatures:     uint64(0),
		RefcountOrder:         uint32(QCOW2_REFCOUNT_ORDER), // NOTE: qemu now supported only refcount_order = 4
		HeaderLength:          uint32(unsafe.Sizeof(QCowHeader{})),
	}
	//set enable subcluster
	if enableSc {
		header.IncompatibleFeatures |= QCOW2_INCOMPAT_EXTL2
		header.L1Size = header.L1Size * 2
	}
	if dataFile != "" {
		header.IncompatibleFeatures |= QCOW2_INCOMPAT_DATA_FILE
		header.AutoclearFeatures |= QCOW2_AUTOCLEAR_DATA_FILE_RAW
	}
	//set the backing file
	if backingFile != "" {
		header.BackingFileOffset = BACKING_FILE_OFFSET
		if _, err = os.Stat(backingFile); err != nil {
			return err
		}
		if backingFile, err = filepath.Abs(backingFile); err != nil {
			return err
		}
		header.BackingFileSize = uint32(len(backingFile))
	}

	//initiate the BlockDriverState struct
	qcow2State := initiate_qcow2_state(header, enableSc)
	bs := &BlockDriverState{
		filename: filename,
		options:  make(map[string]any),
		opaque:   qcow2State,
		//SupportedWriteFlags: BDRV_REQ_WRITE_UNCHANGED | BDRV_REQ_FUA,
		SupportedWriteFlags: 0,
		RequestAlignment:    DEFAULT_ALIGNMENT,
		PdiscardAlignment:   DEFAULT_CLUSTER_SIZE,
		MaxTransfer:         DEFAULT_MAX_TRANSFER,
	}

	bdrv_link_child(bs, child, filename)
	//write the header to buffer in big-endian manner
	if _, err := Blk_Pwrite_Object(bs.current, 0, header, uint64(unsafe.Sizeof(*header))); err != nil {
		return err
	}
	//write the data file if any
	// store the datafile header ext length to use with backing file format header ext
	dataFileHeaderExtLength := uint64(0)
	if dataFile != "" {
		dataFileHeaderExtLength, err = header_ext_add_external_data_file(bs, uint64(unsafe.Sizeof(QCowHeader{})), dataFile)
		if err != nil {
			return err
		}
		var dataChild *BdrvChild
		//now open the child
		if dataChild, err = bdrv_open_child(dataFile, "raw", options, BDRV_O_CREATE|BDRV_O_RDWR); err != nil {
			return err
		} else {
			bdrv_set_perm(dataChild, PERM_ALL)
			qcow2State.DataFile = dataChild
		}

	} else {
		qcow2State.DataFile = child
	}
	//write the backing file
	if backingFile != "" {
		if _, err := Blk_Pwrite_Object(bs.current, BACKING_FILE_OFFSET,
			([]byte)(backingFile), uint64(len(backingFile))); err != nil {
			return err
		}
		if backingFileFmt != "" {
			// write backing file extension
			header_ext_add_backing_file_format(bs, uint64(unsafe.Sizeof(QCowHeader{}))+dataFileHeaderExtLength, backingFileFmt)
		}
	}

	//temporary initiate cache for writing the meta information
	qcow2State.L2TableCache = qcow2_cache_create(bs, 1, qcow2State.ClusterSize)
	qcow2State.RefcountBlockCache = qcow2_cache_create(bs, 1, qcow2State.ClusterSize)

	// Write a refcount table with one refcount block
	qcow2State.RefcountTable = make([]uint64, 2*REFCOUNT_TABLE_SIZE)
	qcow2State.RefcountTable[0] = 2 * DEFAULT_CLUSTER_SIZE
	if _, err := Blk_Pwrite_Object(bs.current, REFCOUNT_TABLE_OFFSET,
		qcow2State.RefcountTable, uint64(2*REFCOUNT_TABLE_SIZE*SIZE_UINT64)); err != nil {
		return err
	}
	bdrv_flush(bs)

	//write l1 table
	qcow2State.L1Table = make([]uint64, header.L1Size)
	if _, err := Blk_Pwrite_Object(bs.current, L1_TABLE_OFFSET, qcow2State.L1Table,
		l1Size*SIZE_UINT64); err != nil {
		return err
	}
	//sync to disk
	bdrv_flush(bs)

	//alloc 4 clusters for the header block,
	//then mark them as occupied
	if _, err = qcow2_alloc_clusters(bs, 4*DEFAULT_CLUSTER_SIZE); err != nil {
		return err
	}

	//close the file
	qcow2_close(bs)
	return err
}

// func Open(bs *BlockDriverState, options *QDict, flag int) error {
func qcow2_open(filename string, opts map[string]any, flags int) (*BlockDriverState, error) {

	var err error
	var child *BdrvChild
	var backing *BdrvChild
	var enableSc bool
	var l2CacheSize uint64
	var l2CacehNum uint32
	var dataFile string

	//check file name
	if filename == "" {
		return nil, Err_IncompleteParameters
	}

	if val, ok := opts[OPT_L2CACHESIZE]; ok {
		l2CacheSize = val.(uint64)
	}

	//now open the child
	if child, err = bdrv_open_child(filename, "raw", opts, flags); err != nil {
		return nil, err
	} else {
		bdrv_set_perm(child, PERM_ALL)
	}

	//now read the header
	var header QCowHeader
	if _, err = Blk_Pread_Object(child, 0, &header, uint64(unsafe.Sizeof(header))); err != nil {
		return nil, fmt.Errorf("qcow2 file %s read fail, err: %v", filename, err)
	}
	//check header
	if err = check_header(&header); err != nil {
		return nil, err
	}
	child.header = &header

	//read the backing file
	var backingFile string
	if header.BackingFileOffset > 0 && header.BackingFileSize > 0 {
		backingBytes := make([]byte, header.BackingFileSize)
		if _, err = Blk_Pread_Object(child, header.BackingFileOffset,
			backingBytes, uint64(header.BackingFileSize)); err != nil {
			return nil, fmt.Errorf("can not read backing file, err: %v", err)
		}
		backingFile = string(backingBytes)
		if backing, err = bdrv_open_child(backingFile, "qcow2", opts, flags); err != nil {
			return nil, err
		} else {
			bdrv_set_perm(backing, PERM_READABLE)
		}
	}

	if header.IncompatibleFeatures&QCOW2_INCOMPAT_EXTL2 > 0 {
		enableSc = true
	}

	qcow2State := initiate_qcow2_state(&header, enableSc)
	//opaque.DataFile = child
	//initiate the BlockDriverState struct
	bs := &BlockDriverState{
		filename:            filename,
		backingFile:         backingFile,
		opaque:              qcow2State, //initiate the BDRVQcow2State struct
		options:             make(map[string]any),
		SupportedWriteFlags: 0,
		RequestAlignment:    DEFAULT_ALIGNMENT,
		PdiscardAlignment:   DEFAULT_CLUSTER_SIZE,
		MaxTransfer:         DEFAULT_MAX_TRANSFER,
		TotalSectors:        header.Size / BDRV_SECTOR_SIZE,
		InheritsFrom:        nil,
		OpenFlags:           flags,
	}
	//update child
	bdrv_link_child(bs, child, filename)
	//link backing
	if backing != nil {
		bdrv_link_backing(bs, backing, backingFile)
	}

	if header.IncompatibleFeatures&QCOW2_INCOMPAT_DATA_FILE > 0 {
		//read the header ext
		if dataFile, err = header_ext_read_external_data_file(bs, uint64(unsafe.Sizeof(header))); err != nil {
			return nil, err
		}
		var dataChild *BdrvChild
		//now open the child
		if dataChild, err = bdrv_open_child(dataFile, "raw", opts, flags); err != nil {
			return nil, err
		} else {
			bdrv_set_perm(dataChild, PERM_ALL)
			dataChild.name = dataFile
			qcow2State.DataFile = dataChild
		}
	} else {
		qcow2State.DataFile = child
	}

	//load refcount table
	if err = qcow2_refcount_init(bs); err != nil {
		return nil, fmt.Errorf("could not initialize refcount table, err: %v", err)
	}
	//load l1 table
	if qcow2State.L1Size > 0 {
		qcow2State.L1Table = make([]uint64, qcow2State.L1Size)
		if _, err = Blk_Pread_Object(bs.current, qcow2State.L1TableOffset, qcow2State.L1Table,
			uint64(qcow2State.L1Size)*SIZE_UINT64); err != nil {
			return nil, fmt.Errorf("could not read L1 table")
		}
	}

	//initiate the caches
	if l2CacheSize > 0 {
		l2CacheSize = round_up(l2CacheSize, DEFAULT_CLUSTER_SIZE)
		l2CacehNum = uint32(l2CacheSize / DEFAULT_CLUSTER_SIZE)
	} else {
		l2CacehNum = qcow2State.L1Size
	}
	qcow2State.L2TableCache = qcow2_cache_create(bs, l2CacehNum, qcow2State.ClusterSize)
	//since the refcount block cache must be less than 50% of l2 table cache,
	//so 50% of l2 cache is good enough for refcount block cache
	refcountCacheNum := max(l2CacehNum/2, 1)
	qcow2State.RefcountBlockCache = qcow2_cache_create(bs, refcountCacheNum, qcow2State.ClusterSize)

	return bs, nil
}

func initiate_qcow2_state(header *QCowHeader, enableSC bool) *BDRVQcow2State {

	s := &BDRVQcow2State{
		ClusterBits:          header.ClusterBits,
		ClusterSize:          1 << header.ClusterBits,
		L1Size:               header.L1Size,
		RefcountBlockBits:    header.ClusterBits - (header.RefcountOrder - 3),
		RefcountBlockSize:    1 << (header.ClusterBits - (header.RefcountOrder - 3)),
		RefcountTableOffset:  header.RefcountTableOffset,
		RefcountTableSize:    header.RefcountTableClusters << (header.ClusterBits - 3),
		ClusterOffsetMask:    1<<(70-header.ClusterBits) - 1, //only 54 bits
		L1TableOffset:        header.L1TableOffset,
		QcowVersion:          int(header.Version),
		ClusterAllocs:        list.New(),
		Discards:             list.New(),
		get_refcount:         get_refcount,
		set_refcount:         set_refcount,
		AioTaskRoutine:       qcow2_aio_routine,
		Lock:                 &sync.Mutex{},
		AutoclearFeatures:    header.AutoclearFeatures,
		IncompatibleFeatures: header.IncompatibleFeatures,
		CompatibleFeatures:   header.CompatibleFeatures,
	}

	s.CsizeShift = (62 - (header.ClusterBits - 8))
	s.CsizeMask = (uint32(1) << s.CsizeShift) - 1
	//subcluster related
	if enableSC {
		s.IncompatibleFeatures |= QCOW2_INCOMPAT_EXTL2
		s.SubclustersPerCluster = QCOW_EXTL2_SUBCLUSTERS_PER_CLUSTER
		s.SubclusterBits = uint64(header.ClusterBits - 5)
		s.SubclusterSize = uint64(s.ClusterSize) / QCOW_EXTL2_SUBCLUSTERS_PER_CLUSTER
		s.L2Bits = header.ClusterBits - 4
		s.L2Size = 1 << s.L2Bits
		s.L2SliceSize = 1 << (header.ClusterBits - 4)
	} else {
		s.IncompatibleFeatures = 0
		s.SubclustersPerCluster = 1
		s.SubclusterSize = 1 << header.ClusterBits
		s.SubclusterBits = uint64(header.ClusterBits)
		s.L2Bits = header.ClusterBits - 3
		s.L2Size = 1 << (header.ClusterBits - 3)
		s.L2SliceSize = 1 << (header.ClusterBits - 3)
	}

	s.DiscardPassthrough[QCOW2_DISCARD_NEVER] = false
	s.DiscardPassthrough[QCOW2_DISCARD_ALWAYS] = true
	s.DiscardPassthrough[QCOW2_DISCARD_REQUEST] = true
	s.DiscardPassthrough[QCOW2_DISCARD_SNAPSHOT] = false
	s.DiscardPassthrough[QCOW2_DISCARD_OTHER] = false
	return s
}

func check_header(header *QCowHeader) error {
	//check header magic
	if header.Magic != binary.BigEndian.Uint32(QCOW_MAGIC) {
		return fmt.Errorf("not qcow2 format")
	}
	//check header version
	if header.Version != QCOW2_VERSION2 && header.Version != QCOW2_VERSION3 {
		return fmt.Errorf("not support header version: %d", header.Version)
	}
	//check cluster bits
	if header.ClusterBits != DEFAULT_CLUSTER_BITS {
		return fmt.Errorf("not support cluster size of %d, only cluster size of 64 kib is supported", 1<<header.ClusterBits)
	}
	//check refcountorder
	if header.RefcountOrder != QCOW2_REFCOUNT_ORDER {
		return fmt.Errorf("not support refcount order of %d, only refcount order of 4 is supported", header.RefcountOrder)
	}
	//check crypt method
	if header.CryptMethod != QCOW2_CRYPT_METHOD {
		return fmt.Errorf("not support cryption")
	}
	//check header length
	if header.HeaderLength > uint32(unsafe.Sizeof(QCowHeader{})) {
		return fmt.Errorf("not support extended qcow2 header")
	}
	return nil
}

func qcow2_preadv_part(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	s := bs.opaque.(*BDRVQcow2State)
	var err error
	var curBytes uint32 /* number of bytes in current iteration */
	var hostOffset uint64
	var sctype QCow2SubclusterType
	var isAio bool

	for bytes != 0 {

		curBytes = uint32(bytes)
		s.Qlock()
		err = qcow2_get_host_offset(bs, offset, &curBytes,
			&hostOffset, &sctype)
		s.Qunlock()
		if err != nil {
			goto out
		}
		if sctype == QCOW2_SUBCLUSTER_ZERO_PLAIN ||
			sctype == QCOW2_SUBCLUSTER_ZERO_ALLOC ||
			(sctype == QCOW2_SUBCLUSTER_UNALLOCATED_PLAIN && bs.backing == nil) ||
			(sctype == QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC && bs.backing == nil) {
			qemu_iovec_memset(qiov, qiovOffset, 0, uint64(curBytes))
		} else {
			if !isAio && curBytes != uint32(bytes) {
				isAio = true
			}
			if err = qcow2_add_task(bs, isAio, qcow2_preadv_task_entry, sctype,
				hostOffset, offset, uint64(curBytes),
				qiov, qiovOffset, nil); err != nil {
				goto out
			}
			/*if err = qcow2_preadv_task(bs, sctype, hostOffset, offset, bytes, qiov, qiovOffset); err != nil {
				goto out
			}*/
		}
		bytes -= uint64(curBytes)
		offset += uint64(curBytes)
		qiovOffset += uint64(curBytes)
	}
out:
	return err
}

func qcow2_pwritev_part(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	s := bs.opaque.(*BDRVQcow2State)
	var err error

	var curBytes uint64 /* number of sectors in current iteration */
	var hostOffset uint64
	var l2meta *QCowL2Meta
	var isAio bool

	for bytes != 0 {

		l2meta = nil
		curBytes = bytes

		s.Qlock()
		/*
		* retrieve the hostOffset which is the position in the qcow2 file for writing the buffer
		* l2meta contains all the meta information regarding the write ops.
		 */
		if err = qcow2_alloc_host_offset(bs, offset, &curBytes, &hostOffset, &l2meta); err != nil {
			goto out_locked
		}
		s.Qunlock()

		if !isAio && curBytes != bytes {
			isAio = true
		}
		err = qcow2_add_task(bs, isAio, qcow2_pwritev_task_entry, 0, hostOffset, offset, curBytes, qiov, qiovOffset, l2meta)
		l2meta = nil /* l2meta is consumed by qcow2_co_pwritev_task() */
		if err != nil {
			goto fail_nometa
		}

		bytes -= uint64(curBytes)
		offset += uint64(curBytes)
		qiovOffset += uint64(curBytes)
	}
	err = nil
	s.Qlock()

out_locked:
	//update the l2meta information, and flush it to l2 table as well as l2 cache if success.
	qcow2_handle_l2meta(bs, &l2meta, false)
	s.Qunlock()
fail_nometa:
	return err
}

func qcow2_pwrite_zeroes(bs *BlockDriverState, offset uint64, bytes uint64, flags BdrvRequestFlags) error {

	var err error
	s := bs.opaque.(*BDRVQcow2State)

	head := offset_into_subcluster(s, offset)
	tail := round_up(offset+bytes, s.SubclusterSize) - (offset + bytes)
	if offset+bytes == bs.TotalSectors*BDRV_SECTOR_SIZE {
		tail = 0
	}
	if head > 0 || tail > 0 {
		var off uint64
		var nr uint32
		var sctype QCow2SubclusterType

		Assert((head + bytes + tail) <= s.SubclusterSize)

		/* check whether remainder of cluster already reads as zero */
		if !(is_zero(bs, offset-head, head) &&
			is_zero(bs, offset+bytes, tail)) {
			return ERR_ENOTSUP
		}

		s.Qlock()
		/* We can have new write after previous check */
		offset -= head
		bytes = s.SubclusterSize
		nr = uint32(s.SubclusterSize)
		err = qcow2_get_host_offset(bs, offset, &nr, &off, &sctype)
		if err != nil ||
			(sctype != QCOW2_SUBCLUSTER_UNALLOCATED_PLAIN &&
				sctype != QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC &&
				sctype != QCOW2_SUBCLUSTER_ZERO_PLAIN &&
				sctype != QCOW2_SUBCLUSTER_ZERO_ALLOC) {
			s.Qunlock()
			return err
		}
	} else {
		s.Qlock()
	}

	/* Whatever is left can use real zero subclusters */
	err = qcow2_subcluster_zeroize(bs, offset, bytes, int(flags))
	s.Qunlock()

	return err
}

func qcow2_handle_l2meta(bs *BlockDriverState, pl2meta **QCowL2Meta, linkL2 bool) error {
	var l2meta *QCowL2Meta
	l2meta = *pl2meta
	var err error

	s := bs.opaque.(*BDRVQcow2State)
	for l2meta != nil {
		if linkL2 {
			if err = qcow2_alloc_cluster_link_l2(bs, l2meta); err != nil {
				goto out
			}
		} else {
			qcow2_alloc_cluster_abort(bs, l2meta)
		}

		s.ClusterAllocs.Remove(l2meta.NextInFlight)
		l2meta = l2meta.Next
	}

out:
	*pl2meta = l2meta
	return err
}

func qcow2_parse_compressed_l2_entry(bs *BlockDriverState, l2Entry uint64, cOffset *uint64, cSize *int) {
	s := bs.opaque.(*BDRVQcow2State)

	*cOffset = l2Entry & s.ClusterOffsetMask

	nbCsectors := uint64((uint32(l2Entry >> s.CsizeShift) & s.CsizeMask) + 1)
	*cSize = int((nbCsectors * QCOW2_COMPRESSED_SECTOR_SIZE) - (*cOffset & (QCOW2_COMPRESSED_SECTOR_SIZE - 1)))
}

func qcow2_zlib_decompress(bs *BlockDriverState, dest *[]byte, src []byte) error {
	destBuf := bytes.NewBuffer(*dest)
	srcBuf := bytes.NewReader(src)


	r := flate.NewReader(srcBuf)

	_, err := io.Copy(destBuf, r)
	r.Close()
	if err != nil {
		return err
	}
	*dest = destBuf.Bytes()
	return nil
}

type cacheLine struct {
	l2Entry uint64
	buf []byte
}

type cache struct {
	idx int
	cache []*cacheLine
}

var compCache cache = cache{
	cache: make([]*cacheLine, 50),
}

func qcow2_readv_compressed(bs *BlockDriverState, l2Entry uint64, offset uint64, bytes uint64, qiov *QEMUIOVector, qiovOffset uint64) error {
	s := bs.opaque.(*BDRVQcow2State)
	offsetInCluster := offset_into_cluster(s, offset)
	var cOffset uint64
	var cSize int

	qcow2_parse_compressed_l2_entry(bs, l2Entry, &cOffset,  &cSize)

	// Check the cache.  If the entry is already in the cache, use it.
	// If not, read the data and kick out an existing entry if the
	// cache is full.
	var buf_out []byte
	for _, cl := range compCache.cache {
		if cl == nil {
			continue
		}
		if l2Entry == cl.l2Entry {
			buf_out = cl.buf
			break
		}
	}
	if buf_out == nil {
		bufBytes := make([]byte, cSize * 2, cSize * 2)
		buf := unsafe.Pointer(&bufBytes[0])

		err := bdrv_pread(bs.current, cOffset, buf, uint64(cSize))
		if err != nil {
			return err
		}

		// Only try zlib for now
		buf_out = make([]byte, 0, cSize)
		err = qcow2_zlib_decompress(bs, &buf_out, bufBytes)
		if err != nil {
			return err
		}

		compCache.cache[compCache.idx] = &cacheLine{
			l2Entry: l2Entry,
			buf: buf_out,
		}
		compCache.idx = (compCache.idx + 1) % len(compCache.cache)
	}

	qemu_iovec_from_buf(qiov, qiovOffset, unsafe.Pointer(&buf_out[offsetInCluster]), uint64(len(buf_out)))

	return nil
}

func qcow2_preadv_task(bs *BlockDriverState, scType QCow2SubclusterType,
	hostOffset uint64, offset uint64, bytes uint64, qiov *QEMUIOVector, qiovOffset uint64) error {

	s := bs.opaque.(*BDRVQcow2State)
	switch scType {
	case QCOW2_SUBCLUSTER_ZERO_PLAIN, QCOW2_SUBCLUSTER_ZERO_ALLOC:
		/* Both zero types are handled in qcow2_co_preadv_part */
		Assert(false)
	case QCOW2_SUBCLUSTER_UNALLOCATED_PLAIN, QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC:
		return bdrv_preadv_part(bs.backing, offset, bytes, qiov, qiovOffset, 0)
	case QCOW2_SUBCLUSTER_COMPRESSED:
		//do nothing
		return qcow2_readv_compressed(bs, hostOffset, offset, bytes, qiov, qiovOffset)
	case QCOW2_SUBCLUSTER_NORMAL:
		return bdrv_preadv_part(s.DataFile, hostOffset,
			bytes, qiov, qiovOffset, 0)
	default:
		Assert(false)
	}
	Assert(false)
	return nil
}

func qcow2_pwritev_task(bs *BlockDriverState, hostOffset uint64, offset uint64,
	bytes uint64, qiov *QEMUIOVector, qiovOffset uint64, l2meta *QCowL2Meta) error {

	var err error
	s := bs.opaque.(*BDRVQcow2State)

	/* Try to efficiently initialize the physical space with zeroes */
	if err = handle_alloc_space(bs, l2meta); err != nil {
		goto out_unlocked
	}

	if !merge_cow(offset, bytes, qiov, qiovOffset, l2meta) {
		if err = bdrv_pwritev_part(s.DataFile, hostOffset,
			bytes, qiov, qiovOffset, 0); err != nil {
			goto out_unlocked
		}
	}

	s.Qlock()
	err = qcow2_handle_l2meta(bs, &l2meta, true)
	goto out_locked

out_unlocked:
	s.Qlock()

out_locked:
	qcow2_handle_l2meta(bs, &l2meta, false)
	s.Qunlock()
	return err
}

func handle_alloc_space(bs *BlockDriverState, l2meta *QCowL2Meta) error {

	s := bs.opaque.(*BDRVQcow2State)
	var m *QCowL2Meta
	var err error
	if s.DataFile.bs.SupportedZeroFlags&BDRV_REQ_NO_FALLBACK == 0 {
		return nil
	}

	for m = l2meta; m != nil; m = m.Next {
		var ret bool
		startOffset := m.AllocOffset + m.CowStart.Offset
		nbBytes := m.CowEnd.Offset + m.CowEnd.NbBytes - m.CowStart.Offset

		if m.CowStart.NbBytes == 0 && m.CowEnd.NbBytes == 0 {
			continue
		}

		ret, err = is_zero_cow(bs, m)
		if err != nil {
			return err
		} else if !ret {
			continue
		}

		if err = bdrv_pwrite_zeroes(s.DataFile, startOffset, nbBytes, BDRV_REQ_NO_FALLBACK); err != nil {
			if err != ERR_ENOTSUP && err != ERR_EAGAIN {
				return err
			}
			continue
		}
		m.SkipCow = true
	}
	return nil
}

func merge_cow(offset uint64, bytes uint64, qiov *QEMUIOVector, qiovOffset uint64, l2meta *QCowL2Meta) bool {

	var m *QCowL2Meta

	for m = l2meta; m != nil; m = m.Next {
		/* If both COW regions are empty then there's nothing to merge */
		if m.CowStart.NbBytes == 0 && m.CowEnd.NbBytes == 0 {
			continue
		}
		/* If COW regions are handled already, skip this too */
		if m.SkipCow {
			continue
		}

		if l2meta_cow_start(m)+m.CowStart.NbBytes != offset {
			/* In this case the request starts before this region */
			Assert(offset < l2meta_cow_start(m))
			Assert(m.CowStart.NbBytes == 0)
			continue
		}

		/* The write request should end immediately before the second
		 * COW region (see above for why it does not always happen) */
		if m.Offset+m.CowEnd.Offset != offset+bytes {
			Assert(offset+bytes > m.Offset+m.CowEnd.Offset)
			Assert(m.CowEnd.NbBytes == 0)
			continue
		}
		/* Make sure that adding both COW regions to the QEMUIOVector
		 * does not exceed IOV_MAX */
		if qemu_iovec_subvec_niov(qiov, qiovOffset, bytes) > IOV_MAX-2 {
			continue
		}

		m.DataQiov = qiov
		m.DataQiovOffset = qiovOffset
		return true
	}

	return false
}

func is_zero_cow(bs *BlockDriverState, m *QCowL2Meta) (bool, error) {
	var ret bool
	var err error
	if ret, err = bdrv_is_zero_fast(bs, m.Offset+m.CowStart.Offset,
		m.CowStart.NbBytes); err != nil {
		return ret, err
	}
	return bdrv_is_zero_fast(bs, m.Offset+m.CowEnd.Offset,
		m.CowEnd.NbBytes)
}

func is_zero(bs *BlockDriverState, offset uint64, bytes uint64) bool {
	var nr uint64
	var res uint64
	var err error

	/* Clamp to image length, before checking status of underlying sectors */
	if offset+bytes > bs.TotalSectors*BDRV_SECTOR_SIZE {
		bytes = bs.TotalSectors*BDRV_SECTOR_SIZE - offset
	}

	if bytes == 0 {
		return true
	}

	for {
		res, err = bdrv_block_status_above(bs, nil, offset, bytes, &nr, nil, nil)
		offset += nr
		bytes -= nr
		if err == nil && (res&BDRV_BLOCK_ZERO > 0) && nr > 0 && bytes > 0 {
		} else {
			break
		}
	}

	return err == nil && (res&BDRV_BLOCK_ZERO) > 0 && bytes == 0
}

func qcow2_flush_to_os(bs *BlockDriverState) error {
	s := bs.opaque.(*BDRVQcow2State)
	s.Qlock()
	defer s.Qunlock()
	return qcow2_write_caches(bs)
}

func qcow2_refcount_metadata_size(clusters uint64, clusterSize uint64, refcountOrder int,
	generousIncrease bool, refblockCount *uint64) (uint64, error) {
	//TODO2
	//so far never reach here due to qcow2 file is limited to 4TiB
	return 0, nil
}

func qcow2_block_status(bs *BlockDriverState, wantZero bool, offset uint64,
	count uint64, pnum *uint64, tmap *uint64, file **BlockDriverState) (uint64, error) {
	s := bs.opaque.(*BDRVQcow2State)
	var hostOffset uint64
	var bytes uint32
	var scType QCow2SubclusterType
	var status uint64
	var err error

	s.Qlock()
	bytes = uint32(count)
	err = qcow2_get_host_offset(bs, offset, &bytes, &hostOffset, &scType)
	s.Qunlock()
	if err != nil {
		return 0, err
	}

	*pnum = uint64(bytes)

	if scType == QCOW2_SUBCLUSTER_NORMAL ||
		scType == QCOW2_SUBCLUSTER_ZERO_ALLOC ||
		scType == QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC {
		*tmap = hostOffset
		*file = s.DataFile.bs
		status |= BDRV_BLOCK_OFFSET_VALID
	}
	if scType == QCOW2_SUBCLUSTER_ZERO_PLAIN ||
		scType == QCOW2_SUBCLUSTER_ZERO_ALLOC {
		status |= BDRV_BLOCK_ZERO
	} else if scType != QCOW2_SUBCLUSTER_UNALLOCATED_PLAIN &&
		scType != QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC {
		status |= BDRV_BLOCK_DATA
	}
	return status, nil
}

func qcow2_copy_range_from(bs *BlockDriverState, src *BdrvChild, offset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error {
	//do nothing
	return fmt.Errorf("[qcow2_copy_range_from] no implementation")
}

func qcow2_copy_range_to(bs *BlockDriverState, src *BdrvChild, offset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error {
	//do nothing
	return fmt.Errorf("[qcow2_copy_range_to] no implementation")
}

func qcow2_pdiscard(bs *BlockDriverState, offset uint64, bytes uint64) error {
	s := bs.opaque.(*BDRVQcow2State)
	if !is_aligned(offset|bytes, uint64(s.ClusterSize)) {
		Assert(bytes < uint64(s.ClusterSize))
		if !is_aligned(offset, uint64(s.ClusterSize)) ||
			offset+bytes != bs.TotalSectors*BDRV_SECTOR_SIZE {
			return ERR_ENOTSUP
		}
	}
	s.Qlock()
	defer s.Qunlock()
	return qcow2_cluster_discard(bs, offset, bytes, QCOW2_DISCARD_REQUEST, false)
}

// write external datafile ext header at offset
func header_ext_add_external_data_file(bs *BlockDriverState, offset uint64, dataFile string) (uint64, error) {
	return header_ext_add(bs, QCOW2_EXT_MAGIC_DATA_FILE, offset, dataFile)
}

// write backing file format ext header at offset
func header_ext_add_backing_file_format(bs *BlockDriverState, offset uint64, format string) (uint64, error) {
	return header_ext_add(bs, QCOW2_EXT_MAGIC_BACKING_FMT, offset, format)
}

// write the ext header after the qcow2 regular header returns length written to disk
func header_ext_add(bs *BlockDriverState, magic uint32, offset uint64, data string) (uint64, error) {
	extHeader := &QCowExtension{
		Magic:  magic,
		Length: uint32(len(data)),
	}
	extLen := uint64(unsafe.Sizeof(*extHeader))
	//write to the ext header
	if _, err := Blk_Pwrite_Object(bs.current, offset, extHeader, extLen); err != nil {
		return 0, err
	}
	//write the length data
	if _, err := Blk_Pwrite_Object(bs.current, offset+extLen, ([]byte)(data), uint64(len(data))); err != nil {
		return extLen, err
	}

	totalSize := extLen + uint64(len(data))
	if totalSize%8 == 0 {
		// No need of padding as size is multiple of 8
		return totalSize, nil
	}

	// header extension size must be multiple of 8
	paddingLength := (8 - totalSize%8) % 8
	paddingOffset := offset + totalSize
	paddingBytes := strings.Repeat("\x00", int(paddingLength))

	// write the padding data
	if _, err := Blk_Pwrite_Object(bs.current, paddingOffset, ([]byte)(paddingBytes), uint64(paddingLength)); err != nil {
		return totalSize, err
	}

	return totalSize + paddingLength, nil
}

func header_ext_read_external_data_file(bs *BlockDriverState, offset uint64) (string, error) {
	return header_ext_read(bs, offset, QCOW2_EXT_MAGIC_DATA_FILE)
}

func header_ext_read_backing_file_format(bs *BlockDriverState, offset uint64) (string, error) {
	return header_ext_read(bs, offset, QCOW2_EXT_MAGIC_BACKING_FMT)
}

func header_ext_read(bs *BlockDriverState, offset uint64, magic uint32) (string, error) {

	qcow2HeaderExtensionReadError := func(err error) error {
		return fmt.Errorf("qcow2 header extension read fail, err: %v", err)
	}

	var extHeader QCowExtension
	extLen := uint64(unsafe.Sizeof(extHeader))
	//read the ext header
	if _, err := Blk_Pread_Object(bs.current, offset, &extHeader, extLen); err != nil {
		return "", qcow2HeaderExtensionReadError(err)
	}
	if extHeader.Magic != magic || extHeader.Length == 0 {
		return "", nil
	}
	//read the data file path
	var data string
	bytes := make([]byte, extHeader.Length)
	if _, err := Blk_Pread_Object(bs.current, offset+extLen, bytes, uint64(extHeader.Length)); err != nil {
		return "", qcow2HeaderExtensionReadError(err)
	} else {
		data = string(bytes)
	}
	return data, nil
}

func qcow2_pwritev_task_entry(task *Qcow2Task) error {
	Assert(task.subclusterType == 0)
	return qcow2_pwritev_task(task.bs, task.hostOffset, task.offset, task.bytes, task.qiov, task.qiovOffset, task.l2meta)
}

func qcow2_preadv_task_entry(task *Qcow2Task) error {
	Assert(task.l2meta == nil)
	return qcow2_preadv_task(task.bs, task.subclusterType, task.hostOffset, task.offset, task.bytes, task.qiov, task.qiovOffset)
}

func qcow2_add_task(bs *BlockDriverState, isAio bool, taskfunc AioTaskFunc, subclusterType QCow2SubclusterType,
	hostOffset uint64, offset uint64, bytes uint64, qiov *QEMUIOVector, qiovOffset uint64,
	l2meta *QCowL2Meta) error {

	var err error
	s := bs.opaque.(*BDRVQcow2State)
	task := &Qcow2Task{
		taskFunc:       taskfunc,
		bs:             bs,
		subclusterType: subclusterType,
		qiov:           qiov,
		hostOffset:     hostOffset,
		offset:         offset,
		bytes:          bytes,
		qiovOffset:     qiovOffset,
		l2meta:         l2meta,
		completeCh:     make(chan any, 1),
		errorCh:        make(chan error, 1),
	}

	if !isAio {
		return taskfunc(task)
	}

	if s.AioTaskList == nil {
		s.Qlock()
		if s.AioTaskList == nil {
			s.AioTaskList = NewSignalList()
			go s.AioTaskRoutine(context.Background(), s.AioTaskList)
		}
		s.Qunlock()
	}

	s.AioTaskList.Push(task)
	select {
	case <-task.completeCh:
	case err = <-task.errorCh:
	}
	return err
}
