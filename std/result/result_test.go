package result

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

func TestRailway(t *testing.T) {
	boom := errors.New("boom")

	// Of enters the railway from Go pairs.
	if got := Of(strconv.Atoi("42")); !IsOk[int, error](got) {
		t.Fatalf("Of ok: %v", got)
	}
	if got := Of(strconv.Atoi("nope")); !IsErr[int, error](got) {
		t.Fatalf("Of err: %v", got)
	}

	// Bind applies switches; Err bypasses.
	half := func(n int) Result[int, error] {
		if n%2 != 0 {
			return Err[int, error]{Err: boom}
		}
		return Ok[int, error]{Value: n / 2}
	}
	if got := Bind(Ok[int, error]{Value: 8}, half); got != (Ok[int, error]{Value: 4}) {
		t.Fatalf("Bind: %v", got)
	}
	touched := false
	if got := Bind(Err[int, error]{Err: boom}, func(n int) Result[int, error] {
		touched = true
		return Ok[int, error]{Value: n}
	}); got != (Err[int, error]{Err: boom}) || touched {
		t.Fatalf("Bind bypass: %v touched=%v", got, touched)
	}

	// Map lifts plain functions; Tee runs Ok-only side effects.
	if got := Map(Ok[int, error]{Value: 21}, func(n int) string { return fmt.Sprint(n * 2) }); got != (Ok[string, error]{Value: "42"}) {
		t.Fatalf("Map: %v", got)
	}
	seen := 0
	Tee(Ok[int, error]{Value: 1}, func(int) { seen++ })
	Tee(Err[int, error]{Err: boom}, func(int) { seen++ })
	if seen != 1 {
		t.Fatalf("Tee: %d", seen)
	}

	// MapError transforms the failure track with typed errors.
	wrapped := MapError(Err[int, error]{Err: boom}, func(e error) *strconv.NumError {
		return &strconv.NumError{Func: "x", Num: "y", Err: e}
	})
	if !IsErr[int, *strconv.NumError](wrapped) {
		t.Fatalf("MapError: %v", wrapped)
	}

	// Unpack leaves the railway; UnwrapOr collapses it.
	if v, err := Unpack(Ok[int, error]{Value: 7}); v != 7 || err != nil {
		t.Fatalf("Unpack ok: %v %v", v, err)
	}
	if v, err := Unpack(Err[int, error]{Err: boom}); v != 0 || err != boom {
		t.Fatalf("Unpack err: %v %v", v, err)
	}
	if got := UnwrapOr(Err[int, error]{Err: boom}, 9); got != 9 {
		t.Fatalf("UnwrapOr: %d", got)
	}
}
