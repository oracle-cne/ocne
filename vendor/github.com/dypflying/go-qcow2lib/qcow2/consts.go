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

var (
	QCOW_MAGIC = []byte{0x51, 0x46, 0x49, 0xFB} //magic ("QFI\xfb").
)

const (
	MAX_QCOW2_SIZE                     = 1 << 42 //single qcow2 file limit within 4TiB
	QCOW_EXTL2_SUBCLUSTERS_PER_CLUSTER = uint64(32)
	QCOW_L2_BITMAP_ALL_ALLOC           = uint64(1)<<32 - 1
	QCOW_L2_BITMAP_ALL_ZEROES          = QCOW_L2_BITMAP_ALL_ALLOC << 32
	QCOW_MAX_REFTABLE_SIZE             = (8 * 1024 * 1024)
)

// L1 & L2 & Refcount masks
const (
	L1E_OFFSET_MASK  = uint64(0x00fffffffffffe00)
	L2E_OFFSET_MASK  = uint64(0x00fffffffffffe00)
	REFT_OFFSET_MASK = uint64(0xfffffffffffffe00)
	INV_OFFSET       = uint64(0xff00000000000000)
)

const (
	DEFAULT_CLUSTER_SIZE = 65536
	DEFAULT_SECTOR_SIZE  = 512
	DEFAULT_CLUSTER_BITS = 16
	//1 refcount block can hold 2GiB data (64k * 32k),
	//1 refcount table holds 8k refcount blocks,
	//so 1 table can hold up to 16TiB data (including refcount blocks and l2 blocks and data blocks)
	DEFAULT_REFCOUNT_TABLE_CLUSTERS = 1
	QCOW2_VERSION2                  = 2
	QCOW2_VERSION3                  = 3
	QCOW2_REFCOUNT_ORDER            = 4
	QCOW2_CRYPT_METHOD              = 0
	//	DEFAULT_ALIGNMENT               = 4096    //align to 4k
	DEFAULT_ALIGNMENT    = DEFAULT_SECTOR_SIZE //align to sector
	DEFAULT_MAX_TRANSFER = 1 << 31             //2G
	DEFAULT_L2_CACHE     = 1024 * 1024         //default 1MiB for l2 cache
)

// backing file offset
const (
	BACKING_FILE_OFFSET = 32768
)

// L1 table offset
const (
	L1_TABLE_OFFSET = 65536 * 3
	L1_TABLE_SIZE   = 65536 >> 3
)

// refcount table offset
const (
	REFCOUNT_TABLE_OFFSET = 65536
	REFCOUNT_TABLE_SIZE   = 65536 >> 3
)

// L1 & L2 bit options
const (
	QCOW_OFLAG_COPIED     = 1 << 63
	QCOW_OFLAG_COMPRESSED = 1 << 62
	QCOW_OFLAG_ZERO       = 1 << 0
)

const (
	QCOW2_INCOMPAT_DIRTY_BITNR       = 0
	QCOW2_INCOMPAT_CORRUPT_BITNR     = 1
	QCOW2_INCOMPAT_DATA_FILE_BITNR   = 2
	QCOW2_INCOMPAT_COMPRESSION_BITNR = 3
	QCOW2_INCOMPAT_EXTL2_BITNR       = 4
	QCOW2_INCOMPAT_DIRTY             = 1 << QCOW2_INCOMPAT_DIRTY_BITNR
	QCOW2_INCOMPAT_CORRUPT           = 1 << QCOW2_INCOMPAT_CORRUPT_BITNR
	QCOW2_INCOMPAT_DATA_FILE         = 1 << QCOW2_INCOMPAT_DATA_FILE_BITNR
	QCOW2_INCOMPAT_COMPRESSION       = 1 << QCOW2_INCOMPAT_COMPRESSION_BITNR
	QCOW2_INCOMPAT_EXTL2             = 1 << QCOW2_INCOMPAT_EXTL2_BITNR
	QCOW2_INCOMPAT_MASK              = QCOW2_INCOMPAT_DIRTY | QCOW2_INCOMPAT_CORRUPT | QCOW2_INCOMPAT_DATA_FILE | QCOW2_INCOMPAT_COMPRESSION | QCOW2_INCOMPAT_EXTL2
)

/* Autoclear feature bits */
const (
	QCOW2_AUTOCLEAR_BITMAPS_BITNR       = 0
	QCOW2_AUTOCLEAR_DATA_FILE_RAW_BITNR = 1
	QCOW2_AUTOCLEAR_BITMAPS             = 1 << QCOW2_AUTOCLEAR_BITMAPS_BITNR
	QCOW2_AUTOCLEAR_DATA_FILE_RAW       = 1 << QCOW2_AUTOCLEAR_DATA_FILE_RAW_BITNR
	QCOW2_AUTOCLEAR_MASK                = QCOW2_AUTOCLEAR_BITMAPS | QCOW2_AUTOCLEAR_DATA_FILE_RAW
)

// cluster type
const (
	QCOW2_CLUSTER_UNALLOCATED = iota
	QCOW2_CLUSTER_ZERO_PLAIN
	QCOW2_CLUSTER_ZERO_ALLOC
	QCOW2_CLUSTER_NORMAL
	QCOW2_CLUSTER_COMPRESSED
)

