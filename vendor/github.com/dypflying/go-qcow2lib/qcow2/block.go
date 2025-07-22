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
	"os"
	"unsafe"
)

func Blk_Create(filename string, options map[string]any) error {
	var err error
	if err = bdrv_create(filename, options); err != nil {
		return err
	}
	return err
}

func Blk_Probe(filename string) (string, error) {
	var err error
	var file *os.File
	var buffer bytes.Buffer
	var magic uint32
	buf := make([]byte, 4)
	if file, err = os.Open(filename); err != nil {
		return "", err
	}
	defer file.Close()
	if _, err = file.Read(buf); err != nil {
		return "", err
	}
	buffer.Write(buf)
	binary.Read(&buffer, binary.BigEndian, &magic)

	if magic == binary.BigEndian.Uint32(QCOW_MAGIC) {
		return TYPE_QCOW2_NAME, nil
	} else {
		return TYPE_RAW_NAME, nil
	}
}

/* this function return root BdrvChild
* default cache model is 'writeback',
* to use cache model 'none', set flags |= (BDRV_O_NOCACHE|BDRV_REQ_FUA)
* to use cache model 'writethrough', set flags |= BDRV_REQ_FUA
 */
func Blk_Open(filename string, options map[string]any, flags int) (*BdrvChild, error) {

	var child *BdrvChild
	var err error
	var format string

	if val, ok := options[OPT_FMT]; !ok {
		return nil, Err_IncompleteParameters
	} else {
		format = val.(string)
	}

	if child, err = bdrv_open_child(filename, format, options, flags); err != nil {
		return nil, err
	} else {
		bdrv_set_perm(child, PERM_ALL)
	}

	return child, err
}

func Blk_Close(child *BdrvChild) {
	if child == nil || child.bs == nil {
		return
	}
	bdrv_close(child.bs)
}

func Blk_Pread(root *BdrvChild, offset uint64, buf []uint8, bytes uint64) (uint64, error) {

	var qiov QEMUIOVector
	var err error
	if root == nil || root.bs == nil {
		return 0, Err_NullObject
	}

	qemu_iovec_init_buf(&qiov, unsafe.Pointer(&buf[0]), bytes)
	if err = bdrv_preadv_part(root, offset, bytes, &qiov, 0, 0); err != nil {
		return 0, err
	}
	return bytes, nil
}

// read object from the file, the object's size must be obtainable
func Blk_Pread_Object(child *BdrvChild, offset uint64, object any, size uint64) (uint64, error) {

	var buffer bytes.Buffer
	var err error
	var buf []byte
	var ret uint64
	if child == nil || child.bs == nil {
		return 0, Err_NullObject
	}
	buf = make([]byte, size)
	if ret, err = Blk_Pread(child, offset, buf[:], size); err != nil {
		return ret, err
	}
	buffer.Write(buf)
	binary.Read(&buffer, binary.BigEndian, object)

	return ret, nil
}

/*
* default write cache model is 'writeback',
* to use 'writethough' model, set 'flags |= BDRV_REQ_FUA'
* Note: if the BDRV_REQ_FUA flag has been set in the openflags when open the block dev
* the flag take effects for all writes as well
 */
func Blk_Pwrite(root *BdrvChild, offset uint64, buf []uint8,
	bytes uint64, flags BdrvRequestFlags) (uint64, error) {

	Assert(root != nil)
	bs := root.bs
	var qiov QEMUIOVector
	var err error
	if root == nil {
		return 0, Err_NullObject
	}
	if (bs.OpenFlags & BDRV_REQ_FUA) > 0 {
		flags |= BDRV_REQ_FUA
	}
	qemu_iovec_init_buf(&qiov, unsafe.Pointer(&buf[0]), bytes)
	if err = bdrv_pwritev_part(root, offset, bytes, &qiov, 0, flags); err != nil {
		return 0, err
	}
	return bytes, nil
}

func Blk_Pwrite_Zeroes(root *BdrvChild, offset uint64,
	bytes uint64, flags BdrvRequestFlags) (uint64, error) {

	Assert(root != nil)
	bs := root.bs
	var qiov QEMUIOVector
	var err error
	if root == nil {
		return 0, Err_NullObject
	}
	if (bs.OpenFlags & BDRV_REQ_FUA) > 0 {
		flags |= BDRV_REQ_FUA
	}
	qemu_iovec_init_buf(&qiov, nil, bytes)
	if err = bdrv_pwritev_part(root, offset, bytes, &qiov, 0, flags|BDRV_REQ_ZERO_WRITE); err != nil {
		return 0, err
	}
	return bytes, nil
}

// Write the object to the file in the bigendian manner, return -1 if error occurs
func Blk_Pwrite_Object(child *BdrvChild, offset uint64, object any, size uint64) (uint64, error) {

	var buffer bytes.Buffer
	var err error
	var ret uint64
	if child == nil || child.bs == nil {
		return 0, Err_NullObject
	}
	binary.Write(&buffer, binary.BigEndian, object)
	buffer.Truncate(int(size))
	if ret, err = Blk_Pwrite(child, offset, buffer.Bytes(), size, 0); err != nil {
		return ret, err
	}
	return ret, nil
}

/**
 * Return length in bytes on success, -errno on error.
 * The length is always a multiple of BDRV_SECTOR_SIZE.
 */
/*
 int64_t bdrv_getlength(BlockDriverState *bs)
*/
func Blk_Getlength(child *BdrvChild) (uint64, error) {

	bs := child.bs
	//equal to has_variable_length
	if bs.Drv != nil && bs.Drv.bdrv_getlength != nil {
		return bs.Drv.bdrv_getlength(bs)
	}
	var ret uint64 = bdrv_nb_sectors(bs)
	if ret > math.MaxInt64/BDRV_SECTOR_SIZE {
		return 0, ERR_E2BIG
	}
	return ret * BDRV_SECTOR_SIZE, nil
}

func Blk_Discard(child *BdrvChild, offset uint64, bytes uint64) error {
	return bdrv_pdiscard(child, offset, bytes)
}

func Blk_Flush(child *BdrvChild) error {
	Assert(child != nil)
	return bdrv_flush(child.bs)
}

func Blk_Info(child *BdrvChild, detail bool, pretty bool) string {
	bs := child.bs
	return bs.Info(detail, pretty)
}

func get_driver(fmt string) *BlockDriver {
	switch fmt {
	case "raw":
		return newRawDriver()
	case "qcow2":
		return newQcow2Driver()
	default:
		return nil
	}
}
