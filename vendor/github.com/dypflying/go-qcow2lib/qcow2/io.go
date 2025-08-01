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
	"encoding/binary"
	"math"
	"sync/atomic"
	"unsafe"
)

func bdrv_flush(bs *BlockDriverState) error {

	if bs.Drv == nil {
		return Err_NoDriverFound
	}

	atomic.AddUint64(&bs.InFlight, 1)
	defer atomic.AddUint64(&bs.InFlight, ^uint64(0))

	if bs.Drv.bdrv_flush != nil {
		return bs.Drv.bdrv_flush(bs)
	}
	if bs.Drv.bdrv_flush_to_os != nil {
		return bs.Drv.bdrv_flush_to_os(bs)
	}
	if bs.Drv.bdrv_flush_to_disk != nil {
		return bs.Drv.bdrv_flush_to_disk(bs)
	}

	return nil
}

func bdrv_pread(child *BdrvChild, offset uint64, buf unsafe.Pointer, bytes uint64) error {
	var qiov QEMUIOVector
	qemu_iovec_init_buf(&qiov, buf, bytes)
	return bdrv_preadv(child, offset, bytes, &qiov, 0)
}

func bdrv_pwrite(child *BdrvChild, offset uint64, buf unsafe.Pointer, bytes uint64) error {
	var qiov QEMUIOVector
	qemu_iovec_init_buf(&qiov, buf, bytes)
	return bdrv_pwritev(child, offset, bytes, &qiov, 0)
}

