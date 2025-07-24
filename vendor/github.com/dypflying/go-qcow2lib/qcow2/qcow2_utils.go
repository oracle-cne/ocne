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
	"unsafe"
)

// return the start point of a cluster, in which the offset falls
func start_of_cluster(s *BDRVQcow2State, offset uint64) uint64 {
	return offset &^ uint64(s.ClusterSize-1)
}

// return the offset after mask with the cluster size
func offset_into_cluster(s *BDRVQcow2State, offset uint64) uint64 {
	return offset & (uint64(s.ClusterSize) - 1)
}

// return the number of clusters
func size_to_clusters(s *BDRVQcow2State, size uint64) uint64 {
	return (size + uint64(s.ClusterSize-1)) >> uint(s.ClusterBits)
}

// return the number of l1s
func size_to_l1(s *BDRVQcow2State, size uint64) uint64 {
	shift := s.ClusterBits + s.L2Bits
	return (size + (1 << uint(shift)) - 1) >> uint(shift)
}

// return the l1 index, which holds the offset
func offset_to_l1_index(s *BDRVQcow2State, offset uint64) uint64 {
	return uint64(offset >> (s.L2Bits + s.ClusterBits))
}

// return the l2 index, which holds the offset
func offset_to_l2_index(s *BDRVQcow2State, offset uint64) uint64 {
	return (offset >> uint(s.ClusterBits) & uint64(s.L2Size-1))
}

// return the l2 slice index, which holds the offset
// note: for SC enabled l2 table, which doubles from the normal L2
func offset_to_l2_slice_index(s *BDRVQcow2State, offset uint64) uint64 {
	return uint64(offset>>s.ClusterBits) & uint64(s.L2SliceSize-1)
}

// offset to a SC index
func offset_to_sc_index(s *BDRVQcow2State, offset uint64) uint64 {
	return uint64(offset>>s.SubclusterBits) & (s.SubclustersPerCluster - 1)
}

func offset_into_subcluster(s *BDRVQcow2State, offset uint64) uint64 {
	return offset & (s.SubclusterSize - 1)
}

func size_to_subclusters(s *BDRVQcow2State, size uint64) uint64 {
	return (size + (s.SubclusterSize - 1)) >> s.SubclusterBits
}

func get_l2_entry(s *BDRVQcow2State, l2Slice unsafe.Pointer, idx uint32) uint64 {
	idx *= uint32(l2_entry_size(s) / SIZE_UINT64)
	ret := *(*uint64)(unsafe.Pointer(uintptr(l2Slice) + uintptr(idx*uint32(SIZE_UINT64))))
	ret = be64_to_cpu(ret)
	return ret
}

func set_l2_entry(s *BDRVQcow2State, l2Slice unsafe.Pointer, idx uint32, entry uint64) {
	idx *= uint32(l2_entry_size(s) / SIZE_UINT64)
	*(*uint64)(unsafe.Pointer(uintptr(l2Slice) + uintptr(idx*uint32(SIZE_UINT64)))) = cpu_to_be64(entry)
}

func get_l2_bitmap(s *BDRVQcow2State, l2Slice unsafe.Pointer, idx uint32) uint64 {
	if has_subclusters(s) {
		idx *= uint32(l2_entry_size(s) / SIZE_UINT64)
		ret := *(*uint64)(unsafe.Pointer(uintptr(l2Slice) + uintptr((idx+1)*uint32(SIZE_UINT64))))
		ret = be64_to_cpu(ret)
		return ret
	} else {
		return 0 /* For convenience only; this value has no meaning. */
	}
}

func set_l2_bitmap(s *BDRVQcow2State, l2Slice unsafe.Pointer, idx uint32, bitmap uint64) {
	idx *= uint32(l2_entry_size(s) / SIZE_UINT64)
	*(*uint64)(unsafe.Pointer(uintptr(l2Slice) + uintptr((idx+1)*uint32(SIZE_UINT64)))) = cpu_to_be64(bitmap)
}

func has_subclusters(s *BDRVQcow2State) bool {
	return s.IncompatibleFeatures&QCOW2_INCOMPAT_EXTL2 > 0
}

func l2_entry_size(s *BDRVQcow2State) uint64 {
	if has_subclusters(s) {
		return L2E_SIZE_EXTENDED
	}
	return L2E_SIZE_NORMAL
}

func l2meta_cow_start(m *QCowL2Meta) uint64 {
	return m.Offset + m.CowStart.Offset
}

func l2meta_cow_end(m *QCowL2Meta) uint64 {
	return m.Offset + m.CowEnd.Offset + m.CowEnd.NbBytes
}
