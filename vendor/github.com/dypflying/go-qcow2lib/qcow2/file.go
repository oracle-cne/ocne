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
	"os"
	"unsafe"
)

/*
 * write the iov to the raw file, whose byte order follows the iov.
 */
func pwritev(ctx context.Context, file *os.File, iov []iovec, iovcnt int, offset uint64) (uint64, error) {

	i := int(0)
	ret := uint64(0)
	var err error
	var n int

	file.Seek(int64(offset), 0)
	for i < iovcnt {
		buffer := unsafe.Slice((*byte)(iov[i].iov_base), iov[i].iov_len)
		if n, err = file.Write(buffer); err != nil {
			return ret, err
		} else if n > 0 {
			ret += uint64(n)
		} else if n == 0 {
			break
		} else if err == ERR_EINTR {
			continue
		} else {
			/* else it is some "other" error,
			 * only return if there was no data processed. */
			if n == 0 {
				return ret, err
			}
			break
		}
		i++
	}
	return ret, nil
}

/*
 * read the iov from the raw file, whose byte order follows the iov.
 */
func preadv(ctx context.Context, file *os.File, iov []iovec, iovcnt int, offset uint64) (uint64, error) {

	i := int(0)
	ret := uint64(0)
	var err error
	var n int

	file.Seek(int64(offset), 0)
	for i < iovcnt {

		buffer := unsafe.Slice((*byte)(iov[i].iov_base), iov[i].iov_len)

		if n, err = file.Read(buffer); err != nil {
			return ret, err
		} else if n > 0 {
			ret += uint64(n)
		} else if n == 0 {
			break
		} else if err == ERR_EINTR {
			continue
		} else {
			/* else it is some "other" error,
			 * only return if there was no data processed. */
			if n == 0 {
				return ret, err
			}
			break
		}
		i++

	}
	return ret, nil
}