func bdrv_pwritev(child *BdrvChild, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error {
	return bdrv_pwritev_part(child, offset, bytes, qiov, 0, flags)
}

func bdrv_pwritev_part(child *BdrvChild, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	//check permission
	if child.perm&PERM_WRITABLE == 0 {
		return Err_NoWritePerm
	}
	bs := child.bs
	var pad BdrvRequestPadding
	var err error
	padded := false
	align := bs.RequestAlignment

	/* If the request is misaligned then we can't make it efficient */
	if flags&BDRV_REQ_NO_FALLBACK > 0 &&
		!is_aligned(int(offset|bytes), int(align)) {
		return Err_Misaligned
	}

	if bytes == 0 && !is_aligned(int(offset), int(align)) {
		return err
	}

	if flags&BDRV_REQ_ZERO_WRITE == 0 {
		if err = bdrv_pad_request(bs, &qiov, &qiovOffset, &offset, &bytes, &pad,
			&padded); err != nil {
			return err
		}
	}

	atomic.AddUint64(&bs.InFlight, 1)

	if flags&BDRV_REQ_ZERO_WRITE > 0 {
		Assert(!padded)
		err = bdrv_do_zero_pwritev(child, offset, bytes, flags)
		goto out
	}

	if padded {
		//if the requests is queued already, won't use this feature
		//bdrv_make_request_serialising(&req, align)
		overlapOffset := offset & uint64(^(align - 1))
		overlapBytes := round_up(offset+bytes, uint64(align)) - overlapOffset
		bdrv_padding_rmw_read(child, overlapOffset, overlapBytes, &pad, false)
	}

	err = bdrv_aligned_pwritev(child, offset, bytes, uint64(align),
		qiov, qiovOffset, flags)

	bdrv_padding_destroy(&pad)

out:
	atomic.AddUint64(&bs.InFlight, ^uint64(0))
	return err
}

func bdrv_pad_request(bs *BlockDriverState, qiov **QEMUIOVector, qiovOffset *uint64,
	offset *uint64, bytes *uint64, pad *BdrvRequestPadding, padded *bool) error {

	var err error

	if !bdrv_init_padding(bs, *offset, *bytes, pad) {
		if padded != nil {
			*padded = false
		}
		return nil
	}

	if err = qemu_iovec_init_extended(&pad.LocalQiov, unsafe.Pointer(&pad.Buf[0]), pad.Head,
		*qiov, *qiovOffset, *bytes, unsafe.Pointer(&pad.Buf[pad.BufLen-pad.Tail]), pad.Tail); err != nil {
		bdrv_padding_destroy(pad)
		return err
	}

	*bytes += pad.Head + pad.Tail
	*offset -= pad.Head
	*qiov = &pad.LocalQiov
	*qiovOffset = 0
	if padded != nil {
		*padded = true
	}

	return nil
}

func bdrv_do_zero_pwritev(child *BdrvChild, offset uint64, bytes uint64, flags BdrvRequestFlags) error {

	bs := child.bs
	var localQiov QEMUIOVector
	align := uint64(bs.RequestAlignment)
	var err error
	var padding bool
	var pad BdrvRequestPadding

	padding = bdrv_init_padding(bs, offset, bytes, &pad)

	if padding {
		//bdrv_make_request_serialising(req, align);
		overlapOffset := offset & ^(align - 1)
		overlapBytes := round_up(offset+bytes, align) - overlapOffset
		bdrv_padding_rmw_read(child, overlapOffset, overlapBytes, &pad, true)

		if pad.Head > 0 || pad.MergeReads {
			alignedOffset := offset & ^(align - 1)
			var writeBytes uint64
			if pad.MergeReads {
				writeBytes = pad.BufLen
			} else {
				writeBytes = align
			}

			qemu_iovec_init_buf(&localQiov, unsafe.Pointer(&pad.Buf[0]), writeBytes)
			if err = bdrv_aligned_pwritev(child, alignedOffset, writeBytes,
				align, &localQiov, 0, flags & ^BDRV_REQ_ZERO_WRITE); err != nil || pad.MergeReads {
				/* Error or all work is done */
				goto out
			}
			offset += writeBytes - pad.Head
			bytes -= writeBytes - pad.Head
		}
	}

	Assert(bytes == 0 || (offset&(align-1)) == 0)
	if bytes >= align {
		/* Write the aligned part in the middle. */
		alignedBytes := bytes & ^(align - 1)
		if err = bdrv_aligned_pwritev(child, offset, alignedBytes, align,
			nil, 0, flags); err != nil {
			goto out
		}
		bytes -= alignedBytes
		offset += alignedBytes
	}
	Assert(bytes == 0 || (offset&(align-1)) == 0)
	if bytes > 0 {
		Assert(align == pad.Tail+bytes)
		qemu_iovec_init_buf(&localQiov, unsafe.Pointer(&pad.TailBuf[0]), align)
		err = bdrv_aligned_pwritev(child, offset, align, align,
			&localQiov, 0, flags & ^BDRV_REQ_ZERO_WRITE)
	}

out:
	bdrv_padding_destroy(&pad)
	return err
}

func bdrv_padding_rmw_read(child *BdrvChild, overlapOffset uint64, overlapBytes uint64, pad *BdrvRequestPadding, zeroMiddle bool) error {

	var localQiov QEMUIOVector
	bs := child.bs
	align := bs.RequestAlignment
	var err error
	var bytes uint64

	if pad.Head > 0 || pad.MergeReads {

		if pad.MergeReads {
			bytes = pad.BufLen
		} else {
			bytes = uint64(align)
		}

		qemu_iovec_init_buf(&localQiov, unsafe.Pointer(&pad.Buf[0]), bytes)

		if err = bdrv_aligned_preadv(child, overlapOffset, bytes,
			align, &localQiov, 0, 0); err != nil {
			return err
		}

		if pad.MergeReads {
			goto zero_mem
		}
	}

	if pad.Tail > 0 {

		qemu_iovec_init_buf(&localQiov, unsafe.Pointer(&pad.TailBuf[0]), uint64(align))

		if err = bdrv_aligned_preadv(child, overlapOffset+overlapBytes-uint64(align),
			uint64(align), align, &localQiov, 0, 0); err != nil {
			return err
		}
	}

zero_mem:
	if zeroMiddle {
		memset(unsafe.Pointer(&pad.Buf[pad.Head]), int(pad.BufLen-pad.Head-pad.Tail))
	}
	return err
}

func bdrv_aligned_pwritev(child *BdrvChild, offset uint64, bytes uint64,
	align uint64, qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	bs := child.bs
	var err error
	bytesRemaining := bytes

	Assert((offset & (align - 1)) == 0)
	Assert((bytes & (align - 1)) == 0)
	maxTransfer := uint64(align_down(bs.MaxTransfer, uint32(align))) //64MiB

	if flags&BDRV_REQ_ZERO_WRITE > 0 {
		err = bdrv_do_pwrite_zeroes(bs, offset, bytes, flags)
	} else if flags&BDRV_REQ_WRITE_COMPRESSED > 0 {
		//do nothing
	} else if bytes <= maxTransfer {
		err = bdrv_driver_pwritev(bs, offset, bytes, qiov, qiovOffset, flags)
	} else {

		for bytesRemaining > 0 {
			num := min(bytesRemaining, maxTransfer)
			localFlags := flags
			Assert(num > 0)
			if num < bytesRemaining && (flags&BDRV_REQ_FUA > 0 && bs.SupportedWriteFlags&BDRV_REQ_FUA == 0) {
				/* If FUA is going to be emulated by flush, we only
				 * need to flush on the last iteration */
				localFlags &= ^BDRV_REQ_FUA
			}
			if err = bdrv_driver_pwritev(bs, offset+bytes-bytesRemaining,
				num, qiov, qiovOffset+bytes-bytesRemaining, localFlags); err != nil {
				break
			}
			bytesRemaining -= num
		}
	}

	//we don't have to resize it
	//Bdrv_co_write_req_finish(child, offset, bytes, req, ret)

	return err
}

func bdrv_aligned_preadv(child *BdrvChild, offset uint64,
	bytes uint64, align uint32, qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	//check permission
	if child.perm&PERM_READABLE == 0 {
		return Err_NoReadPerm
	}
	bs := child.bs
	var totalBytes, maxBytes uint64
	var err error
	var bytesRemaining uint64 = bytes
	var maxTransfer uint64
	var ret uint64

	//TODO1: not aligned when writting zeroes
	Assert((offset & uint64(align-1)) == 0)
	Assert((bytes & uint64(align-1)) == 0)
	Assert((bs.OpenFlags & BDRV_O_NO_IO) == 0)
	maxTransfer = align_down(uint64(bs.MaxTransfer), uint64(align))
	Assert((int(flags) & ^(BDRV_REQ_COPY_ON_READ | BDRV_REQ_PREFETCH)) == 0)

	if flags&BDRV_REQ_COPY_ON_READ > 0 {
		var pnum uint64

		/* The flag BDRV_REQ_COPY_ON_READ has reached its addressee */
		flags &= ^BDRV_REQ_COPY_ON_READ

		if ret, err = bdrv_is_allocated(bs, offset, bytes, &pnum); err != nil {
			goto out
		}

		if ret == 0 || pnum != bytes {
			err = bdrv_do_copy_on_readv(child, offset, bytes,
				qiov, qiovOffset, flags)
			goto out
		} else if flags&BDRV_REQ_PREFETCH > 0 {
			goto out
		}
	}

	/* Forward the request to the BlockDriver, possibly fragmenting it */
	if totalBytes, err = bdrv_getlength(bs); err != nil {
		goto out
	}

	Assert((uint64(flags) & ^bs.SupportedReadFlags) == 0)
	maxBytes = round_up(max(0, totalBytes-offset), uint64(align))
	if bytes <= maxBytes && bytes <= maxTransfer {
		err = bdrv_driver_preadv(bs, offset, bytes, qiov, qiovOffset, flags)
		goto out
	}

	for bytesRemaining > 0 {
		var num uint64

		if maxBytes > 0 {
			num = min(bytesRemaining, maxBytes, maxTransfer)
			Assert(num > 0)
			err = bdrv_driver_preadv(bs, offset+bytes-bytesRemaining,
				num, qiov, qiovOffset+bytes-bytesRemaining, flags)
			maxBytes -= num
		} else {
			num = bytesRemaining
			qemu_iovec_memset(qiov, qiovOffset+bytes-bytesRemaining, 0, bytesRemaining)
		}
		if err != nil {
			goto out
		}
		bytesRemaining -= num
	}

out:
	return err
}

func bdrv_padding_destroy(pad *BdrvRequestPadding) {
	if pad.Buf != nil {
		qemu_iovec_destroy(&pad.LocalQiov)
	}
	memset(unsafe.Pointer(pad), int(unsafe.Sizeof(*pad)))
}

func bdrv_init_padding(bs *BlockDriverState, offset uint64, bytes uint64, pad *BdrvRequestPadding) bool {

	align := uint64(bs.RequestAlignment)
	var sum uint64

	memset(unsafe.Pointer(pad), int(unsafe.Sizeof(*pad)))

	pad.Head = offset & (align - 1)
	pad.Tail = ((offset + bytes) & (align - 1))
	if pad.Tail > 0 {
		pad.Tail = align - pad.Tail
	}

	if pad.Head == 0 && pad.Tail == 0 {
		return false
	}

	Assert(bytes > 0)
	sum = pad.Head + bytes + pad.Tail
	if sum > align && pad.Head > 0 && pad.Tail > 0 {
		pad.BufLen = 2 * align
	} else {
		pad.BufLen = align
	}
	//In C implementation, this allocate a memory range align to the size, by default it is 4k
	pad.Buf = make([]uint8, pad.BufLen)
	pad.MergeReads = (sum == pad.BufLen)
	if pad.Tail > 0 {
		pad.TailBuf = pad.Buf[pad.BufLen-align:]
	}

	return true
}

func bdrv_do_pwrite_zeroes(bs *BlockDriverState, offset uint64, bytes uint64, flags BdrvRequestFlags) error {
	var (
		qiov      QEMUIOVector
		buf       unsafe.Pointer //[]byte
		err       error
		needFlush bool   = false
		head      uint64 = 0
		tail      uint64 = 0
	)
	drv := bs.Drv
	maxWriteZeroes := Max_WRITE_ZEROS
	alignment := Max_WRITE_ZEROS
	maxTransfer := Max_WRITE_ZEROS

	if flags&BdrvRequestFlags(^bs.SupportedZeroFlags)&BDRV_REQ_NO_FALLBACK > 0 {
		return ERR_ENOTSUP
	}

	head = offset % alignment
	tail = (offset + bytes) % alignment
	maxWriteZeroes = align_down(maxWriteZeroes, alignment)

	for bytes > 0 && err == nil {
		num := bytes
		if head > 0 {
			num = min(bytes, maxTransfer, alignment-head)
			head = (head + num) % alignment
			Assert(num < maxWriteZeroes)
		} else if tail > 0 && num > alignment {
			/* Shorten the request to the last aligned sector.  */
			num -= tail
		}

		/* limit request size */
		if num > maxWriteZeroes {
			num = maxWriteZeroes
		}

		err = ERR_ENOTSUP
		/* First try the efficient write zeroes operation */
		if drv.bdrv_pwrite_zeroes != nil {
			err = drv.bdrv_pwrite_zeroes(bs, offset, num,
				flags&BdrvRequestFlags(bs.SupportedZeroFlags))
			if err != ERR_ENOTSUP && flags&BDRV_REQ_FUA > 0 &&
				bs.SupportedZeroFlags&BDRV_REQ_FUA == 0 {
				needFlush = true
			}
		} else {
			Assert(bs.SupportedZeroFlags == 0)
		}

		if err == ERR_ENOTSUP && flags&BDRV_REQ_NO_FALLBACK == 0 {
			/* Fall back to bounce buffer if write zeroes is unsupported */
			var writeFlags BdrvRequestFlags = flags & ^BDRV_REQ_ZERO_WRITE
			if flags&BDRV_REQ_FUA > 0 && bs.SupportedWriteFlags&BDRV_REQ_FUA == 0 {
				/* No need for bdrv_driver_pwrite() to do a fallback
				 * flush on each chunk; use just one at the end */
				writeFlags &= ^BDRV_REQ_FUA
				needFlush = true
			}
			num = min(num, maxTransfer)
			if buf == nil {
				p := make([]byte, num)
				buf = unsafe.Pointer(&p[0])
			}
			qemu_iovec_init_buf(&qiov, buf, num)

			err = bdrv_driver_pwritev(bs, offset, num, &qiov, 0, writeFlags)
			if num < maxTransfer {
				buf = nil
			}

		}
		offset += num
		bytes -= num

	}

	if err == nil && needFlush {
		err = bdrv_flush(bs)
	}
	return err
}

// do write the buffer to disk
func bdrv_driver_pwritev(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	var localQiov QEMUIOVector
	var err error

	drv := bs.Drv

	Assert((flags & ^BDRV_REQ_MASK) == 0)
	Assert((flags & BDRV_REQ_NO_FALLBACK) == 0)

	if drv == nil {
		return Err_NoDriverFound
	}
	if drv.bdrv_pwritev_part != nil {
		err = drv.bdrv_pwritev_part(bs, offset, bytes, qiov, qiovOffset, flags&BdrvRequestFlags(bs.SupportedWriteFlags))
		flags &= BdrvRequestFlags(^bs.SupportedWriteFlags)
		goto out
	}

	if qiovOffset > 0 || bytes != qiov.size {
		qemu_iovec_init_slice(&localQiov, qiov, qiovOffset, bytes)
		qiov = &localQiov
	}

	if drv.bdrv_pwritev != nil {
		err = drv.bdrv_pwritev(bs, offset, bytes, qiov,
			flags&BdrvRequestFlags(bs.SupportedWriteFlags))
		flags &= BdrvRequestFlags(^bs.SupportedWriteFlags)
		goto out
	}

	Assert(false)
out:

	if err == nil && flags&BDRV_REQ_FUA > 0 {
		err = bdrv_flush(bs)
	}
	if qiov == &localQiov {
		qemu_iovec_destroy(&localQiov)
	}
	return err
}

func bdrv_pwrite_zeroes(child *BdrvChild, offset uint64, bytes uint64, flags BdrvRequestFlags) error {

	if child.bs.OpenFlags&BDRV_O_UNMAP == 0 {
		flags &= ^BDRV_REQ_MAY_UNMAP
	}
	return bdrv_pwritev(child, offset, bytes, nil, BDRV_REQ_ZERO_WRITE|flags)
}

func bdrv_preadv(child *BdrvChild, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error {
	return bdrv_preadv_part(child, offset, bytes, qiov, 0, flags)
}

func bdrv_preadv_part(child *BdrvChild, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {
	bs := child.bs
	var pad BdrvRequestPadding
	var err error

	if bytes == 0 && !is_aligned(offset, uint64(bs.RequestAlignment)) {
		return nil
	}

	atomic.AddUint64(&bs.InFlight, 1)

	if err = bdrv_pad_request(bs, &qiov, &qiovOffset, &offset, &bytes, &pad, nil); err != nil {
		goto fail
	}
	err = bdrv_aligned_preadv(child, offset, bytes, bs.RequestAlignment, qiov, qiovOffset, flags)

	bdrv_padding_destroy(&pad)

fail:
	atomic.AddUint64(&bs.InFlight, ^uint64(0))
	return err
}

func bdrv_is_allocated(bs *BlockDriverState, offset uint64, bytes uint64, pnum *uint64) (uint64, error) {

	var err error
	var dummy uint64
	var ret uint64
	if pnum != nil {
		ret, err = bdrv_common_block_status_above(bs, bs, true, false, offset,
			bytes, pnum, nil, nil, nil)
	} else {
		ret, err = bdrv_common_block_status_above(bs, bs, true, false, offset,
			bytes, &dummy, nil,
			nil, nil)
	}
	if err != nil {
		return 0, err
	}
	if ret&BDRV_BLOCK_ALLOCATED > 0 {
		ret = 1
	}
	return ret, nil
}

func bdrv_driver_preadv(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	var localQiov QEMUIOVector
	var err error
	drv := bs.Drv

	if drv == nil {
		return Err_NoDriverFound
	}
	Assert((flags & ^BDRV_REQ_MASK) == 0)
	Assert((flags & BDRV_REQ_NO_FALLBACK) == 0)

	if drv.bdrv_preadv_part != nil {
		return drv.bdrv_preadv_part(bs, offset, bytes, qiov, qiovOffset, flags)
	}
	if qiovOffset > 0 || bytes != qiov.size {
		qemu_iovec_init_slice(&localQiov, qiov, qiovOffset, bytes)
		qiov = &localQiov
	}
	if drv.bdrv_preadv != nil {
		err = drv.bdrv_preadv(bs, offset, bytes, qiov, flags)
		goto out
	}
	Assert(false)
out:
	if qiov == &localQiov {
		qemu_iovec_destroy(&localQiov)
	}
	return err
}

func bdrv_do_copy_on_readv(child *BdrvChild, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	bs := child.bs
	var bounceBuffer []byte
	drv := bs.Drv
	var clusterOffset, clusterBytes, skipBytes uint64

	var err error
	maxTransfer := bs.MaxTransfer
	var progress uint64 = 0
	var skipWrite bool
	var ret uint64

	skipWrite = bs.OpenFlags&BDRV_O_INACTIVE > 0

	bdrv_round_to_clusters(bs, offset, bytes, &clusterOffset, &clusterBytes)

	skipBytes = offset - clusterOffset

	for clusterBytes > 0 {
		var pnum uint64
		if skipWrite {
			ret = 1 /* "already allocated", so nothing will be copied */
			pnum = min(clusterBytes, uint64(maxTransfer))
		} else {
			ret, err = bdrv_is_allocated(bs, clusterOffset,
				min(clusterBytes, uint64(maxTransfer)), &pnum)
			if err != nil {
				pnum = min(clusterBytes, uint64(maxTransfer))
			}

			/* Stop at EOF if the image ends in the middle of the cluster */
			if ret == 0 && pnum == 0 {
				Assert(progress >= bytes)
				break
			}
			Assert(skipBytes < pnum)
		}

		if ret <= 0 {
			var localQiov QEMUIOVector

			/* Must copy-on-read; use the bounce buffer */
			pnum = min(pnum, MAX_BOUNCE_BUFFER)
			if bounceBuffer == nil {
				var max_we_need = max(pnum, clusterBytes-pnum)
				var max_allowed = min(uint64(maxTransfer), MAX_BOUNCE_BUFFER)
				var bounce_buffer_len = min(max_we_need, max_allowed)
				bounceBuffer = make([]byte, bounce_buffer_len)
			}

			qemu_iovec_init_buf(&localQiov, unsafe.Pointer(&bounceBuffer[0]), pnum)

			if err = bdrv_driver_preadv(bs, clusterOffset, pnum, &localQiov, 0, 0); err != nil {
				goto err
			}
			if drv.bdrv_pwrite_zeroes != nil &&
				buffer_is_zero(bounceBuffer, pnum) {
				err = bdrv_do_pwrite_zeroes(bs, clusterOffset, pnum,
					BDRV_REQ_WRITE_UNCHANGED)
			} else {
				err = bdrv_driver_pwritev(bs, clusterOffset, pnum,
					&localQiov, 0, BDRV_REQ_WRITE_UNCHANGED)
			}
			if err != nil {
				goto err
			}

			if flags&BDRV_REQ_PREFETCH == 0 {
				qemu_iovec_to_buf(qiov, qiovOffset+progress,
					unsafe.Pointer(&bounceBuffer[skipBytes]), min(pnum-skipBytes, bytes-progress))
			}
		} else if flags&BDRV_REQ_PREFETCH == 0 {
			/* Read directly into the destination */
			if err = bdrv_driver_preadv(bs, offset+progress, min(pnum-skipBytes, bytes-progress),
				qiov, qiovOffset+progress, 0); err != nil {
				goto err
			}
		}
		clusterOffset += pnum
		clusterBytes -= pnum
		progress += pnum - skipBytes
		skipBytes = 0

	} //end for
	err = nil
err:
	return err
}

func bdrv_block_status_above(bs *BlockDriverState, base *BlockDriverState,
	offset uint64, bytes uint64, pnum *uint64, tmap *uint64, file **BlockDriverState) (uint64, error) {
	return bdrv_common_block_status_above(bs, base, false, true, offset, bytes,
		pnum, tmap, file, nil)
}

func bdrv_common_block_status_above(bs *BlockDriverState, base *BlockDriverState,
	includeBase bool, wantZero bool, offset uint64, bytes uint64,
	pnum *uint64, tmap *uint64, file **BlockDriverState, depth *int) (uint64, error) {

	var ret uint64
	var err error
	var p *BlockDriverState
	var eof uint64 = 0
	var dummy int

	Assert(!includeBase || base != nil)
	if depth == nil {
		depth = &dummy
	}
	*depth = 0

	if !includeBase && bs == base {
		*pnum = bytes
		return 0, nil
	}

	ret, err = bdrv_block_status(bs, wantZero, offset, bytes, pnum, tmap, file)
	*depth++
	if err != nil || *pnum == 0 || ret&BDRV_BLOCK_ALLOCATED > 0 || bs == base {
		return ret, err
	}
	if ret&BDRV_BLOCK_EOF > 0 {
		eof = offset + *pnum
	}
	Assert(*pnum <= bytes)
	bytes = *pnum

	for p = bdrv_filter_or_cow_bs(bs); includeBase || p != base; p = bdrv_filter_or_cow_bs(p) {
		ret, err = bdrv_block_status(p, wantZero, offset, bytes, pnum, tmap, file)
		*depth++
		if err != nil {
			return 0, err
		}
		if *pnum == 0 {
			Assert(ret&BDRV_BLOCK_EOF > 0)
			*pnum = bytes
			if file != nil {
				*file = p
			}
			ret = BDRV_BLOCK_ZERO | BDRV_BLOCK_ALLOCATED
			break
		}
		if ret&BDRV_BLOCK_ALLOCATED > 0 {
			ret &= ^uint64(BDRV_BLOCK_EOF)
			break
		}
		if p == base {
			Assert(includeBase)
			break
		}
		Assert(*pnum <= bytes)
		bytes = *pnum

	}
	if offset+*pnum == eof {
		ret |= BDRV_BLOCK_EOF
	}

	return ret, nil
}

func bdrv_is_zero_fast(bs *BlockDriverState, offset uint64, bytes uint64) (bool, error) {
	var ret uint64
	var err error
	pnum := bytes

	if bytes == 0 {
		return true, nil
	}

	if ret, err = bdrv_common_block_status_above(bs, nil, false, false, offset,
		bytes, &pnum, nil, nil, nil); err != nil {
		return false, err
	}

	if pnum == bytes && ret&BDRV_BLOCK_ZERO > 0 {
		return true, nil
	}
	return false, nil
}

func bdrv_block_status(bs *BlockDriverState, wantZero bool, offset uint64, bytes uint64,
	pnum *uint64, tmap *uint64, file **BlockDriverState) (uint64, error) {

	var total_size uint64
	var n uint64 /* bytes */
	var ret uint64
	var err error
	var local_map uint64
	var local_file *BlockDriverState
	var aligned_offset, aligned_bytes uint64
	var align uint32

	Assert(pnum != nil)
	*pnum = 0
	if total_size, err = bdrv_getlength(bs); err != nil {
		goto early_out
	}

	if offset >= total_size {
		ret = BDRV_BLOCK_EOF
		goto early_out
	}
	if bytes == 0 {
		ret = 0
		goto early_out
	}

	n = total_size - offset
	if n < bytes {
		bytes = n
	}

	atomic.AddUint64(&bs.InFlight, uint64(0))

	/* Round out to request_alignment boundaries */
	align = bs.RequestAlignment
	aligned_offset = align_down(offset, uint64(align))
	aligned_bytes = round_up(offset+bytes, uint64(align)) - aligned_offset

	Assert(bs.Drv != nil)
	if bs.Drv.bdrv_block_status != nil {
		if ret, err = bs.Drv.bdrv_block_status(bs, wantZero, aligned_offset,
			aligned_bytes, pnum, &local_map,
			&local_file); err != nil {
			*pnum = 0
			goto out
		}
	}

	*pnum -= offset - aligned_offset
	if *pnum > bytes {
		*pnum = bytes
	}
	if ret&BDRV_BLOCK_OFFSET_VALID > 0 {
		local_map += offset - aligned_offset
	}

	if ret&BDRV_BLOCK_RAW > 0 {
		ret, err = bdrv_block_status(local_file, wantZero, local_map,
			*pnum, pnum, &local_map, &local_file)
		goto out
	}

	if ret&(BDRV_BLOCK_DATA|BDRV_BLOCK_ZERO) > 0 {
		ret |= BDRV_BLOCK_ALLOCATED
	} else if bs.Drv.SupportBacking {
		cow_bs := bdrv_cow_bs(bs)

		if cow_bs == nil {
			ret |= BDRV_BLOCK_ZERO
		} else if wantZero {
			size2, err := bdrv_getlength(cow_bs)

			if err == nil && offset >= size2 {
				ret |= BDRV_BLOCK_ZERO
			}
		}
	}

	if wantZero && ret&BDRV_BLOCK_RECURSE > 0 &&
		local_file != nil && local_file != bs &&
		ret&BDRV_BLOCK_DATA > 0 && ret&BDRV_BLOCK_ZERO == 0 &&
		ret&BDRV_BLOCK_OFFSET_VALID > 0 {
		var file_pnum uint64
		var ret2 uint64

		ret2, err = bdrv_block_status(local_file, wantZero, local_map,
			*pnum, &file_pnum, nil, nil)
		if err == nil {
			if ret2&BDRV_BLOCK_EOF > 0 &&
				(file_pnum == 0 || ret2&BDRV_BLOCK_ZERO > 0) {
				ret |= BDRV_BLOCK_ZERO
			} else {
				/* Limit request to the range reported by the protocol driver */
				*pnum = file_pnum
				ret |= (ret2 & BDRV_BLOCK_ZERO)
			}
		}
	}
out:
	atomic.AddUint64(&bs.InFlight, ^uint64(0))
	if err == nil && offset+*pnum == total_size {
		ret |= BDRV_BLOCK_EOF
	}
early_out:
	if file != nil {
		*file = local_file
	}
	if tmap != nil {
		*tmap = local_map
	}
	return ret, err

}

func bdrv_round_to_clusters(bs *BlockDriverState, offset uint64, bytes uint64,
	clusterOffset *uint64, clusterBytes *uint64) {

	if bs == nil || bs.opaque == nil {
		*clusterOffset = offset
		*clusterBytes = bytes
	}
	*clusterOffset = align_down(offset, uint64(DEFAULT_CLUSTER_SIZE))
	*clusterBytes = align_up(offset-*clusterOffset+bytes, uint64(DEFAULT_CLUSTER_SIZE))
}

func bdrv_open_child(filename string, format string, options map[string]any, flags int) (*BdrvChild, error) {
	var bs *BlockDriverState
	var err error
	var drv *BlockDriver = get_driver(format)

	if bs, err = drv.bdrv_open(filename, options, flags); err != nil {
		return nil, err
	}
	bs.Drv = drv

	child := &BdrvChild{
		bs: bs,
	}
	return child, nil
}

func bdrv_set_perm(child *BdrvChild, perm uint8) {
	child.perm = perm
}

func bdrv_link_child(parent *BlockDriverState, child *BdrvChild, childName string) {
	if parent == nil || child == nil || child.bs == nil {
		return
	}
	child.name = childName
	parent.current = child
	child.bs.InheritsFrom = parent
}

func bdrv_link_backing(parent *BlockDriverState, child *BdrvChild, childName string) {
	if parent == nil || child == nil || child.bs == nil {
		return
	}
	child.name = childName
	parent.backing = child
	child.bs.InheritsFrom = parent
}

func bdrv_opt_mem_align(bs *BlockDriverState) uint64 {
	return uint64(DEFAULT_ALIGNMENT)
}

func bdrv_getlength(bs *BlockDriverState) (uint64, error) {

	if bs.Drv != nil && bs.Drv.bdrv_getlength != nil {
		return bs.Drv.bdrv_getlength(bs)
	}
	var ret uint64 = bdrv_nb_sectors(bs)
	if ret > math.MaxInt64/BDRV_SECTOR_SIZE {
		return 0, ERR_E2BIG
	}
	return ret * BDRV_SECTOR_SIZE, nil
}

func bdrv_nb_sectors(bs *BlockDriverState) uint64 {
	return bs.TotalSectors
}

func bdrv_cow_bs(bs *BlockDriverState) *BlockDriverState {
	return child_bs(bdrv_cow_child(bs))
}

func child_bs(child *BdrvChild) *BlockDriverState {
	if child != nil {
		return child.bs
	} else {
		return nil
	}
}

func bdrv_cow_child(bs *BlockDriverState) *BdrvChild {
	if bs == nil || bs.Drv == nil {
		return nil
	}
	return bs.backing
}

func bdrv_create(filename string, opts map[string]any) error {
	var format string
	if val, ok := opts[OPT_FMT]; !ok {
		return Err_IncompleteParameters
	} else {
		format = val.(string)
	}
	drv := get_driver(format)
	if drv == nil || drv.bdrv_create == nil {
		return Err_NoDriverFound
	}
	return drv.bdrv_create(filename, opts)
}

func bdrv_filter_or_cow_bs(bs *BlockDriverState) *BlockDriverState {
	return child_bs(bdrv_filter_or_cow_child(bs))
}

func bdrv_filter_or_cow_child(bs *BlockDriverState) *BdrvChild {
	return bdrv_cow_child(bs)
}

func bdrv_close(bs *BlockDriverState) {
	bdrv_flush(bs)
	if bs.Drv != nil {
		if bs.Drv.bdrv_close != nil {
			bs.Drv.bdrv_close(bs)
		}
		bs.Drv = nil
	}
}

// read object from the file, the object's size must be obtainable, for debugging purpose
func bdrv_direct_pread(child *BdrvChild, offset int64, object any, size int64) (int, error) {

	var n int
	var err error
	var buffer bytes.Buffer

	s := child.bs.opaque.(*BDRVRawState)
	if child == nil || s.File == nil {
		return -1, Err_NullObject
	}

	var buf = make([]byte, size)
	if n, err = s.File.ReadAt(buf, offset); err != nil {
		return -1, err
	}
	buffer.Write(buf)
	binary.Read(&buffer, binary.BigEndian, object)
	return n, nil
}

// Write the object to the file in the bigendian manner, return -1 if error occurs, for debugging purpose
func bdrv_direct_pwrite(child *BdrvChild, offset int64, object any, size int64) (int, error) {

	var n int
	var err error
	var buffer bytes.Buffer
	s := child.bs.opaque.(*BDRVRawState)
	if child == nil || s.File == nil {
		return -1, Err_NullObject
	}
	binary.Write(&buffer, binary.BigEndian, object)
	buffer.Truncate(int(size))
	if n, err = s.File.WriteAt(buffer.Bytes(), offset); err != nil {
		return -1, err
	}
	return n, nil
}

func bdrv_pdiscard(child *BdrvChild, offset uint64, bytes uint64) error {

	var head, tail, align uint64
	bs := child.bs
	var err error

	if bs == nil || bs.Drv == nil {
		return Err_NoDriverFound
	}

	if bs.OpenFlags&BDRV_O_UNMAP == 0 {
		return nil
	}

	//depends on driver to execute the discard
	if bs.Drv.bdrv_pdiscard == nil {
		return Err_NoDriverFound
	}

	align = uint64(max(bs.RequestAlignment, bs.PdiscardAlignment))
	head = offset % align
	tail = (offset + bytes) % align

	atomic.AddUint64(&bs.InFlight, 1)

	for bytes > 0 {
		num := bytes

		if head > 0 {
			/* Make small requests to get to alignment boundaries. */
			num = min(bytes, align-head)
			if !is_aligned(num, uint64(bs.RequestAlignment)) {
				num %= uint64(bs.RequestAlignment)
			}
			head = (head + num) % align
		} else if tail > 0 {
			if num > align {
				num -= tail
			} else if !is_aligned(tail, uint64(bs.RequestAlignment)) &&
				tail > uint64(bs.RequestAlignment) {
				tail = tail % uint64(bs.RequestAlignment)
				num -= tail
			}
		}
		if bs.Drv.bdrv_pdiscard != nil {
			err = bs.Drv.bdrv_pdiscard(bs, offset, num)
		}
		if err != nil && err != ERR_ENOTSUP {
			goto out
		}

		offset += num
		bytes -= num
	}
	err = nil
out:
	atomic.AddUint64(&bs.InFlight, ^uint64(0))
	return err
}
