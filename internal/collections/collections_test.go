package collections

import (
	"sort"
	"testing"
)

func TestSet(t *testing.T) {
	s := NewSet[string]()

	if s.Has("a") {
		t.Fatal("empty set should not contain a")
	}
	if s.Len() != 0 {
		t.Fatalf("empty set len = %d, want 0", s.Len())
	}

	s.Add("a")
	s.Add("a") // idempotent
	s.Add("b")

	if !s.Has("a") || !s.Has("b") {
		t.Fatal("set should contain added values")
	}
	if s.Has("c") {
		t.Fatal("set should not contain c")
	}
	if s.Len() != 2 {
		t.Fatalf("set len = %d, want 2", s.Len())
	}
}

func TestSyncMapStoreLoad(t *testing.T) {
	var sm SyncMap[string, int]

	if v, ok := sm.Load("missing"); ok || v != 0 {
		t.Fatalf("Load(missing) = (%d, %v), want (0, false)", v, ok)
	}

	sm.Store("x", 42)
	v, ok := sm.Load("x")
	if !ok || v != 42 {
		t.Fatalf("Load(x) = (%d, %v), want (42, true)", v, ok)
	}
}

func TestSyncMapRange(t *testing.T) {
	var sm SyncMap[string, int]
	sm.Store("a", 1)
	sm.Store("b", 2)
	sm.Store("c", 3)

	got := make(map[string]int)
	sm.Range(func(k string, v int) bool {
		got[k] = v
		return true
	})

	if len(got) != 3 || got["a"] != 1 || got["b"] != 2 || got["c"] != 3 {
		t.Fatalf("Range collected %v, want a=1,b=2,c=3", got)
	}
}

func TestSyncMapRangeEarlyStop(t *testing.T) {
	var sm SyncMap[string, int]
	sm.Store("a", 1)
	sm.Store("b", 2)
	sm.Store("c", 3)

	var visited []string
	sm.Range(func(k string, v int) bool {
		visited = append(visited, k)
		return false // stop after first
	})

	if len(visited) != 1 {
		sort.Strings(visited)
		t.Fatalf("Range visited %v, want exactly one key", visited)
	}
}
