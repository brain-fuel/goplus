package guarded

import (
	"sync"
	"testing"
)

func TestGuardedWithGetSet(t *testing.T) {
	g := New(10)
	if g.Get() != 10 {
		t.Fatal("initial")
	}
	g.With(func(v *int) { *v += 5 })
	if g.Get() != 15 {
		t.Fatal("with-mutation")
	}
	g.Set(1)
	if g.Get() != 1 {
		t.Fatal("set")
	}
}

func TestGuardedZeroValueUsable(t *testing.T) {
	var g Guarded[[]string]
	g.With(func(v *[]string) { *v = append(*v, "a") })
	if got := g.Get(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("zero-value cell: %v", got)
	}
}

func TestGuardedConcurrentIncrements(t *testing.T) {
	g := New(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				g.With(func(v *int) { *v++ })
			}
		}()
	}
	wg.Wait()
	if g.Get() != 10000 {
		t.Fatalf("lost increments: %d", g.Get())
	}
}

func TestRWGuardedReadersAndWriters(t *testing.T) {
	g := NewRW(map[string]int{"a": 1})
	g.With(func(m *map[string]int) { (*m)["b"] = 2 })
	sum := 0
	g.RWith(func(m map[string]int) {
		for _, v := range m {
			sum += v
		}
	})
	if sum != 3 {
		t.Fatalf("sum = %d", sum)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			g.RWith(func(m map[string]int) { _ = m["a"] })
		}()
		go func() {
			defer wg.Done()
			g.With(func(m *map[string]int) { (*m)["a"]++ })
		}()
	}
	wg.Wait()
	if got := g.Get()["a"]; got != 51 {
		t.Fatalf("a = %d", got)
	}
}
