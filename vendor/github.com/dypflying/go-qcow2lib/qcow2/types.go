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
	"container/list"
	"encoding/json"
	"os"
	"sync"
	"unsafe"
)

// the qcow2 header struct, compatible with version 3
type QCowHeader struct {
	Magic                 uint32
	Version               uint32
	BackingFileOffset     uint64
	BackingFileSize       uint32
	ClusterBits           uint32
	Size                  uint64 /* in bytes */
	CryptMethod           uint32
	L1Size                uint32
	L1TableOffset         uint64
	RefcountTableOffset   uint64
	RefcountTableClusters uint32
	NbSnapshots           uint32
	SnapshotsOffset       uint64

	/* The following fields are only valid for version >= 3 */
	IncompatibleFeatures uint64
	CompatibleFeatures   uint64
	AutoclearFeatures    uint64

	RefcountOrder uint32
	HeaderLength  uint32

	/* Additional fields */
	CompressionType uint8

	/* header must be a multiple of 8 */
	Padding [7]uint8 `json:"-"`
}

//for raw file state

type BDRVRawState struct {
	File      *os.File
	OpenFlags int
	BufAlign  uint64

	/* The current permissions. */
	Perm       uint64
	SharedPerm uint64
}

// for qcow2 file state
type BDRVQcow2State struct {
	ClusterBits       uint32
	ClusterSize       uint32
	L2Bits            uint32
	L2Size            uint32
	L1Size            uint32
	RefcountBlockBits uint32
	RefcountBlockSize uint32

	ClusterOffsetMask uint64
	L1TableOffset     uint64
	L1Table           []uint64

	L2TableCache       *Qcow2Cache
	RefcountBlockCache *Qcow2Cache

	RefcountTable       []uint64
	RefcountTableOffset uint64
	RefcountTableSize   uint32

	MaxRefcountTableIndex uint32
	FreeClusterIndex      uint64
	QcowVersion           int

	FreeByteOffset uint64 //not used
	Lock           *sync.Mutex
	Flags          int //not used

	L2SliceSize int

	L2eOffsetMask uint64

	//variables for subcluster feature
	SubclusterBits        uint64
	SubclustersPerCluster uint64
	SubclusterSize        uint64

	ClusterAllocs *list.List
	Discards      *list.List

	get_refcount Get_Refcount_Func
	set_refcount Set_Refcount_Func

	DataFile *BdrvChild

	CacheDiscards      bool
	DiscardPassthrough [QCOW2_DISCARD_MAX]bool

	AioTaskList    *SignalList
	AioTaskRoutine AioTaskRoutineFunc

	/* The following fields are only valid for version >= 3 */
	IncompatibleFeatures uint64
	CompatibleFeatures   uint64
	AutoclearFeatures    uint64
}

func (s *BDRVQcow2State) Qlock() {
	if s.Lock != nil {
		s.Lock.Lock()
	}
}

func (s *BDRVQcow2State) Qunlock() {
	if s.Lock != nil {
		s.Lock.Unlock()
	}
}

type QCowL2Meta struct {
	Offset          uint64
	AllocOffset     uint64
	NbClusters      int
	KeepOldClusters bool
	CowStart        Qcow2COWRegion
	CowEnd          Qcow2COWRegion
	SkipCow         bool

	DataQiov       *QEMUIOVector
	DataQiovOffset uint64

	/** Pointer to next L2Meta of the same write request */
	Next         *QCowL2Meta
	NextInFlight *list.Element
}

type Qcow2COWRegion struct {
	Offset  uint64
	NbBytes uint64
}

type QEMUIOVector struct {
	iov       []iovec
	niov      int //total iov
	nalloc    int //allocate iov
	local_iov iovec
	//the size must be always equal to local_iov.iov_len, since the original c implementation use a union struct.
	size uint64
}

type iovec struct {
	iov_base unsafe.Pointer
	iov_len  uint64
}

/*
 * Request padding
 *
 *  |<---- align ----->|                     |<----- align ---->|
 *  |<- head ->|<------------- bytes ------------->|<-- tail -->|
 *  |          |       |                     |     |            |
 * -*----------$-------*-------- ... --------*-----$------------*---
 *  |          |       |                     |     |            |
 *  |          offset  |                     |     end          |
 *  ALIGN_DOWN(offset) ALIGN_UP(offset)      ALIGN_DOWN(end)   ALIGN_UP(end)
 *  [buf   ... )                             [tail_buf          )
 *
 * @buf is an aligned allocation needed to store @head and @tail paddings. @head
 * is placed at the beginning of @buf and @tail at the @end.
 *
 * @tail_buf is a pointer to sub-buffer, corresponding to align-sized chunk
 * around tail, if tail exists.
 *
 * @merge_reads is true for small requests,
 * if @buf_len == @head + bytes + @tail. In this case it is possible that both
 * head and tail exist but @buf_len == align and @tail_buf == @buf.
 */
