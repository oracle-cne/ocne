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
	"container/list"
	"fmt"
	"math"
	"unsafe"
)

func get_refcount(refcountArray unsafe.Pointer, index uint64) uint16 {
	value := *(*uint16)(unsafe.Pointer(uintptr(refcountArray) + uintptr(index*2))) //uint16 occpies 2 bytes.
	value = be16_to_cpu(value)
	return value
}

func set_refcount(refcountArray unsafe.Pointer, index uint64, value uint16) {
	p := (*uint16)(unsafe.Pointer(uintptr(refcountArray) + uintptr(index*2))) //uint16 occpies 2 bytes.
	*p = cpu_to_be16(value)
}

// Initate the refcount table
func qcow2_refcount_init(bs *BlockDriverState) error {

	s := bs.opaque.(*BDRVQcow2State)
	var err error
	s.RefcountTable = make([]uint64, s.RefcountTableSize)

	if s.RefcountTableSize > 0 {
		//try read the refcount table
		if err = bdrv_pread(bs.current, s.RefcountTableOffset, unsafe.Pointer(&s.RefcountTable[0]),
			uint64(s.RefcountTableSize)*SIZE_UINT64); err != nil {
			return err
		}
		for i := uint32(0); i < s.RefcountTableSize; i++ {
			s.RefcountTable[i] = be64_to_cpu(s.RefcountTable[i])
		}
		update_max_refcount_table_index(s)
	}
	return nil
}

func qcow2_refcount_close(bs *BlockDriverState) {
	//do nothing
}

func load_refcount_block(bs *BlockDriverState, refcountBlockOffset uint64) (unsafe.Pointer, error) {
	s := bs.opaque.(*BDRVQcow2State)
	return qcow2_cache_get(bs, s.RefcountBlockCache, refcountBlockOffset)
}

func qcow2_get_refcount(bs *BlockDriverState, clusterIndex uint64) (uint16, error) {
	s := bs.opaque.(*BDRVQcow2State)
	var refcountTableIndex, blockIndex uint64
	var refcountBlockOffset uint64
	var err error
	var refcountBlock unsafe.Pointer

	refcount := uint16(0)
	refcountTableIndex = clusterIndex >> s.RefcountBlockBits
	if refcountTableIndex >= uint64(s.RefcountTableSize) {
		return 0, nil
	}
	refcountBlockOffset = s.RefcountTable[refcountTableIndex] & REFT_OFFSET_MASK
	if refcountBlockOffset == 0 {
		return 0, nil
	}

	if offset_into_cluster(s, refcountBlockOffset) > 0 {
		return 0, ERR_EIO
	}

	if refcountBlock, err = qcow2_cache_get(bs, s.RefcountBlockCache, refcountBlockOffset); err != nil {
		return 0, err
	}
	blockIndex = clusterIndex & uint64(s.RefcountBlockSize-1)
	refcount = s.get_refcount(refcountBlock, blockIndex)

	qcow2_cache_put(s.RefcountBlockCache, refcountBlock)

	return refcount, nil
}

/* Checks if two offsets are described by the same refcount block */
func in_same_refcount_block(s *BDRVQcow2State, offsetA uint64, offsetB uint64) bool {
	blockA := offsetA >> (s.ClusterBits + s.RefcountBlockBits)
	blockB := offsetB >> (s.ClusterBits + s.RefcountBlockBits)
	return (blockA == blockB)
}

