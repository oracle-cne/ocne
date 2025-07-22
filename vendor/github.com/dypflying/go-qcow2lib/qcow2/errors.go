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
	"fmt"
	"syscall"
)

var (
	ERR_EIO     = syscall.EIO
	ERR_E2BIG   = syscall.E2BIG
	ERR_EFBIG   = syscall.EFBIG
	ERR_EINTR   = syscall.EINTR
	ERR_ENOTSUP = syscall.ENOTSUP
	ERR_ENOSPC  = syscall.ENOSPC
	ERR_EINVAL  = syscall.EINVAL
	ERR_EAGAIN  = syscall.EAGAIN

	Err_IdxOutOfRange        = fmt.Errorf("index is out of range")
	Err_NoDriverFound        = fmt.Errorf("no driver found")
	Err_NullObject           = fmt.Errorf("null object")
	Err_IncompleteParameters = fmt.Errorf("incomplete parameters")
	Err_L2Alloc              = fmt.Errorf("allocate l2 table fails")
	Err_RefcountAlloc        = fmt.Errorf("allocate refcount block fails")
	Err_NoWritePerm          = fmt.Errorf("no write permission")
	Err_NoReadPerm           = fmt.Errorf("no read permission")
	Err_Misaligned           = fmt.Errorf("misaligned")
)
