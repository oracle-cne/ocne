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
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type binTree[E Key[E], V any] struct {
	root *binTreeNode[E, V]
}

// GetRoot returns the root node of this trie, which can be nil for a zero-valued uninitialized trie, but not for any other trie
func (tree *binTree[E, V]) GetRoot() *binTreeNode[E, V] {
	return tree.root
}

// Size returns the number of elements in the tree.
// Only nodes for which IsAdded() returns true are counted.
// When zero is returned, IsEmpty() returns true.
func (tree *binTree[E, V]) Size() int {
	if tree == nil {
		return 0
	}
	return tree.GetRoot().Size()
}

// NodeSize returns the number of nodes in the tree, which is always more than the number of elements.
func (tree *binTree[E, V]) NodeSize() int {
	if tree == nil {
		return 0
	}
	return tree.GetRoot().NodeSize()
}

// ContainingMaxElements returns true if and only if the total number of individial keys contained by the prefix block keys of
// added nodes in the trie is the maximum possible.
// In other words, the keys of the added nodes together contain all the possible individual sub-keys.
func (tree *binTree[E, V]) ContainingMaxElements() bool {
	if tree == nil {
		return false
	}
	return tree.GetRoot().ContainingMaxElements()
}

// Returns the total number of keys covered by keys added to the sub-tree starting from this node as root and moving downwards to sub-nodes.
func (tree *binTree[E, V]) GetMatchingKeyCount() *big.Int {
	if tree == nil {
		return bigZero()
	}
	return tree.GetRoot().GetMatchingKeyCount()
}

// GetKeyElementBig returns the added node containing the given index into the keys of the trie, with the index of zero returning the first added node.
// It also returns the remaining index into the key of the returned node.
//
// If the increment is negative, or the increment exceeds GetCount() - 1, GetKeyElementBig panics.
func (tree *binTree[E, V]) GetKeyElementBig(keyIndex *big.Int) (*binTreeNode[E, V], *big.Int) {
	return tree.GetRoot().GetKeyElementBig(keyIndex)
}

// GetKeyElement returns the added node containing the given index into the keys of the trie, with the index of zero returning the first added node.
// It also returns the remaining index into the key of the returned node.
//
// If the increment is negative, or the increment exceeds GetCount() - 1, this panics.
func (tree *binTree[E, V]) GetKeyElement(keyIndex int64) (*binTreeNode[E, V], int64) {
	return tree.GetRoot().GetKeyElement(keyIndex)
}

// Clear removes all added nodes from the tree, after which IsEmpty() will return true
func (tree *binTree[E, V]) Clear() {
	if root := tree.GetRoot(); root != nil {
		root.Clear()
	}
}

func (tree *binTree[E, V]) isInitialRoot() bool {
	return tree.root.isInitialRoot()
}

// IsEmpty returns true if there are not any added nodes within this tree
func (tree *binTree[E, V]) IsEmpty() bool {
	return tree.isInitialRoot()
}

func (tree binTree[E, V]) format(state fmt.State, verb rune) {
	switch verb {
	case 's', 'v':
		_, _ = state.Write([]byte(tree.String()))
		return
	}
	// In default fmt handling (see printValue), we write all the fields of each struct inside curlies {}
	// When a pointer is encountered, the pointer is printed unless the nesting depth is 0
	// How that pointer is printed varies a lot depending on the verb and flags.
	// So, in the case of unsupported flags, let's print { rootPointer } where rootPointer is printed according to the flags and verb.
	s := flagsFromState(state, verb)
	rootStr := fmt.Sprintf(s, binTreeNodePtr[E, V](tree.root))
	bytes := make([]byte, len(rootStr)+2)
	bytes[0] = '{'
	shifted := bytes[1:]
	copy(shifted, rootStr)
	shifted[len(rootStr)] = '}'
	_, _ = state.Write(bytes)
}

// String returns a visual representation of the tree with one node per line.
// It is equivalent to calling TreeString(true)
func (tree *binTree[E, V]) String() string {
	return tree.TreeString(true)
}

// TreeString returns a visual representation of the tree with one node per line, with or without the non-added keys.
func (tree *binTree[E, V]) TreeString(withNonAddedKeys bool) string {
	return tree.GetRoot().TreeString(withNonAddedKeys, true)
}

// TreeString returns a visual representation of the tree with one node per line, with or without the non-added keys.
func (tree *binTree[E, V]) TreeStringWithCounts(withNonAddedKeys, withSizes, withMatchingAddressCounts bool) string {
	return tree.GetRoot().TreeStringWithCounts(withNonAddedKeys, withSizes, withMatchingAddressCounts)
}

func (tree *binTree[E, V]) printTree(builder *strings.Builder, inds indents, withNonAddedKeys, withSizes, withMatchingAddressCounts bool) {
	if tree == nil {
		builder.WriteString(inds.nodeIndent)
		builder.WriteString(nilString())
		builder.WriteByte('\n')
	} else {
		tree.GetRoot().printTree(builder, inds, withNonAddedKeys, withSizes, withMatchingAddressCounts)
	}
}

const treeKeyWildcard = '*'

// Produces a visual representation of the given tries joined by a single root node, with one node per line.
func treesString[E Key[E], V any](
	withNonAddedKeys,
	withSize,
	withMatchingAddressCounts bool,
	treePrinter func(tree *binTree[E, V], builder *strings.Builder, inds indents, withNonAddedKeys, withSizes, withMatchingAddressCounts bool),
	trees ...*binTree[E, V]) string {

	totalEntrySize := 0
	for _, tree := range trees {
		totalEntrySize += tree.Size()
	}
	builder := strings.Builder{}
	builder.Grow(totalEntrySize * 120) // 2 labels 60 chars each
	builder.WriteByte('\n')
	builder.WriteString(nonAddedNodeCircle)
	isEmpty := len(trees) == 0
	if !isEmpty {
		totalSize := 0
		for _, tree := range trees {
			totalSize += tree.Size()
		}
		if withNonAddedKeys && withSize {
			builder.WriteByte(' ')
			builder.WriteByte(treeKeyWildcard)
			builder.WriteString(" (")
			builder.WriteString(strconv.Itoa(totalSize))
			builder.WriteByte(')')
		}
		builder.WriteByte('\n')
		lastTreeIndex := len(trees) - 1
		for i := 0; i < lastTreeIndex; i++ {
			treePrinter(
				trees[i],
				&builder,
				indents{
					nodeIndent: leftElbow,
					subNodeInd: inBetweenElbows,
				},
				withNonAddedKeys,
				withSize,
				withMatchingAddressCounts)
		}
		treePrinter(
			trees[lastTreeIndex],
			&builder,
			indents{
				nodeIndent: rightElbow,
				subNodeInd: belowElbows,
			},
			withNonAddedKeys,
			withSize,
			withMatchingAddressCounts)
	} else {
		if withNonAddedKeys {
			builder.WriteByte(' ')
			builder.WriteByte(treeKeyWildcard)
			builder.WriteString(" (0)")
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}