func alloc_refcount_block(bs *BlockDriverState, clusterIndex uint64) (unsafe.Pointer, error) {

	s := bs.opaque.(*BDRVQcow2State)
	var refcountTableIndex uint64
	var err error
	var refcountBlock unsafe.Pointer
	var metaOffset, blocksUsed uint64
	var data64 uint64

	refcountTableIndex = clusterIndex >> s.RefcountBlockBits

	if uint32(refcountTableIndex) < s.RefcountTableSize {

		refcountBlockOffset := s.RefcountTable[refcountTableIndex] & REFT_OFFSET_MASK
		//this means we already have a allocated refcount block
		if refcountBlockOffset > 0 {
			if offset_into_cluster(s, refcountBlockOffset) > 0 {
				return nil, ERR_EAGAIN
			}

			if refcountBlock, err = load_refcount_block(bs, refcountBlockOffset); err != nil {
				return nil, err
			}
			return refcountBlock, nil
		}
	}

	/* We write to the refcount table, so we might depend on L2 tables */
	if err = qcow2_cache_flush(bs, s.L2TableCache); err != nil {
		return nil, err
	}
	/* Allocate the refcount block itself and mark it as used */
	var newBlockOffset uint64
	if newBlockOffset, err = alloc_clusters_noref(bs, uint64(s.ClusterSize), math.MaxInt64); err != nil {
		return nil, err
	} else if newBlockOffset == 0 {
		return nil, Err_RefcountAlloc
	}
	Assert((newBlockOffset & REFT_OFFSET_MASK) == newBlockOffset)

	if in_same_refcount_block(s, uint64(newBlockOffset), clusterIndex<<uint64(s.ClusterBits)) {
		/* Zero the new refcount block before updating it */
		if refcountBlock, err = qcow2_cache_get_empty(bs, s.RefcountBlockCache, uint64(newBlockOffset)); err != nil {
			goto fail
		}

		memset(refcountBlock, int(s.ClusterSize))

		/* The block describes itself, need to update the cache */
		blockIndex := (newBlockOffset >> uint64(s.ClusterBits)) & uint64(s.RefcountBlockSize-1)
		s.set_refcount(refcountBlock, uint64(blockIndex), 1)
	} else {
		if err = update_refcount(bs, newBlockOffset, uint64(s.ClusterSize), 1, false, QCOW2_DISCARD_NEVER); err != nil {
			goto fail
		}
		if err = qcow2_cache_flush(bs, s.RefcountBlockCache); err != nil {
			goto fail
		}
		if refcountBlock, err = qcow2_cache_get_empty(bs, s.RefcountBlockCache, uint64(newBlockOffset)); err != nil {
			goto fail
		}
		memset(refcountBlock, int(s.ClusterSize))
	}

	/* Now the new refcount block needs to be written to disk */
	qcow2_cache_entry_mark_dirty(s.RefcountBlockCache, refcountBlock)
	if err = qcow2_cache_flush(bs, s.RefcountBlockCache); err != nil {
		goto fail
	}

	if refcountTableIndex < uint64(s.RefcountTableSize) {
		data64 = cpu_to_be64(newBlockOffset)
		if err = bdrv_pwrite(bs.current, s.RefcountTableOffset+refcountTableIndex*REFTABLE_ENTRY_SIZE,
			unsafe.Pointer(&data64), REFTABLE_ENTRY_SIZE); err != nil {
			goto fail
		}

		s.RefcountTable[refcountTableIndex] = uint64(newBlockOffset)

		if s.MaxRefcountTableIndex < uint32(refcountTableIndex) {
			s.MaxRefcountTableIndex = uint32(refcountTableIndex)
		}

		return refcountBlock, ERR_EAGAIN
	}

	qcow2_cache_put(s.RefcountBlockCache, refcountBlock)

	blocksUsed = round_up(max(clusterIndex+1, newBlockOffset>>s.ClusterBits+1),
		uint64(s.RefcountBlockSize))

	metaOffset = (blocksUsed * uint64(s.RefcountBlockSize)) * uint64(s.ClusterSize)

	if _, err = qcow2_refcount_area(bs, metaOffset, 0, false, refcountTableIndex, uint64(newBlockOffset)); err != nil {
		return nil, err
	}

	if refcountBlock, err = load_refcount_block(bs, uint64(newBlockOffset)); err != nil {
		return nil, err
	}
	return refcountBlock, ERR_EAGAIN

fail:
	if refcountBlock != nil {
		qcow2_cache_put(s.RefcountBlockCache, refcountBlock)
	}
	return refcountBlock, err
}

