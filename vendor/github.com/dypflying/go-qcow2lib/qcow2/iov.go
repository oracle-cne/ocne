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
	"math"
	"unsafe"
)

const (
	IOV_MAX = 1024
)

func New_QEMUIOVector() *QEMUIOVector {
	return &QEMUIOVector{}
}

func qemu_iovec_init(qiov *QEMUIOVector, allocHint int) {
	qiov.iov = make([]iovec, allocHint)
	qiov.niov = 0
	qiov.nalloc = allocHint
	qiov.size = 0
}

func qemu_iovec_add(qiov *QEMUIOVector, base unsafe.Pointer, len uint64) {
	if qiov.niov == qiov.nalloc {
		newNalloc := 2*qiov.nalloc + 1
		newIov := make([]iovec, newNalloc)
		for i := 0; i < qiov.nalloc; i++ {
			newIov[i] = qiov.iov[i]
		}
		qiov.iov = newIov
		qiov.nalloc = newNalloc
	}
	qiov.iov[qiov.niov].iov_base = base
	qiov.iov[qiov.niov].iov_len = len
	qiov.size += len
	qiov.niov++
}

func qemu_iovec_init_buf(qiov *QEMUIOVector, buf unsafe.Pointer, len uint64) {
	qiov.niov = 1
	qiov.nalloc = -1
	qiov.size = len
	qiov.local_iov = iovec{
		iov_base: buf,
		iov_len:  len,
	}
	qiov.iov = make([]iovec, 1)
	qiov.iov[0] = qiov.local_iov
}

func iov_from_buf(iov []iovec, iovCnt uint64, offset uint64, buf unsafe.Pointer, bytes uint64) uint64 {
	var done uint64 = 0
	var i uint64 = 0
	for i = 0; (offset > 0 || done < bytes) && i < iovCnt; i++ {
		if offset < iov[i].iov_len {
			len := min(iov[i].iov_len-offset, bytes-done)
			memcpy(unsafe.Pointer(uintptr(iov[i].iov_base)+uintptr(offset)),
				unsafe.Pointer(uintptr(buf)+uintptr(done)), len)
			done += len
			offset = 0
		} else {
			offset -= iov[i].iov_len
		}
	}
	return done
}

func iov_to_buf(iov []iovec, iovCnt uint64, offset uint64, buf unsafe.Pointer, bytes uint64) uint64 {
	var done uint64 = 0
	var i uint64 = 0
	for i = 0; (offset > 0 || done < bytes) && i < iovCnt; i++ {
		if offset < iov[i].iov_len {
			len := min(iov[i].iov_len-offset, bytes-done)
			memcpy(unsafe.Pointer(uintptr(buf)+uintptr(done)),
				unsafe.Pointer(uintptr(iov[i].iov_base)+uintptr(offset)), len)
			done += len
			offset = 0
		} else {
			offset -= iov[i].iov_len
		}
	}
	return done
}

func qiov_slice(qiov *QEMUIOVector, offset uint64, length uint64, head *uint64, tail *uint64, niov *int) []iovec {

	var iov, endIov []iovec

	iov = iov_skip_offset(qiov.iov, offset, head)
	endIov = iov_skip_offset(iov, *head+length, tail)

	if *tail > 0 {
		*tail = endIov[0].iov_len - *tail
		endIov = endIov[1:]
	}

	*niov = len(iov) - len(endIov)
	return iov
}

func iov_skip_offset(iov []iovec, offset uint64, remainingOffset *uint64) []iovec {
	idx := 0
	for offset > 0 && offset >= iov[idx].iov_len {
		offset -= iov[idx].iov_len
		idx++
	}
	*remainingOffset = offset

	return iov[idx:]
}

func qemu_iovec_destroy(qiov *QEMUIOVector) {
	memset(unsafe.Pointer(qiov), int(unsafe.Sizeof(*qiov)))
}

