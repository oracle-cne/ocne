//
// Copyright 2022-2026 Sean C Foley
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tree

import (
	"math/big"
	"strconv"
)

// A PrefixLen indicates the length of the prefix for an address, section, division grouping, segment, or division.
// The zero value, which is nil, indicates that there is no prefix length.
type PrefixLen = *PrefixBitCount

// A PrefixBitCount is the count of bits in a non-nil PrefixLen.
type PrefixBitCount uint8

// Len returns the length of the prefix.  If the receiver is nil, representing the absence of a prefix length, returns 0.
// It will also return 0 if the receiver is a prefix with length of 0.  To distinguish the two, compare the receiver with nil.
func (p *PrefixBitCount) Len() BitCount {
	if p == nil {
		return 0
	}
	return p.bitCount()
}

func (p *PrefixBitCount) bitCount() BitCount {
	return BitCount(*p)
}

// IsNil returns true if this is nil, meaning it represents having no prefix length, or the absence of a prefix length
func (prefixBitCount *PrefixBitCount) IsNil() bool {
	return prefixBitCount == nil
}

// Equal compares two PrefixLen values for equality.  This method is intended for the PrefixLen type.  BitCount values should be compared with the == operator.
func (prefixBitCount *PrefixBitCount) Equal(other PrefixLen) bool {
	if prefixBitCount == nil {
		return other == nil
	}
	return other != nil && prefixBitCount.bitCount() == other.bitCount()
}

// Matches compares a PrefixLen value with a bit count
func (prefixBitCount *PrefixBitCount) Matches(other BitCount) bool {
	return prefixBitCount != nil && prefixBitCount.bitCount() == other
}

// Compare compares PrefixLen values, returning -1, 0, or 1 if this prefix length is less than, equal to, or greater than the given prefix length.
// This method is intended for the PrefixLen type.  BitCount values should be compared with ==, >, <, >= and <= operators.
func (prefixBitCount *PrefixBitCount) Compare(other PrefixLen) int {
	if prefixBitCount == nil {
		if other == nil {
			return 0
		}
		return 1
	} else if other == nil {
		return -1
	}
	return prefixBitCount.bitCount() - other.bitCount()
}

// String returns the bit count as a base-10 positive integer string, or "<nil>" if the receiver is a nil pointer.
func (prefixBitCount *PrefixBitCount) String() string {
	if prefixBitCount == nil {
		return nilString()
	}
	return strconv.Itoa(prefixBitCount.bitCount())
}

// A BitCount represents a count of bits in an address, section, grouping, segment, or division.
// Using signed integers allows for easier arithmetic, avoiding bugs.
// However, all methods adjust bit counts to match address size,
// so negative bit counts or bit counts larger than address size are meaningless.
type BitCount = int // using signed integers allows for easier arithmetic

func bigZero() *big.Int {
	return new(big.Int)
}

var zero = bigZero()

func bigZeroConst() *big.Int {
	return zero
}
