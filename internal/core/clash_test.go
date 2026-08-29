package core

import "testing"

// A proposition in scope can rule out a variant that sign analysis alone
// cannot: `Vec[T, n]` may be `Nil : Vec[T, 0]` for all the shape of `n`
// says, but under a bound saying n is positive it never can.
func TestIndexClashUnderHypotheses(t *testing.T) {
	bound := func(op PropOp, a, b string) Fact {
		f, ok := PropFact(op, a, b, nil, nil)
		if !ok {
			t.Fatalf("PropFact(%s %s %s)", a, op.Symbol(), b)
		}
		return f
	}
	for _, tc := range []struct {
		name      string
		hyps      []Fact
		use, vari string
		want      bool
	}{
		{"no hypothesis leaves n and 0 undecided", nil, "n", "0", false},
		{"0 < n rules out the zero variant", []Fact{bound(PropLt, "0", "n")}, "n", "0", true},
		{"1 <= n rules it out too", []Fact{bound(PropLe, "1", "n")}, "n", "0", true},
		{"0 <= n does not: n may still be zero", []Fact{bound(PropLe, "0", "n")}, "n", "0", false},
		{"a bound on an unrelated index proves nothing", []Fact{bound(PropLt, "0", "m")}, "n", "0", false},
		{"n+1 never equals 0 without any hypothesis", nil, "n+1", "0", true},
		{"n < 3 rules out the 5 variant", []Fact{bound(PropLt, "n", "3")}, "n", "5", true},
		{"n < 3 does not rule out the 2 variant", []Fact{bound(PropLt, "n", "3")}, "n", "2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IndexClashUnder(tc.hyps, tc.use, tc.vari, nil); got != tc.want {
				t.Errorf("IndexClashUnder(%s vs %s) = %v, want %v", tc.use, tc.vari, got, tc.want)
			}
		})
	}
	// The hypothesis-free entry point keeps its old behaviour exactly.
	if IndexClash("n", "0", nil) {
		t.Error("IndexClash must stay conservative with no hypotheses")
	}
	if !IndexClash("n+1", "0", nil) {
		t.Error("IndexClash lost a clash it used to find")
	}
}
