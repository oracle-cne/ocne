// Copyright (c) 2025, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

// Set is a generic set implementation using Go generics.
// It supports fast membership tests and population from slices and map keys.
type Set[T comparable] struct {
    items map[T]struct{}
}

// NewSetFromSlice creates a set populated from the elements of a slice.
func NewSetFromSlice[T comparable](elems []T) *Set[T] {
    set := &Set[T]{items: make(map[T]struct{}, len(elems))}
    for _, elem := range elems {
        set.items[elem] = struct{}{}
    }
    return set
}

// NewSetFromMapKeys creates a set populated from the keys of the given map.
func NewSetFromMapKeys[T comparable, V any](m map[T]V) *Set[T] {
    set := &Set[T]{items: make(map[T]struct{}, len(m))}
    for key := range m {
        set.items[key] = struct{}{}
    }
    return set
}

// Add inserts an element into the set.
func (s *Set[T]) Add(elem T) {
    s.items[elem] = struct{}{}
}

// Contains tests if an element is in the set.
func (s *Set[T]) Contains(elem T) bool {
    _, exists := s.items[elem]
    return exists
}

// Remove deletes an element from the set.
func (s *Set[T]) Remove(elem T) {
    delete(s.items, elem)
}

// Size returns the number of elements in the set.
func (s *Set[T]) Size() int {
    return len(s.items)
}