type BdrvRequestPadding struct {
	Buf        []uint8
	BufLen     uint64
	TailBuf    []uint8
	Head       uint64
	Tail       uint64
	MergeReads bool
	LocalQiov  QEMUIOVector
}

type Get_Refcount_Func func(refcountArray unsafe.Pointer, index uint64) uint16
type Set_Refcount_Func func(refcountArray unsafe.Pointer, index uint64, value uint16)

type BlockDriverState struct {
	opaque      any
	filename    string
	backingFile string
	backing     *BdrvChild //so far, not used.
	current     *BdrvChild
	options     map[string]any
	//static configuration
	RequestAlignment  uint32
	PdiscardAlignment uint32
	MaxTransfer       uint32
	//statistic information
	InFlight            uint64
	SupportedWriteFlags uint64
	SupportedReadFlags  uint64
	SupportedZeroFlags  uint64
	OpenFlags           int /* flags used to open the file, re-used for re-open */
	TotalSectors        uint64
	InheritsFrom        *BlockDriverState
	Drv                 *BlockDriver
}

type BdrvChild struct {
	name   string
	bs     *BlockDriverState
	perm   uint8
	header *QCowHeader
}

func (child *BdrvChild) SetBS(bs *BlockDriverState) {
	child.bs = bs
}

func (child *BdrvChild) GetBS() *BlockDriverState {
	return child.bs
}

func (bs *BlockDriverState) Info(detail bool, pretty bool) string {

	if bs.current == nil || bs.current.header == nil {
		return ""
	}
	var info BlockInfo
	info.BakcingFileChain = make([]string, 0)
	//get basic information
	info.FileFormat = "qcow2"
	info.VirtualSize = bs.current.header.Size
	if fileinfo, err := os.Stat(bs.filename); err == nil {
		info.DiskSize = uint64(fileinfo.Size())
	}
	info.ClusterSize = 1 << bs.current.header.ClusterBits
	info.RefcountBits = 1 << uint16(bs.current.header.RefcountOrder)
	info.ExtendedL2 = bs.current.header.IncompatibleFeatures&QCOW2_INCOMPAT_EXTL2 > 0

	//get backing chain
	if bs.backing != nil {
		getBackingChain(bs.backing, &info.BakcingFileChain)
	}
	if has_data_file(bs) {
		s := bs.opaque.(*BDRVQcow2State)
		info.DataFile = s.DataFile.name
	}

	//get statistic information
	if detail {
		var stat BlockStatistic
		scanRefcountTable(bs, &stat)
		info.Statistic = &stat
	}

	//response
	var bytes []byte
	if pretty {
		bytes, _ = json.MarshalIndent(info, "", "\t")
	} else {
		bytes, _ = json.Marshal(info)
	}
	return string(bytes)
}

func scanRefcountTable(bs *BlockDriverState, stat *BlockStatistic) {
	if bs == nil || bs.opaque == nil {
		return
	}
	s := bs.opaque.(*BDRVQcow2State)
	var err error
	var p unsafe.Pointer
	//firstly scan the refcount table and block
	for i := 0; i < len(s.RefcountTable); i++ {
		refcountBlockOffset := s.RefcountTable[i] & REFT_OFFSET_MASK
		if refcountBlockOffset > 0 {
			if p, err = load_refcount_block(bs, refcountBlockOffset); err != nil {
				return
			} else {
				for j := uint64(0); j < DEFAULT_CLUSTER_SIZE/2; j++ {
					if s.get_refcount(p, j) > 0 {
						stat.TotalBlocks++
					}
				}
			}
			stat.RecountBlocks++
		}
	}
	stat.RefcountTableBlocks = 1
	stat.HeadBlocks = 1
	stat.L1Blocks = 1

	//then scan the l1 table and block
	for i := 0; i < len(s.L1Table); i++ {
		l2BlockOffset := s.L1Table[i] & L1E_OFFSET_MASK
		if l2BlockOffset > 0 {
			stat.L2Blocks++
		}
	}
	stat.DataBlocks = stat.TotalBlocks - stat.L1Blocks - stat.RefcountTableBlocks - stat.HeadBlocks - stat.RecountBlocks - stat.L2Blocks
}

func getBackingChain(child *BdrvChild, list *[]string) {

	if child == nil {
		return
	}
	*list = append(*list, child.name)
	if child.bs != nil {
		getBackingChain(child.bs.backing, list)
	}
}

