package clock

import (
	"testing"
	"testing/quick"
	"time"
)

func received(ch <-chan time.Time) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestFakeAdvanceMonotonicLaw(t *testing.T) {
	start := time.Unix(1_000, 0)
	law := func(first uint16, second uint16) bool {
		clock := NewFake(start)
		clock.Advance(time.Duration(first) * time.Millisecond)
		middle := clock.Now()
		clock.Advance(time.Duration(second) * time.Millisecond)
		return !middle.Before(start) && !clock.Now().Before(middle)
	}
	if cause := quick.Check(law, nil); cause != nil {
		t.Fatal(cause)
	}
}

func TestFakeAfterDeadlineLaw(t *testing.T) {
	law := func(raw uint16) bool {
		delay := time.Duration(raw%10_000+1) * time.Nanosecond
		clock := NewFake(time.Unix(0, 0))
		fire := clock.After(delay)
		clock.Advance(delay - time.Nanosecond)
		if received(fire) {
			return false
		}
		clock.Advance(time.Nanosecond)
		return received(fire) && clock.PendingLen() == 0
	}
	if cause := quick.Check(law, nil); cause != nil {
		t.Fatal(cause)
	}
}

func TestFakeAfterImmediateLaw(t *testing.T) {
	clock := NewFake(time.Time{})
	if !received(clock.After(0)) || !received(clock.After(-time.Second)) {
		t.Fatal("non-positive After must fire synchronously")
	}
	if clock.PendingLen() != 0 {
		t.Fatal("immediate channels must not remain pending")
	}
}

func TestFakeAfterFuncCancellationLaw(t *testing.T) {
	clock := NewFake(time.Unix(0, 0))
	fired := 0
	stop := clock.AfterFunc(time.Second, func() { fired++ })
	if clock.TimerLen() != 1 || !stop.Stop() || stop.Stop() {
		t.Fatal("pending timer must be live and stoppable exactly once")
	}
	if clock.TimerLen() != 0 {
		t.Fatal("stopped timer must not be observable as live")
	}
	clock.Advance(time.Second)
	if fired != 0 {
		t.Fatal("stopped callback fired")
	}
}

func TestFakeAfterFuncOrderingAndReentryLaw(t *testing.T) {
	clock := NewFake(time.Unix(0, 0))
	order := []int{}
	clock.AfterFunc(20*time.Millisecond, func() { order = append(order, 20) })
	clock.AfterFunc(10*time.Millisecond, func() {
		if clock.Now() != time.Unix(0, 0).Add(10*time.Millisecond) {
			t.Fatalf("callback now = %v", clock.Now())
		}
		order = append(order, 10)
		clock.AfterFunc(0, func() { order = append(order, 11) })
	})
	clock.AfterFunc(20*time.Millisecond, func() { order = append(order, 21) })
	clock.Advance(20 * time.Millisecond)
	want := []int{10, 11, 20, 21}
	if len(order) != len(want) {
		t.Fatalf("order = %v", order)
	}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("order = %v", order)
		}
	}
	if clock.Now() != time.Unix(0, 0).Add(20*time.Millisecond) {
		t.Fatalf("final now = %v", clock.Now())
	}
}

func TestFakeMixedShapeInsertionOrderLaw(t *testing.T) {
	clock := NewFake(time.Unix(0, 0))
	channel := clock.After(time.Second)
	callbackFired := false
	clock.AfterFunc(time.Second, func() {
		if !received(channel) {
			t.Fatal("earlier channel registration must fire before tied callback")
		}
		callbackFired = true
	})
	clock.Advance(time.Second)
	if !callbackFired {
		t.Fatal("callback did not fire")
	}
}

func TestFakeRejectsBackwardTime(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("negative Advance must panic")
		}
	}()
	NewFake(time.Time{}).Advance(-time.Nanosecond)
}
