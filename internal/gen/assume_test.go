package gen

import "testing"

// The nearest token by LINE is what anchors an assumption: once a line's
// content changes, its column moves too, so requiring the column to match
// would find nothing exactly when remapping is needed.
func TestNearestAssumeToken(t *testing.T) {
	tokens := []assumeToken{{line: 4, col: 20}, {line: 9, col: 31}}
	for _, tc := range []struct {
		want int
		line int
	}{{3, 4}, {4, 4}, {5, 4}, {7, 9}, {9, 9}, {40, 9}} {
		got, ok := nearestAssumeToken(tokens, tc.want)
		if !ok || got.line != tc.line {
			t.Errorf("nearestAssumeToken(want=%d) = (%+v, %v), want line %d", tc.want, got, ok, tc.line)
		}
	}
	if _, ok := nearestAssumeToken(nil, 3); ok {
		t.Error("no tokens must report no match rather than inventing one")
	}
}

// Two assumes at the same column must not both anchor on the first one:
// the source's own tokens are the ground truth, in order.
func TestAssumeTokens(t *testing.T) {
	src := []byte("package p\n\nfunc a() { return Cast(2, 3, assume, v) }\nfunc b() { return Cast(5, 7, assume, v) }\n")
	got := assumeTokens(src)
	want := []assumeToken{{line: 3, col: 30}, {line: 4, col: 30}}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// An identifier that merely contains "assume" is not the token.
	for _, s := range []string{"x := assumed\n", "x := reassume\n", "x := assume_1\n"} {
		if ts := assumeTokens([]byte(s)); len(ts) != 0 {
			t.Errorf("%q matched %+v, want none", s, ts)
		}
	}
	if ts := assumeTokens(nil); ts != nil {
		t.Errorf("empty source yielded %+v", ts)
	}
}
