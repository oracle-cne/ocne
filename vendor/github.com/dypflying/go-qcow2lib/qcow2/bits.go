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

func clz32(x uint32) int {
	if x == 0 {
		return 32
	}
	clz := 0
	if x>>16 == 0 {
		clz += 16
		x <<= 16
	}
	if x>>24 == 0 {
		clz += 8
		x <<= 8
	}
	if x>>28 == 0 {
		clz += 4
		x <<= 4
	}
	if x>>30 == 0 {
		clz += 2
		x <<= 2
	}
	if x>>31 == 0 {
		clz++
	}
	return clz
}

func clo32(x uint32) int {
	return clz32(^x)
}

func ctz32(x uint32) int {
	if x == 0 {
		return 32
	}
	ctz := 0
	if x<<16 == 0 {
		ctz += 16
		x >>= 16
	}
	if x<<24 == 0 {
		ctz += 8
		x >>= 8
	}
	if x<<28 == 0 {
		ctz += 4
		x >>= 4
	}
	if x<<30 == 0 {
		ctz += 2
		x >>= 2
	}
	if x<<31 == 0 {
		ctz++
	}
	return ctz
}

func cto32(x uint32) int {
	return ctz32(^x)
}

func qcow_oflag_sub_alloc_range(x, y uint32) uint64 {
	return (1<<y - 1<<x)
}

func qcow_oflag_sub_zero_range(x, y uint32) uint64 {
	return qcow_oflag_sub_alloc_range(x, y) << 32
}

func qcow_oflag_sub_alloc(x uint32) int {
	return 1 << x
}

/* The subcluster X [0..31] reads as zeroes */
func qcow_oflag_sub_zero(x uint32) int {
	return qcow_oflag_sub_alloc(x) << 32
}
