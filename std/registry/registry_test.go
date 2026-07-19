package registry

import (
	"sort"
	"sync"
	"testing"
)

func TestRegisterLookupFreeze(t *testing.T) {
	r := New[int]()
	r.Register("a", 1)
	r.Register("b", 2)
	if v, ok := r.Lookup("a"); !ok || v != 1 {
		t.Fatal("pre-freeze lookup")
	}
	if r.Frozen() {
		t.Fatal("not frozen yet")
	}
	r.Freeze()
	r.Freeze() // idempotent
	if !r.Frozen() {
		t.Fatal("frozen")
	}
	if v, ok := r.Lookup("b"); !ok || v != 2 {
		t.Fatal("post-freeze lookup")
	}
	if _, ok := r.Lookup("missing"); ok {
		t.Fatal("missing must miss")
	}
	names := r.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("names = %v", names)
	}
}

func TestRegisterAfterFreezePanics(t *testing.T) {
	r := New[string]()
	r.Freeze()
	defer func() {
		if recover() == nil {
			t.Fatal("post-freeze Register must panic")
		}
	}()
	r.Register("late", "x")
}

func TestDuplicateRegisterPanics(t *testing.T) {
	r := New[string]()
	r.Register("dup", "x")
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate Register must panic")
		}
	}()
	r.Register("dup", "y")
}

func TestZeroValueUsable(t *testing.T) {
	var r Registry[int]
	r.Register("z", 9)
	r.Freeze()
	if v, ok := r.Lookup("z"); !ok || v != 9 {
		t.Fatal("zero-value registry")
	}
}

func TestConcurrentPostFreezeLookups(t *testing.T) {
	r := New[int]()
	for i, n := range []string{"a", "b", "c", "d"} {
		r.Register(n, i)
	}
	r.Freeze()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				if v, ok := r.Lookup("c"); !ok || v != 2 {
					panic("bad lookup")
				}
			}
		}()
	}
	wg.Wait()
}
