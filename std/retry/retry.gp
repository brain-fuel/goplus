// Package retry is a bounded, context-aware retry loop with exponential
// backoff: attempt, wait base·2^n capped at cap, honor ctx cancellation
// mid-wait, and surface the LAST error when attempts are exhausted — the
// loop every fetcher hand-rolls.
//
// Authored in Go+ and distributed as generated Go — consumers never need
// the goplus toolchain.
package retry

import (
	"context"
	"time"
)

// Policy bounds the loop. Attempts is the total number of tries (>= 1;
// values < 1 are treated as 1). Base is the first backoff delay; each
// subsequent delay doubles, clamped to Cap (Cap <= 0 means uncapped).
type Policy struct {
	Attempts int
	Base     time.Duration
	Cap      time.Duration
}

// Attempts returns the normalized total attempt count.
func Attempts(policy Policy) int {
	if policy.Attempts < 1 { return 1 }
	return policy.Attempts
}

// Wait pauses for delay or returns the context cancellation. Non-positive
// delays complete immediately without allocating a timer.
func Wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 { return nil }
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done(): return ctx.Err()
	case <-timer.C: return nil
	}
}

// NextDelay doubles delay and clamps it to policy.Cap. Overflow saturates
// before the cap is applied.
func NextDelay(policy Policy, delay time.Duration) time.Duration {
	if delay <= 0 { return 0 }
	maximum := time.Duration(1<<63 - 1)
	if delay > maximum/2 { delay = maximum } else { delay *= 2 }
	if policy.Cap > 0 && delay > policy.Cap { return policy.Cap }
	return delay
}

// Do runs f up to p.Attempts times. It returns f's first success, or —
// after the final failure — the zero T and the last error. Between
// attempts it sleeps the backoff delay unless ctx is done first, in which
// case it returns the zero T and ctx's error immediately. A ctx already
// done before the first attempt still gets one attempt: f observes ctx
// itself (matching the attempt-then-check shape of hand-rolled loops).
func Do[T any](ctx context.Context, p Policy, f func(context.Context) (T, error)) (T, error) {
	attempts := Attempts(p)
	var lastErr error
	delay := p.Base
	for i := 0; i < attempts; i++ {
		v, err := f(ctx)
		if err == nil {
			return v, nil
		}
		lastErr = err
		if i == attempts-1 {
			break
		}
		if err := Wait(ctx, delay); err != nil {
			var zero T
			return zero, err
		}
		delay = NextDelay(p, delay)
	}
	var zero T
	return zero, lastErr
}
