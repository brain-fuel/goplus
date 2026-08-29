package core

import "testing"

// Ordering was decidable before it was statable; these pin that the
// translation into the decider's fact shapes is the right one.
func TestDecidePropTexts(t *testing.T) {
	resolve := func(fun any) (string, bool) { return "", false }
	_ = resolve
	for _, tc := range []struct {
		op      PropOp
		a, b    string
		want    bool
		whatFor string
	}{
		{PropEq, "1+1", "2", true, "equality still decides"},
		{PropEq, "2", "3", false, "a false equality is refused"},
		{PropLe, "2", "3", true, "2 <= 3"},
		{PropLe, "3", "3", true, "<= is reflexive"},
		{PropLe, "4", "3", false, "4 is not <= 3"},
		{PropLt, "2", "3", true, "2 < 3"},
		{PropLt, "3", "3", false, "< is irreflexive"},
		{PropLt, "4", "3", false, "4 is not < 3"},
		{PropLt, "n", "n+1", true, "n < n+1 for any n"},
		{PropLe, "n", "n", true, "n <= n for any n"},
		{PropLt, "n", "n", false, "n < n is never provable"},
		{PropLe, "n+1", "n", false, "n+1 <= n is never provable"},
	} {
		got, err := DecidePropTexts(tc.op, tc.a, tc.b, nil, nil, nil)
		if err != nil {
			t.Errorf("%s %s %s: %v", tc.a, tc.op.Symbol(), tc.b, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s %s %s = %v, want %v (%s)", tc.a, tc.op.Symbol(), tc.b, got, tc.want, tc.whatFor)
		}
	}
}

func TestPropNames(t *testing.T) {
	for name, want := range map[string]PropOp{"Eq": PropEq, "Le": PropLe, "Lt": PropLt} {
		got, ok := PropFor(name)
		if !ok || got != want {
			t.Errorf("PropFor(%q) = (%v, %v)", name, got, ok)
		}
	}
	if IsProp("Vec") || IsProp("") {
		t.Error("a non-proposition name must not read as one")
	}
	if PropEq.Witness() != "refl" || PropLt.Witness() != "decide" || PropLe.Witness() != "decide" {
		t.Error("refl is reflexivity: it belongs to equality alone")
	}
}
