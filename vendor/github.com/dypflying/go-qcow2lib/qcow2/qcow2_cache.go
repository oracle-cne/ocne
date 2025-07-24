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

type Qcow2CachedTable struct {
	offset     int64
	lruCounter uint64
	ref        int
	dirty      bool
}

type Qcow2Cache struct {
	entries              []Qcow2CachedTable
	depends              *Qcow2Cache
	size                 int
	tableSize            int
	dependsOnFlush       bool
	tableArray           []byte
	lruCounter           uint64
	cacheCleanLruCounter uint64
}

func qcow2_cache_get_table_addr(c *Qcow2Cache, tableIndex int) unsafe.Pointer {
	return unsafe.Pointer(&c.tableArray[tableIndex*c.tableSize])
}

func qcow2_cache_get_table_idx(c *Qcow2Cache, table unsafe.Pointer) int {
	tableOffset := int(uintptr(table) - uintptr(unsafe.Pointer(&c.tableArray[0])))
	tableIdx := tableOffset / c.tableSize
	Assert(tableIdx >= 0 && tableIdx < c.size && tableOffset%c.tableSize == 0)
	return tableIdx
}

func qcow2_cache_get_name(s *BDRVQcow2State, c *Qcow2Cache) string {
	if c == s.RefcountBlockCache {
		return "refcount block"
	} else if c == s.L2TableCache {
		return "L2 table"
	} else {
		return "unknown"
	}
}

func qcow2_cache_table_release(c *Qcow2Cache, i, numTables int) {
	//do nothing
}

func can_clean_entry(c *Qcow2Cache, i int) bool {
	t := c.entries[i]
	return t.ref == 0 && !t.dirty && t.offset != 0 && t.lruCounter <= c.cacheCleanLruCounter
}

// actually we never clean cache since it will auto evict caches during "cache get"
func qcow2_cache_clean_unused(c *Qcow2Cache) {

	var i int
	for i < c.size {
		var toClean int

		/* Skip the entries that we don't need to clean */
		for i < c.size && !can_clean_entry(c, i) {
			i++
		}

		/* And count how many we can clean in a row */
		for i < c.size && can_clean_entry(c, i) {
			c.entries[i].offset = 0
			c.entries[i].lruCounter = 0
			i++
			toClean++
		}

		if toClean > 0 {
			qcow2_cache_table_release(c, i-toClean, toClean)
		}
	}

	c.cacheCleanLruCounter = c.lruCounter
}

func qcow2_cache_create(bs *BlockDriverState, numTables uint32, tableSize uint32) *Qcow2Cache {

	if bs != nil {
		s := bs.opaque.(*BDRVQcow2State)
		Assert(numTables > 0)
		Assert(tableSize >= (1 << DEFAULT_CLUSTER_BITS))
		Assert(tableSize <= s.ClusterSize)
	}

	c := &Qcow2Cache{
		size:       int(numTables),
		tableSize:  int(tableSize),
		entries:    make([]Qcow2CachedTable, numTables),
		tableArray: make([]byte, numTables*tableSize),
	}
	if c.entries == nil || c.tableArray == nil {
		return nil
	}
	return c
}

func qcow2_cache_destroy(c *Qcow2Cache) error {
	//do nothing
	return nil
}

func qcow2_cache_flush_dependency(bs *BlockDriverState, c *Qcow2Cache) error {

	var err error
	if err = qcow2_cache_flush(bs, c.depends); err != nil {
		return err
	}

	c.depends = nil
	c.dependsOnFlush = false
	return nil
}

func qcow2_cache_entry_flush(bs *BlockDriverState, c *Qcow2Cache, i int) error {

	var err error
	if !c.entries[i].dirty || c.entries[i].offset == 0 {
		return nil
	}
	if c.depends != nil {
		err = qcow2_cache_flush_dependency(bs, c)
	} else if c.dependsOnFlush {
		if err = bdrv_flush(bs.current.bs); err == nil {
			c.dependsOnFlush = false
		}
	}
	if err != nil {
		return err
	}

	if err = bdrv_pwrite(bs.current, uint64(c.entries[i].offset),
		qcow2_cache_get_table_addr(c, i), uint64(c.tableSize)); err != nil {
		return err
	}
	c.entries[i].dirty = false

	return nil
}

func qcow2_cache_write(bs *BlockDriverState, c *Qcow2Cache) error {

	var err, ret error
	for i := int(0); i < c.size; i++ {
		if err = qcow2_cache_entry_flush(bs, c, i); err != nil && err != ERR_ENOSPC {
			ret = err
		}
	}
	return ret
}