func qcow2_refcount_area(bs *BlockDriverState, startOffset uint64, additionalClusters uint64,
	exactSize bool, newRefblockIndex uint64, newRefblockOffset uint64) (uint64, error) {

	s := bs.opaque.(*BDRVQcow2State)
	var totalRefblockCount_u64, additionalRefblockCount uint64
	var totalRefblockCount, tableSize, areaReftableIndex, tableClusters uint64
	var i uint64
	var tableOffset, blockOffset, endOffset uint64
	var err error
	var newTable []uint64
	var oldTableOffset, oldTableSize uint64

	Assert((startOffset % uint64(s.ClusterSize)) == 0)

	qcow2_refcount_metadata_size(startOffset/uint64(s.ClusterSize)+additionalClusters,
		uint64(s.ClusterSize), QCOW2_REFCOUNT_ORDER,
		!exactSize, &totalRefblockCount_u64)

	if totalRefblockCount_u64 > QCOW_MAX_REFTABLE_SIZE {
		return 0, ERR_EFBIG
	}

	totalRefblockCount = totalRefblockCount_u64

	areaReftableIndex = (startOffset / uint64(s.ClusterSize)) / uint64(s.RefcountBlockSize)

	if exactSize {
		tableSize = totalRefblockCount
	} else {
		tableSize = totalRefblockCount + round_up(totalRefblockCount, 2)
	}

	/* The qcow2 file can only store the reftable size in number of clusters */
	tableSize = round_up(tableSize, uint64(s.ClusterSize)/REFTABLE_ENTRY_SIZE)
	tableClusters = (tableSize * REFTABLE_ENTRY_SIZE) / uint64(s.ClusterSize)

	if tableSize > QCOW_MAX_REFTABLE_SIZE {
		return 0, ERR_EFBIG
	}

	Assert(tableSize > 0)
	newTable = make([]uint64, tableSize)

	/* Fill the new refcount table */
	if tableSize > uint64(s.MaxRefcountTableIndex) {
		memcpy(unsafe.Pointer(&newTable[0]), unsafe.Pointer(&s.RefcountTable[0]),
			uint64(s.MaxRefcountTableIndex+1)*uint64(REFTABLE_ENTRY_SIZE))
	} else {
		memcpy(unsafe.Pointer(&newTable[0]), unsafe.Pointer(&s.RefcountTable[0]),
			uint64(tableSize*REFTABLE_ENTRY_SIZE))
	}

	if newRefblockOffset > 0 {
		Assert(newRefblockOffset < totalRefblockCount)
		newTable[newRefblockIndex] = newRefblockOffset
	}

	/* Count how many new refblocks we have to create */
	additionalRefblockCount = 0
	for i = areaReftableIndex; i < totalRefblockCount; i++ {
		if newTable[i] == 0 {
			additionalRefblockCount++
		}
	}

	tableOffset = startOffset + additionalRefblockCount*uint64(s.ClusterSize)
	endOffset = tableOffset + tableClusters*uint64(s.ClusterSize)

	/* Fill the refcount blocks, and create new ones, if necessary */
	blockOffset = startOffset
	for i = areaReftableIndex; i < totalRefblockCount; i++ {
		var refblockData unsafe.Pointer
		var firstOffsetCovered uint64

		/* Reuse an existing refblock if possible, create a new one otherwise */
		if newTable[i] > 0 {
			if refblockData, err = qcow2_cache_get(bs, s.RefcountBlockCache, newTable[i]); err != nil {
				goto fail
			}
		} else {
			if refblockData, err = qcow2_cache_get_empty(bs, s.RefcountBlockCache,
				blockOffset); err != nil {
				goto fail
			}
			memset(refblockData, int(s.ClusterSize))
			qcow2_cache_entry_mark_dirty(s.RefcountBlockCache, refblockData)

			newTable[i] = blockOffset
			blockOffset += uint64(s.ClusterSize)
		}

		/* First host offset covered by this refblock */
		firstOffsetCovered = i * uint64(s.RefcountBlockSize) * uint64(s.ClusterSize)
		if firstOffsetCovered < endOffset {
			var j, endIndex uint64
			if firstOffsetCovered < startOffset {
				j = (startOffset - firstOffsetCovered) / uint64(s.ClusterSize)
			} else {
				j = 0
			}

			endIndex = min((endIndex-firstOffsetCovered)/uint64(s.ClusterSize), uint64(s.RefcountBlockSize))

			for ; j < endIndex; j++ {
				s.set_refcount(refblockData, j, 1)
			}

			qcow2_cache_entry_mark_dirty(s.RefcountBlockCache, refblockData)
		}

		qcow2_cache_put(s.RefcountBlockCache, refblockData)
	} // end for

	Assert(blockOffset == tableOffset)

	if err = qcow2_cache_flush(bs, s.RefcountBlockCache); err != nil {
		goto fail
	}
	/* Write refcount table to disk */
	for i = 0; i < totalRefblockCount; i++ {
		newTable[i] = cpu_to_be64(newTable[i])
	}

	if err = bdrv_pwrite(bs.current, tableOffset, unsafe.Pointer(&newTable[0]), tableSize*REFTABLE_ENTRY_SIZE); err != nil {
		goto fail
	}

	for i = 0; i < totalRefblockCount; i++ {
		newTable[i] = be64_to_cpu(newTable[i])
	}

	//TODO2
	//update the header for the new refcount table
	//since one refcount table can hold up to 16TiB data, and the qcow2 is limited to 4TiB, so never reach here
	//just reserved for sometime in future that we unlimit it.
	Assert(false)

	/* And switch it in memory */
	oldTableOffset = uint64(s.RefcountTableOffset)
	oldTableSize = uint64(s.RefcountTableSize)

	s.RefcountTable = newTable
	s.RefcountTableSize = uint32(tableSize)
	s.RefcountTableOffset = tableOffset
	update_max_refcount_table_index(s)

	/* Free old table. */
	qcow2_free_clusters(bs, oldTableOffset, oldTableSize*REFTABLE_ENTRY_SIZE, QCOW2_DISCARD_OTHER)

	return endOffset, nil

fail:
	return 0, err
}