const ( //permission options
	//used features
	BDRV_O_RDWR   = 0x0002
	BDRV_O_UNMAP  = 0x4000  /* execute guest UNMAP/TRIM operations */
	BDRV_O_CREATE = 0x80000 /* create of non-exist */

	//unused feratures
	BDRV_O_NO_SHARE     = 0x0001 /* don't share permissions */
	BDRV_O_RESIZE       = 0x0004 /* request permission for resizing the node */
	BDRV_O_SNAPSHOT     = 0x0008 /* open the file read only and save writes in a snapshot */
	BDRV_O_TEMPORARY    = 0x0010 /* delete the file after use */
	BDRV_O_NOCACHE      = 0x0020 /* do not use the host page cache */
	BDRV_O_NATIVE_AIO   = 0x0080 /* use native AIO instead of the thread pool */
	BDRV_O_NO_BACKING   = 0x0100 /* don't open the backing file */
	BDRV_O_NO_FLUSH     = 0x0200 /* disable flushing on this disk */
	BDRV_O_COPY_ON_READ = 0x0400 /* copy read backing sectors into image */
	BDRV_O_INACTIVE     = 0x0800 /* consistency hint for migration handoff */
	BDRV_O_CHECK        = 0x1000 /* open solely for consistency check */
	BDRV_O_ALLOW_RDWR   = 0x2000 /* allow reopen to change from r/o to r/w */
	BDRV_O_PROTOCOL     = 0x8000
	BDRV_O_NO_IO        = 0x10000 /* don't initialize for I/O */
	BDRV_O_AUTO_RDONLY  = 0x20000 /* degrade to read-only if opening read-write fails */
	BDRV_O_IO_URING     = 0x40000 /* use io_uring instead of the thread pool */

	BDRV_O_CACHE_MASK = (BDRV_O_NOCACHE | BDRV_O_NO_FLUSH)
)

const (
	BDRV_SECTOR_BITS = 9
	BDRV_SECTOR_SIZE = (1 << BDRV_SECTOR_BITS)
)

/*
 * Allocation status flags for bdrv_block_status() and friends.
 *
 * Public flags:
 * BDRV_BLOCK_DATA: allocation for data at offset is tied to this layer
 * BDRV_BLOCK_ZERO: offset reads as zero
 * BDRV_BLOCK_OFFSET_VALID: an associated offset exists for accessing raw data
 * BDRV_BLOCK_ALLOCATED: the content of the block is determined by this
 *                       layer rather than any backing, set by block layer
 * BDRV_BLOCK_EOF: the returned pnum covers through end of file for this
 *                 layer, set by block layer
 *
 * Internal flags:
 * BDRV_BLOCK_RAW: for use by passthrough drivers, such as raw, to request
 *                 that the block layer recompute the answer from the returned
 *                 BDS; must be accompanied by just BDRV_BLOCK_OFFSET_VALID.
 * BDRV_BLOCK_RECURSE: request that the block layer will recursively search for
 *                     zeroes in file child of current block node inside
 *                     returned region. Only valid together with both
 *                     BDRV_BLOCK_DATA and BDRV_BLOCK_OFFSET_VALID. Should not
 *                     appear with BDRV_BLOCK_ZERO.
 *
 * If BDRV_BLOCK_OFFSET_VALID is set, the map parameter represents the
 * host offset within the returned BDS that is allocated for the
 * corresponding raw guest data.  However, whether that offset
 * actually contains data also depends on BDRV_BLOCK_DATA, as follows:
 *
 * DATA ZERO OFFSET_VALID
 *  t    t        t       sectors read as zero, returned file is zero at offset
 *  t    f        t       sectors read as valid from file at offset
 *  f    t        t       sectors preallocated, read as zero, returned file not
 *                        necessarily zero at offset
 *  f    f        t       sectors preallocated but read from backing_hd,
 *                        returned file contains garbage at offset
 *  t    t        f       sectors preallocated, read as zero, unknown offset
 *  t    f        f       sectors read from unknown file or offset
 *  f    t        f       not allocated or unknown offset, read as zero
 *  f    f        f       not allocated or unknown offset, read from backing_hd
 */
const (
	BDRV_BLOCK_DATA         = 0x01
	BDRV_BLOCK_ZERO         = 0x02
	BDRV_BLOCK_OFFSET_VALID = 0x04
	BDRV_BLOCK_RAW          = 0x08
	BDRV_BLOCK_ALLOCATED    = 0x10
	BDRV_BLOCK_EOF          = 0x20
	BDRV_BLOCK_RECURSE      = 0x40
)

// options
const (
	OPT_FMT              = "fmt"
	OPT_SIZE             = "size"
	OPT_FILENAME         = "filename"
	OPT_BACKING          = "backing"
	OPT_SUBCLUSTER       = "enable-subcluster"
	OPT_L2CACHESIZE      = "l2-cache-size"
	OPT_DATAFILE         = "datafile"
	OPT_BACKING_FILE_FMT = "backingFileFmt"
)

/* permission constants */
const (
	PERM_READ     = 0x01
	PERM_WRITE    = 0x02
	PERM_RESIZE   = 0x04
	PERM_ALL      = PERM_READ | PERM_WRITE | PERM_RESIZE
	PERM_READABLE = PERM_READ
	PERM_WRITABLE = PERM_WRITE | PERM_RESIZE
)

const Max_WRITE_ZEROS = uint64(65536)
const MAX_BOUNCE_BUFFER = uint64(32768 << 9)

// external data file magic number
const (
	QCOW2_EXT_MAGIC_DATA_FILE   = uint32(0x44415441)
	QCOW2_EXT_MAGIC_BACKING_FMT = uint32(0xe2792aca)
)