func qemu_iovec_init_extended(qiov *QEMUIOVector, headBuf unsafe.Pointer, headLen uint64,
	midQiov *QEMUIOVector, midOffset uint64, midLen uint64, tailBuf unsafe.Pointer, tailLen uint64) error {

	var midHead, midTail uint64
	var totalNiov, midNiov int
	var p []iovec
	var midIov []iovec

	if math.MaxUint64-headLen < midLen ||
		math.MaxUint64-headLen-midLen < tailLen {
		return ERR_EINVAL
	}

	if midLen > 0 {
		midIov = qiov_slice(midQiov, midOffset, midLen,
			&midHead, &midTail, &midNiov)
	}

	totalNiov = midNiov
	if headLen > 0 {
		totalNiov++
	}
	if tailLen > 0 {
		totalNiov++
	}

	if totalNiov > IOV_MAX {
		return ERR_EINVAL
	}

	if totalNiov == 1 {
		qemu_iovec_init_buf(qiov, nil, 0)
		p = qiov.iov[0:]
	} else {
		qiov.nalloc = totalNiov
		qiov.niov = qiov.nalloc
		qiov.size = headLen + midLen + tailLen
		qiov.iov = make([]iovec, qiov.niov)
		p = qiov.iov[:]
	}

	if headLen > 0 {
		p[0].iov_base = headBuf
		p[0].iov_len = headLen
		p = qiov.iov[1:]
	}

	if midNiov > 0 {
		memcpy(unsafe.Pointer(&p[0]), unsafe.Pointer(&midIov[0]), uint64(midNiov)*uint64(unsafe.Sizeof(p[0])))
		p[0].iov_base = unsafe.Pointer(uintptr(p[0].iov_base) + uintptr(midHead))
		p[0].iov_len -= midHead
		p[midNiov-1].iov_len -= midTail
		p = p[midNiov:]
	}

	if tailLen > 0 {
		p[0].iov_base = tailBuf
		p[0].iov_len = tailLen
	}

	return nil
}

func qemu_iovec_init_slice(qiov *QEMUIOVector /* out */, source *QEMUIOVector, offset uint64, length uint64) error {
	return qemu_iovec_init_extended(qiov, nil, 0, source, offset, length, nil, 0)
}

func qemu_iovec_subvec_niov(qiov *QEMUIOVector, offset uint64, length uint64) int {
	var head, tail uint64
	var niov int
	qiov_slice(qiov, offset, length, &head, &tail, &niov)
	return niov
}

func qemu_iovec_reset(qiov *QEMUIOVector) {
	qiov.niov = 0
	qiov.size = 0
}

func qemu_iovec_concat(dst *QEMUIOVector, src *QEMUIOVector, soffset uint64, sbytes uint64) {
	qemu_iovec_concat_iov(dst, src.iov, uint64(src.niov), soffset, sbytes)
}

func qemu_iovec_concat_iov(dst *QEMUIOVector, srcIov []iovec,
	srcCnt uint64, soffset uint64, sbytes uint64) uint64 {
	var i uint64
	var done uint64
	if sbytes == 0 {
		return 0
	}

	for i = 0; done < sbytes && i < srcCnt; i++ {
		if soffset < srcIov[i].iov_len {
			var len uint64 = min(srcIov[i].iov_len-soffset, sbytes-done)
			qemu_iovec_add(dst, unsafe.Pointer(uintptr(srcIov[i].iov_base)+uintptr(soffset)), len)
			done += len
			soffset = 0
		} else {
			soffset -= srcIov[i].iov_len
		}
	}

	return done
}

func qemu_iovec_memset(qiov *QEMUIOVector, offset uint64, fillc int, bytes uint64) uint64 {
	return iov_memset(qiov.iov, uint64(qiov.niov), offset, fillc, bytes)
}

func iov_memset(iov []iovec, iovCnt uint64, offset uint64, fillc int, bytes uint64) uint64 {
	var done uint64
	var i uint64

	for i = 0; (offset > 0 || done < bytes) && i < iovCnt; i++ {
		if offset < iov[i].iov_len {
			var length uint64 = min(iov[i].iov_len-offset, bytes-done)

			memset(unsafe.Pointer(uintptr(iov[i].iov_base)+uintptr(offset)), int(length))
			done += length
			offset = 0
		} else {
			offset -= iov[i].iov_len
		}
	}

	return done
}

func qemu_iovec_to_buf(qiov *QEMUIOVector, offset uint64, buf unsafe.Pointer, bytes uint64) uint64 {
	return iov_to_buf(qiov.iov, uint64(qiov.niov), offset, buf, bytes)
}

func qemu_iovec_from_buf(qiov *QEMUIOVector, offset uint64, buf unsafe.Pointer, bytes uint64) uint64 {
	return iov_from_buf(qiov.iov, uint64(qiov.niov), offset, buf, bytes)
}