func qcow2_cache_flush(bs *BlockDriverState, c *Qcow2Cache) error {
	err := qcow2_cache_write(bs, c)
	if err == nil {
		if err_flush := bdrv_flush(bs.current.bs); err_flush != nil {
			err = err_flush
		}
	}
	return err
}

func qcow2_cache_set_dependency(bs *BlockDriverState, c, dependency *Qcow2Cache) error {

	var err error
	if dependency.depends != nil {
		if err = qcow2_cache_flush_dependency(bs, dependency); err != nil {
			return err
		}
	}
	if c.depends != nil && c.depends != dependency {
		if err = qcow2_cache_flush_dependency(bs, c); err != nil {
			return err
		}
	}
	c.depends = dependency
	return nil
}

func qcow2_cache_depends_on_flush(c *Qcow2Cache) {
	c.dependsOnFlush = true
}

func qcow2_cache_empty(bs *BlockDriverState, c *Qcow2Cache) error {

	var err error
	if err = qcow2_cache_flush(bs, c); err != nil {
		return err
	}
	for i := int(0); i < c.size; i++ {
		Assert(c.entries[i].ref == 0)
		c.entries[i].offset = 0
		c.entries[i].lruCounter = 0
	}
	qcow2_cache_table_release(c, 0, c.size)
	c.lruCounter = 0
	return nil
}

func qcow2_cache_do_get(bs *BlockDriverState, c *Qcow2Cache, offset uint64, readFromDisk bool) (unsafe.Pointer, error) {

	i := int(0)
	var err error
	lookupIndex := int(0)
	minLruCounter := uint64(math.MaxUint64)
	minLruIndex := int(-1)

	Assert(offset != 0)
	// Check if the table is already cached
	lookupIndex = (int(offset) / c.tableSize * 4) % c.size
	i = lookupIndex
	for {
		t := c.entries[i]
		if uint64(t.offset) == offset {
			goto found
		}
		if t.ref == 0 && t.lruCounter < minLruCounter {
			minLruCounter = t.lruCounter
			minLruIndex = i
		}
		i++
		if i == c.size {
			i = 0
		}
		if i == lookupIndex {
			break
		}
	}

	// Cache miss: write a table back and replace it
	i = minLruIndex
	if err = qcow2_cache_entry_flush(bs, c, i); err != nil {
		return nil, err
	}

	c.entries[i].offset = 0
	if readFromDisk {
		//the object's size must be obtainable
		if err = bdrv_pread(bs.current, offset,
			qcow2_cache_get_table_addr(c, i), uint64(c.tableSize)); err != nil {
			return nil, err
		}
	}
	c.entries[i].offset = int64(offset)
found:
	c.entries[i].ref++
	return qcow2_cache_get_table_addr(c, i), nil
}

func qcow2_cache_get(bs *BlockDriverState, c *Qcow2Cache, offset uint64) (unsafe.Pointer, error) {
	return qcow2_cache_do_get(bs, c, offset, true)
}

func qcow2_cache_get_empty(bs *BlockDriverState, c *Qcow2Cache, offset uint64) (unsafe.Pointer, error) {
	return qcow2_cache_do_get(bs, c, offset, false)
}

func qcow2_cache_put(c *Qcow2Cache, table unsafe.Pointer) {
	i := qcow2_cache_get_table_idx(c, table)

	c.entries[i].ref--
	//table = nil

	if c.entries[i].ref == 0 {
		c.lruCounter++
		c.entries[i].lruCounter = c.lruCounter
	}
	Assert(c.entries[i].ref >= 0)
}

func qcow2_cache_entry_mark_dirty(c *Qcow2Cache, table unsafe.Pointer) {
	i := qcow2_cache_get_table_idx(c, table)
	Assert(c.entries[i].offset != 0)
	c.entries[i].dirty = true
}

func qcow2_cache_is_table_offset(c *Qcow2Cache, offset uint64) unsafe.Pointer {

	for i := int(0); i < c.size; i++ {
		if c.entries[i].offset == int64(offset) {
			return qcow2_cache_get_table_addr(c, i)
		}
	}
	return nil
}

func qcow2_cache_discard(c *Qcow2Cache, table unsafe.Pointer) {
	i := qcow2_cache_get_table_idx(c, table)

	Assert(c.entries[i].ref == 0)

	c.entries[i].offset = 0
	c.entries[i].lruCounter = 0
	c.entries[i].dirty = false

	qcow2_cache_table_release(c, i, 1)
}
