package memo

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestComputeOnceAndIdentity(t *testing.T) {
	var c Cache[string, *int]
	calls := 0
	compute := func() (*int, error) { calls++; v := 42; return &v, nil }
	a, err := c.Get("k", compute)
	if err != nil || *a != 42 || calls != 1 {
		t.Fatalf("first get: %v %v %d", a, err, calls)
	}
	b, _ := c.Get("k", compute)
	if b != a {
		t.Fatal("pointer identity must hold")
	}
	if calls != 1 {
		t.Fatalf("compute ran %d times", calls)
	}
}

func TestErrorCachesNothing(t *testing.T) {
	var c Cache[string, *int]
	boom := errors.New("boom")
	fail := true
	compute := func() (*int, error) {
		if fail {
			return nil, boom
		}
		v := 7
		return &v, nil
	}
	if _, err := c.Get("k", compute); !errors.Is(err, boom) {
		t.Fatal("first error")
	}
	if _, ok := c.Peek("k"); ok {
		t.Fatal("error must cache nothing")
	}
	fail = false
	v, err := c.Get("k", compute)
	if err != nil || *v != 7 {
		t.Fatal("retry after error")
	}
}

func TestConcurrentFirstGetsConverge(t *testing.T) {
	var c Cache[int, *int]
	var computes atomic.Int32
	var wg sync.WaitGroup
	results := make([]*int, 64)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, _ := c.Get(1, func() (*int, error) {
				computes.Add(1)
				n := 5
				return &n, nil
			})
			results[i] = v
		}(i)
	}
	wg.Wait()
	for _, r := range results {
		if r != results[0] {
			t.Fatal("all callers must observe the winning value")
		}
	}
	if c.Len() != 1 {
		t.Fatalf("len = %d", c.Len())
	}
}