func update_refcount(bs *BlockDriverState, offset uint64, length uint64, addend uint64,
	decrease bool, dType Qcow2DiscardType) error {

	s := bs.opaque.(*BDRVQcow2State)
	var start, last, clusterOffset uint64
	var refcountBlock unsafe.Pointer
	oldTableIndex := int64(-1)
	var err error

	if length == 0 {
		return nil
	}

	if decrease {
		qcow2_cache_set_dependency(bs, s.RefcountBlockCache, s.L2TableCache)
	}

	start = start_of_cluster(s, offset)
	last = start_of_cluster(s, offset+length-1)
	for clusterOffset = start; clusterOffset <= last; clusterOffset += uint64(s.ClusterSize) {
		var blockIndex int64
		var refcount uint16
		clusterIndex := int64(clusterOffset >> s.ClusterBits)
		tableIndex := int64(clusterIndex >> s.RefcountBlockBits)
		/* Load the refcount block and allocate it if needed */
		if tableIndex != oldTableIndex {
			if refcountBlock != nil {
				qcow2_cache_put(s.RefcountBlockCache, refcountBlock)
			}
			if refcountBlock, err = alloc_refcount_block(bs, uint64(clusterIndex)); err == ERR_EAGAIN {
				if s.FreeClusterIndex > uint64(start>>s.ClusterBits) {
					s.FreeClusterIndex = uint64(start >> s.ClusterBits)
				}
			} else if err != nil {
				goto fail
			}
		}
		oldTableIndex = tableIndex
		qcow2_cache_entry_mark_dirty(s.RefcountBlockCache, refcountBlock)

		blockIndex = clusterIndex & int64(s.RefcountBlockSize-1)
		refcount = s.get_refcount(refcountBlock, uint64(blockIndex))

		if (decrease && refcount-uint16(addend) > refcount) || (!decrease && (refcount+uint16(addend) < refcount)) {
			err = ERR_EINVAL
			goto fail
		}

		if decrease {
			refcount -= uint16(addend)
		} else {
			refcount += uint16(addend)
		}

		if refcount == 0 && uint64(clusterIndex) < s.FreeClusterIndex {
			s.FreeClusterIndex = uint64(clusterIndex)
		}
		s.set_refcount(refcountBlock, uint64(blockIndex), refcount)

		if refcount == 0 {
			var table unsafe.Pointer

			table = qcow2_cache_is_table_offset(s.RefcountBlockCache, uint64(offset))
			if table != nil {
				qcow2_cache_put(s.RefcountBlockCache, refcountBlock)
				oldTableIndex = -1
				//qcow2_cache_discard(s->refcount_block_cache, table);
			}
			table = qcow2_cache_is_table_offset(s.L2TableCache, uint64(offset))
			if table != nil {
				qcow2_cache_discard(s.L2TableCache, table)
			}

			if s.DiscardPassthrough[dType] {
				update_refcount_discard(bs, clusterOffset, uint64(s.ClusterSize))
			}
		}
	}
	err = nil

fail:
	/* Write last changed block to disk */
	if refcountBlock != nil {
		qcow2_cache_put(s.RefcountBlockCache, refcountBlock)
	}
	if err != nil {
		update_refcount(bs, offset, clusterOffset-offset, addend, !decrease, QCOW2_DISCARD_NEVER)
	}
	return err
}

func qcow2_update_cluster_refcount(bs *BlockDriverState, clusterIndex uint64, addend uint64,
	decrease bool, dType Qcow2DiscardType) error {

	s := bs.opaque.(*BDRVQcow2State)
	var err error
	if err = update_refcount(bs, clusterIndex<<s.ClusterBits, 1, addend, decrease, dType); err != nil {
		return err
	}
	return nil
}

func alloc_clusters_noref(bs *BlockDriverState, size uint64, max uint64) (uint64, error) {

	s := bs.opaque.(*BDRVQcow2State)
	var nbClusters uint64
	var refcount uint16
	var err error

	nbClusters = size_to_clusters(s, size)
retry:
	//try to find nbClusters of continuous empty blocks.
	for i := uint64(0); i < nbClusters; i++ {
		nextClusterIndex := s.FreeClusterIndex
		s.FreeClusterIndex++
		if refcount, err = qcow2_get_refcount(bs, nextClusterIndex); err != nil {
			return 0, err
		} else if refcount != 0 {
			goto retry
		}
	}

	if s.FreeClusterIndex > 0 && s.FreeClusterIndex-1 > (max>>uint(s.ClusterBits)) {
		return 0, ERR_EFBIG
	}
	//return the first pointer of the continuous blocks.
	return (s.FreeClusterIndex - nbClusters) << uint64(s.ClusterBits), nil
}

