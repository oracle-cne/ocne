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
	"encoding/binary"
	"unsafe"
)

// implementation of c: memset, performance to be optimized with copy func
func memset(p unsafe.Pointer, count int) {
	if p == nil || count <= 0 {
		return
	}
	buffer := unsafe.Slice((*byte)(p), count)
	buffer[0] = 0
	for bp := 1; bp < count; bp *= 2 {
		copy(buffer[bp:], buffer[:bp])
	}
}

// implementation of c: memcpy
func memcpy(dst, src unsafe.Pointer, count uint64) {
	if dst == nil || src == nil || count <= 0 {
		return
	}
	buf_dst := unsafe.Slice((*byte)(dst), count)
	buf_src := unsafe.Slice((*byte)(src), count)
	copy(buf_dst[:], buf_src[:])
}

func cpu_to_be64(val uint64) uint64 {
	return binary.BigEndian.Uint64(int_to_bytes64(val))
}

func be64_to_cpu(val uint64) uint64 {
	dst := [8]byte{}
	binary.BigEndian.PutUint64(dst[:], val)
	return *(*uint64)(unsafe.Pointer(&dst[0]))
}

func cpu_to_be16(val uint16) uint16 {
	return binary.BigEndian.Uint16(int_to_bytes16(val))
}

func be16_to_cpu(val uint16) uint16 {
	dst := [2]byte{}
	binary.BigEndian.PutUint16(dst[:], val)
	return *(*uint16)(unsafe.Pointer(&dst[0]))
}

func int_to_bytes64(val uint64) []byte {
	buf := make([]uint8, 8)
	for i := 0; i < 8; i++ {
		buf[i] = uint8(val & 0xff)
		val = val >> 8
	}
	return buf
}

func int_to_bytes16(val uint16) []byte {
	buf := make([]uint8, 2)
	for i := 0; i < 2; i++ {
		buf[i] = uint8(val & 0xff)
		val = val >> 8
	}
	return buf
}

func align_down[V uint64 | uint32 | int | int32 | int64](n, m V) V {
	return n / m * m
}

func align_up[V uint64 | uint32 | int | int32 | int64](n, m V) V {
	return align_down(n+m-1, m)
}

func is_aligned[V uint64 | uint32 | int | int32 | int64](n, m V) bool {
	return (n%m == 0)
}

func round_up[V uint64 | uint32 | int | int32 | int64](n, m V) V {
	if n == 0 || m == 0 {
		return 0
	}
	return (n-1)/m*m + m
}

func round_down[V uint64 | uint32 | int | int32 | int64](n, m V) V {
	if n == 0 || m == 0 {
		return 0
	}
	return n / m * m
}

func max[V uint64 | uint32 | int | int32 | int64](vars ...V) V {
	if len(vars) == 0 {
		return 0
	}
	max := vars[0]
	for i := 1; i < len(vars); i++ {
		if vars[i] > max {
			max = vars[i]
		}
	}
	return max
}

func min[V uint64 | uint32 | int | int32 | int64](vars ...V) V {
	if len(vars) == 0 {
		return 0
	}
	min := vars[0]
	for i := 1; i < len(vars); i++ {
		if vars[i] < min {
			min = vars[i]
		}
	}
	return min
}

/*
 * Checks if a buffer is all zeroes
 */
func buffer_is_zero(buf []byte, length uint64) bool {

	if length == 0 {
		return true
	}

	for i := uint64(0); i < length; i++ {
		if buf[i] != 0 {
			return false
		}
	}
	return true
}

func Assert(expr bool) {
	if !expr {
		panic("unexpected:")
	}
}

func interface2uint64(i interface{}) uint64 {
	switch i.(type) {
	case int:
		return uint64(i.(int))
	case uint:
		return uint64(i.(uint))
	case int16:
		return uint64(i.(int16))
	case uint16:
		return uint64(i.(uint16))
	case int32:
		return uint64(i.(int32))
	case uint32:
		return uint64(i.(uint32))
	case int64:
		return uint64(i.(int64))
	case uint64:
		return i.(uint64)
	}
	return 0
}
