package deepmap

import "testing"

func TestGetSetRoundtripAndOverwrite(t *testing.T) {
	m := New[string, string, string]()
	if _, ok := m.Get("f", "k"); ok {
		t.Fatal("empty map must miss")
	}
	m.Set("f", "k", "v1")
	m.Set("f", "k", "v2")
	if v, ok := m.Get("f", "k"); !ok || v != "v2" {
		t.Fatalf("got %q,%v", v, ok)
	}
	if m.Len() != 1 {
		t.Fatalf("len = %d", m.Len())
	}
}

func TestCrossOuterKeyIndependence(t *testing.T) {
	m := New[string, string, int]()
	m.Set("a", "shared", 1)
	m.Set("b", "shared", 2)
	m.Set("a", "shared", 10)
	if v, _ := m.Get("b", "shared"); v != 2 {
		t.Fatalf("b leaked: %d", v)
	}
}

func TestSnapshotDefensiveCopy(t *testing.T) {
	m := New[string, string, string]()
	m.Set("a", "x", "ax")
	m.Set("b", "z", "bz")
	snap := m.Snapshot()
	delete(snap, "a")
	snap["b"]["injected"] = "iv"
	if _, ok := m.Get("a", "x"); !ok {
		t.Fatal("outer snapshot mutation leaked")
	}
	if _, ok := m.Get("b", "injected"); ok {
		t.Fatal("inner snapshot mutation leaked")
	}
}

func TestDeleteDropsEmptyInner(t *testing.T) {
	m := New[string, string, int]()
	m.Set("a", "x", 1)
	m.Delete("a", "x")
	if m.Len() != 0 {
		t.Fatal("delete")
	}
	if snap := m.Snapshot(); len(snap) != 0 {
		t.Fatalf("inner map retained: %v", snap)
	}
	m.Delete("a", "never") // absent delete is a no-op
}

func TestResetKeepsMapUsable(t *testing.T) {
	m := New[string, string, int]()
	m.Set("a", "x", 1)
	m.Reset()
	if m.Len() != 0 {
		t.Fatal("reset")
	}
	m.Set("b", "y", 2)
	if v, ok := m.Get("b", "y"); !ok || v != 2 {
		t.Fatal("post-reset set/get")
	}
}

func TestNilReceiverTolerance(t *testing.T) {
	var m *Map2[string, string, int]
	if v, ok := m.Get("a", "b"); ok || v != 0 {
		t.Fatal("nil get")
	}
	m.Set("a", "b", 1) // no-op, no panic
	m.Delete("a", "b") // no-op
	m.Reset()          // no-op
	if m.Len() != 0 || m.Snapshot() != nil {
		t.Fatal("nil len/snapshot")
	}
}