func qcow2_alloc_clusters(bs *BlockDriverState, size uint64) (uint64, error) {

	var offset uint64
	var err error
	for {
		if offset, err = alloc_clusters_noref(bs, size, MAX_QCOW2_SIZE); err != nil || offset < 0 {
			return offset, err
		}
		err = update_refcount(bs, offset, size, 1, false, QCOW2_DISCARD_NEVER)
		if err != ERR_EAGAIN {
			break
		}
	}
	return offset, err
}

func qcow2_alloc_clusters_at(bs *BlockDriverState, offset uint64, nbClusters int64) (uint64, error) {

	s := bs.opaque.(*BDRVQcow2State)
	var clusterIndex, i uint64
	var refcount uint16
	var err error

	Assert(nbClusters >= 0)
	if nbClusters == 0 {
		return 0, nil
	}
	for {
		clusterIndex = offset >> s.ClusterBits
		for i = 0; i < uint64(nbClusters); i++ {
			refcount, err = qcow2_get_refcount(bs, clusterIndex)
			clusterIndex++
			if err != nil {
				return 0, err
			} else if refcount != 0 {
				break
			}
		}
		err = update_refcount(bs, offset, i<<s.ClusterBits, 1, false, QCOW2_DISCARD_NEVER)
		if err != ERR_EAGAIN {
			break
		}
	}

	if err != nil {
		return 0, err
	}
	return uint64(i), nil
}

func qcow2_free_clusters(bs *BlockDriverState, offset uint64, size uint64, dType Qcow2DiscardType) {
	if err := update_refcount(bs, offset, size, 1, true, dType); err != nil {
		fmt.Printf("qcow2_free_clusters failed, err: %v\n", err)
	}
}

func qcow2_write_caches(bs *BlockDriverState) error {

	s := bs.opaque.(*BDRVQcow2State)
	var err error

	if err = qcow2_cache_write(bs, s.L2TableCache); err != nil {
		return err
	}
	if err = qcow2_cache_write(bs, s.RefcountBlockCache); err != nil {
		return err
	}
	return nil
}

func qcow2_flush_caches(bs *BlockDriverState) error {

	if err := qcow2_write_caches(bs); err != nil {
		return err
	}
	return bdrv_flush(bs.current.bs)
}

func update_max_refcount_table_index(s *BDRVQcow2State) {

	i := s.RefcountTableSize - 1
	for i > 0 && (s.RefcountTable[i]&REFT_OFFSET_MASK) == 0 {
		i--
	}
	s.MaxRefcountTableIndex = i
}

func update_refcount_discard(bs *BlockDriverState,
	offset uint64, length uint64) {

	s := bs.opaque.(*BDRVQcow2State)
	var d, p *Qcow2DiscardRegion
	var i *list.Element

	for i = s.Discards.Front(); i != nil; i = i.Next() {
		d = i.Value.(*Qcow2DiscardRegion)
		newStart := min(offset, d.offset)
		newEnd := max(offset+length, d.offset+d.bytes)
		if newEnd-newStart <= length+d.bytes {
			Assert(d.bytes+length == newEnd-newStart)
			d.offset = newStart
			d.bytes = newEnd - newStart
			goto found
		}
	}
	d = &Qcow2DiscardRegion{
		bs:     bs,
		offset: offset,
		bytes:  length,
	}
	s.Discards.PushBack(d)

found:
	/* Merge discard requests if they are adjacent now */
	for i = s.Discards.Front(); i != nil; i = i.Next() {
		p = i.Value.(*Qcow2DiscardRegion)
		if p == d || p.offset > d.offset+d.bytes || d.offset > p.offset+p.bytes {
			continue
		}
		s.Discards.Remove(i)
		d.offset = min(d.offset, p.offset)
		d.bytes += p.bytes
	}

}

func qcow2_process_discards(bs *BlockDriverState, err error) {
	s := bs.opaque.(*BDRVQcow2State)
	var d *Qcow2DiscardRegion
	var e *list.Element

	for e = s.Discards.Front(); e != nil; e = e.Next() {
		d = e.Value.(*Qcow2DiscardRegion)
		s.Discards.Remove(e)
		if err == nil {
			bdrv_pdiscard(bs.current, d.offset, d.bytes)
		}
	}
}
