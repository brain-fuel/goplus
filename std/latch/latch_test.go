package latch

import (
	"sync"
	"testing"
	"time"
)

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestTripWithZeroInflightFiresImmediately(t *testing.T) {
	l := New()
	if l.Tripped() || closed(l.Done()) {
		t.Fatal("fresh latch must be untripped and open")
	}
	l.Trip()
	if !l.Tripped() || !closed(l.Done()) {
		t.Fatal("trip at zero inflight must fire immediately")
	}
}

func TestTripIdempotent(t *testing.T) {
	l := New()
	l.Trip()
	l.Trip() // second trip must not panic (double close) or change state
	if !closed(l.Done()) {
		t.Fatal("done must stay closed")
	}
}

func TestRendezvousWaitsForInflight(t *testing.T) {
	l := New()
	l.Inc()
	l.Inc()
	l.Trip()
	if closed(l.Done()) {
		t.Fatal("must not fire with inflight > 0")
	}
	l.Dec()
	if closed(l.Done()) {
		t.Fatal("must not fire with inflight == 1")
	}
	l.Dec()
	if !closed(l.Done()) {
		t.Fatal("must fire when inflight reaches 0 after trip")
	}
}

func TestDecWithoutTripDoesNotFire(t *testing.T) {
	l := New()
	l.Inc()
	l.Dec()
	if closed(l.Done()) {
		t.Fatal("untripped latch must not fire")
	}
	l.Trip()
	if !closed(l.Done()) {
		t.Fatal("trip after quiescence must fire immediately")
	}
}

func TestConcurrentWorkersRendezvous(t *testing.T) {
	l := New()
	const n = 64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		l.Inc()
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond)
			l.Dec()
		}()
	}
	l.Trip()
	select {
	case <-l.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("rendezvous did not fire")
	}
	wg.Wait()
	if got := l.inflight.Load(); got != 0 {
		t.Fatalf("inflight = %d after rendezvous", got)
	}
}
