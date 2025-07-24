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

import "unsafe"

// bdev type
const (
	TYPE_RAW = iota
	TYPE_QCOW2
)
const (
	TYPE_RAW_NAME   = "raw"
	TYPE_QCOW2_NAME = "qcow2"
)

type FileType int

var (
	Supported_Types = map[string]FileType{
		TYPE_QCOW2_NAME: TYPE_QCOW2,
		TYPE_RAW_NAME:   TYPE_RAW,
	}
)

const (
	/* Size of L1 table entries */
	L1E_SIZE = 8
	/* Size of reftable entries */
	REFTABLE_ENTRY_SIZE = 8
	/* Size of normal and extended L2 entries */
	L2E_SIZE_NORMAL   = 8
	L2E_SIZE_EXTENDED = 8 * 2
	SIZE_UINT64       = uint64(unsafe.Sizeof(uint64(0)))
	SIZE_UINT32       = uint32(unsafe.Sizeof(uint32(0)))
	SIZE_INT          = int(unsafe.Sizeof(int(0)))
)

type QCow2ClusterType int

// subcluster type
const (
	QCOW2_SUBCLUSTER_UNALLOCATED_PLAIN = iota
	QCOW2_SUBCLUSTER_UNALLOCATED_ALLOC
	QCOW2_SUBCLUSTER_ZERO_PLAIN
	QCOW2_SUBCLUSTER_ZERO_ALLOC
	QCOW2_SUBCLUSTER_NORMAL
	QCOW2_SUBCLUSTER_COMPRESSED
	QCOW2_SUBCLUSTER_INVALID
)

type QCow2SubclusterType int

// Bdrv request flag
const (
	BDRV_REQ_COPY_ON_READ     = 0x1
	BDRV_REQ_ZERO_WRITE       = 0x2
	BDRV_REQ_MAY_UNMAP        = 0x4
	BDRV_REQ_FUA              = 0x10
	BDRV_REQ_WRITE_COMPRESSED = 0x20
	BDRV_REQ_WRITE_UNCHANGED  = 0x40
	BDRV_REQ_SERIALISING      = 0x80
	BDRV_REQ_NO_FALLBACK      = 0x100
	BDRV_REQ_PREFETCH         = 0x200
	BDRV_REQ_NO_WAIT          = 0x400
	BDRV_REQ_MASK             = 0x7ff
)

type BdrvRequestFlags int

const (
	QCOW2_DISCARD_NEVER = iota
	QCOW2_DISCARD_ALWAYS
	QCOW2_DISCARD_REQUEST
	QCOW2_DISCARD_SNAPSHOT
	QCOW2_DISCARD_OTHER
	QCOW2_DISCARD_MAX
)

type Qcow2DiscardType int
