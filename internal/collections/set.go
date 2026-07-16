// Package collections provides small generic container types.
package collections

// Set is a generic set of comparable values. It is not safe for concurrent use.
type Set[T comparable] struct {
	m map[T]struct{}
}

// NewSet creates an empty Set.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{m: make(map[T]struct{})}
}

// Add inserts v into the set.
func (s *Set[T]) Add(v T) {
	s.m[v] = struct{}{}
}

// Has reports whether v is present in the set.
func (s *Set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	return len(s.m)
}
