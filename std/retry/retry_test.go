package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFirstSuccessNoRetry(t *testing.T) {
	calls := 0
	v, err := Do(context.Background(), Policy{Attempts: 3, Base: time.Hour},
		func(context.Context) (int, error) { calls++; return 42, nil })
	if err != nil || v != 42 || calls != 1 {
		t.Fatalf("v=%d err=%v calls=%d", v, err, calls)
	}
}

func TestExhaustionReturnsLastError(t *testing.T) {
	e1, e2, e3 := errors.New("1"), errors.New("2"), errors.New("3")
	errs := []error{e1, e2, e3}
	calls := 0
	_, err := Do(context.Background(), Policy{Attempts: 3, Base: time.Microsecond},
		func(context.Context) (int, error) { calls++; return 0, errs[calls-1] })
	if calls != 3 || !errors.Is(err, e3) {
		t.Fatalf("calls=%d err=%v (want last error)", calls, err)
	}
}

func TestSucceedsMidway(t *testing.T) {
	calls := 0
	v, err := Do(context.Background(), Policy{Attempts: 5, Base: time.Microsecond},
		func(context.Context) (string, error) {
			calls++
			if calls < 3 {
				return "", errors.New("not yet")
			}
			return "ok", nil
		})
	if err != nil || v != "ok" || calls != 3 {
		t.Fatalf("v=%q err=%v calls=%d", v, err, calls)
	}
}

func TestCtxCancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	start := time.Now()
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := Do(ctx, Policy{Attempts: 10, Base: time.Hour},
		func(context.Context) (int, error) { calls++; return 0, errors.New("x") })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d (cancel must land in the first backoff)", calls)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("did not honor cancel promptly")
	}
}

func TestAttemptsClampedToOne(t *testing.T) {
	calls := 0
	_, err := Do(context.Background(), Policy{Attempts: 0},
		func(context.Context) (int, error) { calls++; return 0, errors.New("x") })
	if calls != 1 || err == nil {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestBackoffDoublesAndCaps(t *testing.T) {
	// Observe delays indirectly: Base=1ms, Cap=2ms, 4 attempts → waits 1,2,2ms.
	start := time.Now()
	calls := 0
	_, _ = Do(context.Background(), Policy{Attempts: 4, Base: time.Millisecond, Cap: 2 * time.Millisecond},
		func(context.Context) (int, error) { calls++; return 0, errors.New("x") })
	elapsed := time.Since(start)
	if calls != 4 {
		t.Fatalf("calls=%d", calls)
	}
	if elapsed < 5*time.Millisecond {
		t.Fatalf("elapsed %v: backoff waits not applied", elapsed)
	}
}

func TestPolicyPrimitives(t *testing.T) {
	if got := Attempts(Policy{}); got != 1 {
		t.Fatalf("Attempts = %d, want 1", got)
	}
	if got := NextDelay(Policy{Cap: 3 * time.Second}, 2*time.Second); got != 3*time.Second {
		t.Fatalf("capped delay = %v", got)
	}
	maximum := time.Duration(1<<63 - 1)
	if got := NextDelay(Policy{}, maximum); got != maximum {
		t.Fatalf("overflow delay = %v, want saturation", got)
	}
	if allocations := testing.AllocsPerRun(1000, func() {
		if err := Wait(context.Background(), 0); err != nil {
			panic(err)
		}
	}); allocations != 0 {
		t.Fatalf("zero-delay Wait allocations = %.1f", allocations)
	}
}
