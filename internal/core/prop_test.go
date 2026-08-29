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

// A conjunction costs the decider nothing: it already takes a LIST of
// facts, and a conjunction is exactly that.
func TestConjunction(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
		why  string
	}{
		{"Le[0, 2]", "Lt[2, 5]", true, "both parts hold"},
		{"Le[0, 2]", "Lt[5, 2]", false, "the right part fails"},
		{"Lt[5, 2]", "Le[0, 2]", false, "the left part fails"},
		{"Le[1, 1]", "Eq[1+1, 2]", true, "mixed relations"},
		{"And[Le[0, 1], Lt[1, 2]]", "Eq[2, 2]", true, "nested conjunction"},
	} {
		got, err := DecidePropTexts(PropAnd, tc.a, tc.b, nil, nil, nil)
		if err != nil {
			t.Errorf("And[%s, %s]: %v", tc.a, tc.b, err)
			continue
		}
		if got != tc.want {
			t.Errorf("And[%s, %s] = %v, want %v (%s)", tc.a, tc.b, got, tc.want, tc.why)
		}
	}
}

// As a hypothesis, every part of a conjunction is available.
func TestConjunctionAsHypothesis(t *testing.T) {
	hyps, ok := PropFactsFor(PropAnd, "Le[1, n]", "Lt[n, 9]", nil, nil)
	if !ok || len(hyps) != 2 {
		t.Fatalf("want two facts from a conjunction, got %d (ok=%v)", len(hyps), ok)
	}
	// Each part is usable on its own.
	for _, tc := range []struct {
		op   PropOp
		a, b string
		want bool
	}{
		{PropLt, "0", "n", true},  // from 1 <= n
		{PropLe, "n", "8", true},  // from n < 9
		{PropLt, "n", "5", false}, // neither part implies it
	} {
		got, err := DecidePropUnder(hyps, tc.op, tc.a, tc.b, nil, nil, nil)
		if err != nil || got != tc.want {
			t.Errorf("under the conjunction, %s %s %s = %v, want %v", tc.a, tc.op.Symbol(), tc.b, got, tc.want)
		}
	}
}

func TestSplitPropRespectsNesting(t *testing.T) {
	name, args := SplitProp("And[Lt[0, n], Lt[n, m]]")
	if name != "And" || len(args) != 2 || args[0] != "Lt[0, n]" || args[1] != "Lt[n, m]" {
		t.Fatalf("SplitProp = %q, %#v", name, args)
	}
	if n, a := SplitProp("Eq[n, m]"); n != "Eq" || len(a) != 2 {
		t.Errorf("SplitProp on a relation = %q, %#v", n, a)
	}
	if n, _ := SplitProp("nat"); n != "" {
		t.Errorf("a non-instantiation must not split: %q", n)
	}
}

// A named proposition is an abbreviation, not a new kind of fact: it
// unfolds into the relations the decider already settles.
func TestNamedPropositionUnfolds(t *testing.T) {
	props := PropDefs{
		"InRange": {"i,n", "And[Le[0, i], Lt[i, n]]"},
		"Pos":     {"n", "Lt[0, n]"},
	}
	// A use unfolds and decides.
	facts, ok := PropTextFactsUnder(props, "InRange[2, 5]", nil, nil)
	if !ok || len(facts) != 2 {
		t.Fatalf("want two facts from InRange[2, 5], got %d (ok=%v)", len(facts), ok)
	}
	// It composes: a named proposition may name another.
	nested := PropDefs{"Both": {"a,b", "And[Pos[a], Pos[b]]"}, "Pos": {"n", "Lt[0, n]"}}
	if fs, ok := PropTextFactsUnder(nested, "Both[3, 4]", nil, nil); !ok || len(fs) != 2 {
		t.Errorf("nested named propositions: got %d facts (ok=%v)", len(fs), ok)
	}
	// Arity is checked.
	if _, ok := PropTextFactsUnder(props, "InRange[2]", nil, nil); ok {
		t.Error("wrong arity must be refused")
	}
	// An undeclared name is refused rather than assumed.
	if _, ok := PropTextFactsUnder(props, "Unknown[1, 2]", nil, nil); ok {
		t.Error("an undeclared proposition must be refused")
	}
	// Self-reference ends as a refusal, not a hang.
	loop := PropDefs{"Loop": {"n", "Loop[n]"}}
	if _, ok := PropTextFactsUnder(loop, "Loop[1]", nil, nil); ok {
		t.Error("a self-referential proposition must be refused")
	}
}

// Unfolding substitutes in ONE pass, so a parameter named like another's
// argument cannot be substituted twice.
func TestUnfoldSubstitutesOnce(t *testing.T) {
	props := PropDefs{"Swap": {"a,b", "And[Eq[a, b], Eq[b, a]]"}}
	got, ok := props.Unfold("Swap", []string{"b", "a"})
	if !ok {
		t.Fatal("Unfold refused a well-formed use")
	}
	if want := "And[Eq[b, a], Eq[a, b]]"; got != want {
		t.Errorf("Unfold = %q, want %q", got, want)
	}
}
