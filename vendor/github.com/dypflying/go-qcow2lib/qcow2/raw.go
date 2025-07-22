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
	"context"
	"fmt"
	"os"
)

func newRawDriver() *BlockDriver {
	return &BlockDriver{
		FormatName:           "raw",
		IsFormat:             true,
		SupportBacking:       false,
		bdrv_create:          raw_create,
		bdrv_open:            raw_open,
		bdrv_close:           raw_close,
		bdrv_flush_to_disk:   raw_flush_to_disk,
		bdrv_getlength:       raw_getlength,
		bdrv_preadv:          raw_preadv,
		bdrv_pwritev:         raw_pwritev,
		bdrv_block_status:    raw_block_status,
		bdrv_pwrite_zeroes:   raw_pwrite_zeroes,
		bdrv_copy_range_from: raw_copy_range_from,
		bdrv_copy_range_to:   raw_copy_range_to,
	}
}

func openflag2PosixFlag(flag int) int {

	var posixFlag int
	if (flag & BDRV_O_CREATE) > 0 {
		posixFlag |= os.O_CREATE
	}
	if (flag & BDRV_O_RDWR) > 0 {
		posixFlag |= os.O_RDWR
	}
	if (flag & BDRV_O_NOCACHE) > 0 {
		posixFlag |= os.O_SYNC
	}
	return posixFlag
}
func raw_create(filename string, options map[string]any) error {

	var file *os.File
	var err error

	//check file name
	if filename == "" {
		return Err_IncompleteParameters
	}

	file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("failed to open %s, err: %v", filename, err)
	}
	defer file.Close()
	return nil
}

func raw_open(filename string, options map[string]any, flags int) (*BlockDriverState, error) {

	var file *os.File
	var err error

	//check file name
	if filename == "" {
		return nil, Err_IncompleteParameters
	}

	//open the qcow file
	if file, err = os.OpenFile(filename, openflag2PosixFlag(flags), os.FileMode(0777)); err != nil {
		return nil, fmt.Errorf("failed to open %s, err: %v", filename, err)
	}

	//initiate the BlockDriverState struct
	bs := &BlockDriverState{
		filename: filename,
		opaque: &BDRVRawState{
			File: file,
		},
		current:             nil,
		backing:             nil,
		options:             make(map[string]any),
		SupportedWriteFlags: 0,
		RequestAlignment:    DEFAULT_ALIGNMENT,
		MaxTransfer:         DEFAULT_MAX_TRANSFER,
		OpenFlags:           flags,
	}

	return bs, nil
}

func raw_close(bs *BlockDriverState) {
	s := bs.opaque.(*BDRVRawState)
	if s == nil || s.File == nil {
		return
	}
	s.File.Close()
}

func raw_getlength(bs *BlockDriverState) (uint64, error) {
	var err error
	var info os.FileInfo
	s := bs.opaque.(*BDRVRawState)

	if s == nil || s.File == nil {
		return 0, Err_NullObject
	}
	if info, err = s.File.Stat(); err != nil {
		return 0, err
	}
	return uint64(info.Size()), nil
}

func raw_preadv(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error {
	return raw_preadv_part(bs, offset, bytes, qiov, 0, flags)
}

func raw_preadv_part(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	var localQiov QEMUIOVector
	var err error
	s := bs.opaque.(*BDRVRawState)
	if s == nil || s.File == nil {
		return Err_NullObject
	}

	if qiovOffset > 0 || bytes != qiov.size {
		qemu_iovec_init_slice(&localQiov, qiov, qiovOffset, bytes)
		qiov = &localQiov
	}

	//call physical read for the qiov buffer
	ctx := context.Background()
	_, err = preadv(ctx, s.File, qiov.iov, qiov.niov, offset)
	if qiov == &localQiov {
		qemu_iovec_destroy(&localQiov)
	}
	return err
}

func raw_pwritev(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, flags BdrvRequestFlags) error {
	return raw_pwritev_part(bs, offset, bytes, qiov, 0, flags)
}

func raw_pwritev_part(bs *BlockDriverState, offset uint64, bytes uint64,
	qiov *QEMUIOVector, qiovOffset uint64, flags BdrvRequestFlags) error {

	s := bs.opaque.(*BDRVRawState)
	var localQiov QEMUIOVector
	var err error
	if s == nil || s.File == nil {
		return Err_NullObject
	}

	if qiovOffset > 0 || bytes != qiov.size {
		qemu_iovec_init_slice(&localQiov, qiov, qiovOffset, bytes)
		qiov = &localQiov
	}

	//call physical read for the qiov buffer
	ctx := context.Background()
	_, err = pwritev(ctx, s.File, qiov.iov, qiov.niov, offset)
	if qiov == &localQiov {
		qemu_iovec_destroy(&localQiov)
	}
	return err
}

func raw_flush_to_disk(bs *BlockDriverState) error {
	s := bs.opaque.(*BDRVRawState)
	return s.File.Sync()
}

func raw_block_status(bs *BlockDriverState, wantZero bool, offset uint64,
	bytes uint64, pnum *uint64, tmap *uint64, file **BlockDriverState) (uint64, error) {
	fmt.Println("[raw_block_status] no implementation")
	return 0, nil
}

func raw_pwrite_zeroes(bs *BlockDriverState, offset uint64, bytes uint64, flags BdrvRequestFlags) error {
	fmt.Println("[raw_pwrite_zeroes] no implementation")
	return nil
}

func raw_copy_range_from(bs *BlockDriverState, src *BdrvChild, offset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error {
	fmt.Println("[raw_copy_range_from] no implementation")
	return nil
}

func raw_copy_range_to(bs *BlockDriverState, src *BdrvChild, offset uint64,
	dst *BdrvChild, dstOffset uint64, bytes uint64,
	readFlags BdrvRequestFlags, writeFlags BdrvRequestFlags) error {
	fmt.Println("[raw_copy_range_to] no implementation")
	return nil
}