type Bdrv_Open_Func func(filename string, options map[string]any, flags int) (*BlockDriverState, error)
type Bdrv_Close_Func func(bs *BlockDriverState)
type Bdrv_Create_Func func(filename string, options map[string]any) error
type Bdrv_Block_Status_Func func(bs *BlockDriverState, want_zero bool, offset uint64, bytes uint64,
	pnum *uint64, tmap *uint64, file **BlockDriverState) (uint64, error)
type Bdrv_Pwritev_Func func(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error
type Bdrv_Preadv_Func func(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error
type Bdrv_Pwritev_Part_Func func(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error
type Bdrv_Preadv_Part_Func func(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error
type Bdrv_Flush_Func func(bs *BlockDriverState) error
type Bdrv_Flush_To_Os_Func func(bs *BlockDriverState) error
type Bdrv_Flush_To_Disk_Func func(bs *BlockDriverState) error
type Bdrv_Pwrite_Zeroes_Func func(bs *BlockDriverState, offset uint64, bytes uint64, flags BdrvRequestFlags) error
type Bdrv_Getlength_Func func(bs *BlockDriverState) (uint64, error)

type Bdrv_Copy_Range_From_Func func(bs *BlockDriverState, src *BdrvChild, srcOffset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error
type Bdrv_Copy_Range_To_Func func(bs *BlockDriverState, src *BdrvChild, srcOffset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error
type Bdrv_Pdiscard_Func func(bs *BlockDriverState, offset uint64, bytes uint64) error

type BlockDriver struct {
	FormatName     string
	InstanceSize   int
	SupportBacking bool
	IsFormat       bool
	//functions
	bdrv_open            Bdrv_Open_Func
	bdrv_close           Bdrv_Close_Func
	bdrv_create          Bdrv_Create_Func
	bdrv_block_status    Bdrv_Block_Status_Func
	bdrv_pwritev_part    Bdrv_Pwritev_Part_Func
	bdrv_pwritev         Bdrv_Pwritev_Func
	bdrv_preadv_part     Bdrv_Preadv_Part_Func
	bdrv_preadv          Bdrv_Preadv_Func
	bdrv_flush           Bdrv_Flush_Func
	bdrv_flush_to_os     Bdrv_Flush_To_Os_Func
	bdrv_flush_to_disk   Bdrv_Flush_To_Disk_Func
	bdrv_pwrite_zeroes   Bdrv_Pwrite_Zeroes_Func
	bdrv_getlength       Bdrv_Getlength_Func
	bdrv_copy_range_from Bdrv_Copy_Range_From_Func //for convert copy
	bdrv_copy_range_to   Bdrv_Copy_Range_To_Func   //for convert copy
	bdrv_pdiscard        Bdrv_Pdiscard_Func
}

type BlockInfo struct {
	//based information
	FileFormat   string `json:"file format"`
	VirtualSize  uint64 `json:"virtual size"`
	DiskSize     uint64 `json:"disk size"`
	ClusterSize  uint32 `json:"cluster size"`
	RefcountBits uint16 `json:"refcount bits"`
	ExtendedL2   bool   `json:"extend l2"`
	//backing chain
	BakcingFileChain []string        `json:"backing chain"`
	DataFile         string          `json:"data file,omitempty"`
	Statistic        *BlockStatistic `json:"stat,omitempty"`
}

type BlockStatistic struct {
	TotalBlocks         uint64 `json:"total blocks,omitempty"`
	HeadBlocks          uint64 `json:"head blocks,omitempty"`
	L1Blocks            uint64 `json:"l1 blocks,omitempty"`
	RefcountTableBlocks uint64 `json:"refcount table blocks,omitempty"`
	L2Blocks            uint64 `json:"l2 blocks,omitempty"`
	RecountBlocks       uint64 `json:"refcount blocks,omitempty"`
	DataBlocks          uint64 `json:"data blocks,omitempty"`
}

type Qcow2DiscardRegion struct {
	bs     *BlockDriverState
	offset uint64
	bytes  uint64
	next   *Qcow2DiscardRegion
}

func has_data_file(bs *BlockDriverState) bool {
	s := bs.opaque.(*BDRVQcow2State)
	return s.DataFile != bs.current
}

func data_file_is_raw(bs *BlockDriverState) bool {
	s := bs.opaque.(*BDRVQcow2State)
	return s.AutoclearFeatures&QCOW2_AUTOCLEAR_DATA_FILE_RAW > 0
}

type QCowExtension struct {
	Magic  uint32
	Length uint32
}
