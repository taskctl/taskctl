package collections

import "sync"

// noCopy is used to help go vet detect unintended copies of types that must not
// be copied after first use.
type noCopy struct{}

func (*noCopy) Lock() {}

// SyncMap is a type-safe wrapper around sync.Map. Like sync.Map it must not be
// copied after first use; use it via a pointer or as a non-copied struct field.
type SyncMap[K comparable, V any] struct {
	noCopy noCopy
	m      sync.Map
}

// Store sets the value for a key.
func (sm *SyncMap[K, V]) Store(k K, v V) {
	sm.m.Store(k, v)
}

// Load returns the value stored for a key. The second result reports whether the
// key was present; on miss the zero value of V is returned.
func (sm *SyncMap[K, V]) Load(k K) (V, bool) {
	v, ok := sm.m.Load(k)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// Range calls f for each key/value pair in the map, stopping if f returns false.
func (sm *SyncMap[K, V]) Range(f func(k K, v V) bool) {
	sm.m.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}

// Snapshot returns a plain map copy of the current key/value pairs.
func (sm *SyncMap[K, V]) Snapshot() map[K]V {
	m := make(map[K]V)
	sm.Range(func(k K, v V) bool {
		m[k] = v
		return true
	})

	return m
}
