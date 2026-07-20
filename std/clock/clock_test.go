package clock

import (
	"testing"
	"time"
)

func recvd(ch <-chan time.Time) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestFakeAfterFiresInDeadlineOrderTiesByInsertion(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	a := c.After(20 * time.Millisecond) // seq 0
	b := c.After(10 * time.Millisecond) // seq 1
	d := c.After(20 * time.Millisecond) // seq 2, tie with a
	if c.PendingLen() != 3 {
		t.Fatalf("PendingLen = %d", c.PendingLen())
	}
	c.Advance(20 * time.Millisecond)
	if c.PendingLen() != 0 {
		t.Fatalf("PendingLen after advance = %d", c.PendingLen())
	}
	for i, ch := range []<-chan time.Time{a, b, d} {
		if !recvd(ch) {
			t.Fatalf("channel %d did not fire", i)
		}
	}
}

func TestFakeAfterZeroFiresSynchronously(t *testing.T) {
	c := NewFake(time.Time{})
	if !recvd(c.After(0)) {
		t.Fatal("After(0) must fire before Advance")
	}
	if !recvd(c.After(-time.Second)) {
		t.Fatal("negative After must fire before Advance")
	}
	if c.PendingLen() != 0 {
		t.Fatal("immediate fires must not pend")
	}
}

func TestFakeAfterNotDueDoesNotFire(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	ch := c.After(50 * time.Millisecond)
	c.Advance(49 * time.Millisecond)
	if recvd(ch) {
		t.Fatal("fired early")
	}
	c.Advance(time.Millisecond)
	if !recvd(ch) {
		t.Fatal("did not fire at deadline")
	}
}

func TestFakeSnapshotOnce(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	first := c.After(10 * time.Millisecond)
	c.Advance(10 * time.Millisecond)
	if !recvd(first) {
		t.Fatal("first did not fire")
	}
	second := c.After(5 * time.Millisecond) // registered after the advance
	if recvd(second) {
		t.Fatal("second fired retroactively")
	}
	c.Advance(5 * time.Millisecond)
	if !recvd(second) {
		t.Fatal("second did not fire on its own advance")
	}
}

func TestFakeAfterFuncStopSemantics(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	fired := 0
	h := c.AfterFunc(10*time.Millisecond, func() { fired++ })
	c.Advance(10 * time.Millisecond)
	if fired != 1 {
		t.Fatalf("fired = %d", fired)
	}
	if h.Stop() {
		t.Fatal("Stop after fire must report false")
	}

	fired2 := 0
	h2 := c.AfterFunc(10*time.Millisecond, func() { fired2++ })
	if !h2.Stop() {
		t.Fatal("first Stop must report true")
	}
	if h2.Stop() {
		t.Fatal("second Stop must report false")
	}
	c.Advance(20 * time.Millisecond)
	if fired2 != 0 {
		t.Fatal("stopped timer fired")
	}
}

func TestFakeDrainLoopRearmFiresSameAdvance(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	var order []int
	c.AfterFunc(10*time.Millisecond, func() {
		order = append(order, 1)
		c.AfterFunc(0, func() { order = append(order, 2) })
	})
	c.Advance(10 * time.Millisecond)
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("order = %v (zero-delay re-arm must fire in the same pass)", order)
	}
}

func TestFakeCallbackOrderingMatchesChannelBlock(t *testing.T) {
	c := NewFake(time.Unix(0, 0))
	var order []int
	c.AfterFunc(20*time.Millisecond, func() { order = append(order, 20) })
	c.AfterFunc(10*time.Millisecond, func() { order = append(order, 10) })
	c.AfterFunc(20*time.Millisecond, func() { order = append(order, 21) })
	c.Advance(20 * time.Millisecond)
	if len(order) != 3 || order[0] != 10 || order[1] != 20 || order[2] != 21 {
		t.Fatalf("order = %v", order)
	}
}

func TestRealClockFires(t *testing.T) {
	var r Real
	if !r.Now().After(time.Time{}) {
		t.Fatal("Now")
	}
	select {
	case <-r.After(time.Millisecond):
	case <-time.After(2 * time.Second):
		t.Fatal("Real.After did not fire")
	}
	done := make(chan struct{})
	r.AfterFunc(time.Millisecond, func() { close(done) })
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Real.AfterFunc did not fire")
	}
	stopped := r.AfterFunc(time.Hour, func() {})
	if !stopped.Stop() {
		t.Fatal("Stop on pending real timer must report true")
	}
}
