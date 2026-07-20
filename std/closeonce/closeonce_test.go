package closeonce

import (
	"errors"
	"sync"
	"testing"
)

func TestCloseRunsOnceAndCachesError(t *testing.T) {
	boom := errors.New("boom")
	calls := 0
	c := New(func() error { calls++; return boom })
	if err := c.Close(); !errors.Is(err, boom) {
		t.Fatal("first close error")
	}
	if err := c.Close(); !errors.Is(err, boom) {
		t.Fatal("cached error")
	}
	if calls != 1 {
		t.Fatalf("teardown ran %d times", calls)
	}
}

func TestNilReceiverAndNilFunc(t *testing.T) {
	var c *Closer
	if c.Close() != nil {
		t.Fatal("nil receiver")
	}
	if New(nil).Close() != nil {
		t.Fatal("nil teardown")
	}
}

func TestConcurrentCloseSingleTeardown(t *testing.T) {
	calls := 0
	c := New(func() error { calls++; return nil })
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Close()
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("teardown ran %d times", calls)
	}
}
