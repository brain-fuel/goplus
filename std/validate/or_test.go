package validate

import "testing"

// Disjunction (Or) is the ∨ dual of And. These pin its acceptance semantics,
// its both-branches-on-total-failure diagnostics, and that its dependent index
// stays collision-free against conjunction over the same operands.

const (
	positivePredicate = 10
	evenPredicate     = 11
)

func positiveRule() Rule[int] {
	return Atom[int](positivePredicate, "positive", "", func(value int) bool { return value > 0 })
}

func evenRule() Rule[int] {
	return Atom[int](evenPredicate, "even", "", func(value int) bool { return value%2 == 0 })
}

func TestOrAcceptsWhenEitherHolds(t *testing.T) {
	rule := Or(positiveRule(), evenRule())
	// positive-only, even-only, and both all satisfy the disjunction.
	for _, value := range []int{3, -4, 6} {
		if !IsValid(rule, value) {
			t.Fatalf("Or rejected %d that satisfies one arm", value)
		}
		if failures := Check(rule, value); len(failures) != 0 {
			t.Fatalf("Or(%d) surfaced failures despite an arm holding: %v", value, failures)
		}
	}
}

func TestOrReportsBothArmsOnTotalFailure(t *testing.T) {
	rule := Or(positiveRule(), evenRule())
	// -3 is neither positive nor even: both arms must be reported, in order.
	failures := Check(rule, -3)
	if len(failures) != 2 || failures[0].Code != "positive" || failures[1].Code != "even" {
		t.Fatalf("Or(-3) failures = %#v, want [positive even]", failures)
	}
	if IsValid(rule, -3) {
		t.Fatal("Or(-3) must be invalid")
	}
}

func TestOrWitnessSurvivesAndRevalidates(t *testing.T) {
	rule := Or(positiveRule(), evenRule())
	accepted := Validate(rule, 6)
	if failures := FailuresOf(accepted); failures != nil {
		t.Fatalf("Or witness rejected valid value: %v", failures)
	}
	value := accepted.(Accepted[int]).Value
	if Value(value) != 6 || !Revalidate(rule, value) {
		t.Fatal("disjunction witness did not retain value/rule")
	}
	// A conjunction witness over the same operands must not cross-validate the
	// disjunction witness: distinct connective ⇒ distinct erased index.
	conj := And(positiveRule(), evenRule())
	if Revalidate(conj, value) {
		t.Fatal("And and Or witnesses collided over the same operands")
	}
}

func TestEitherEncodingDisjointFromBoth(t *testing.T) {
	// The mod-4 partition must keep ∧, ∨, and option refinements over identical
	// operands pairwise apart, and all apart from atoms.
	or := PredicateOfRule(Or(positiveRule(), evenRule()))
	and := PredicateOfRule(And(positiveRule(), evenRule()))
	opt := PredicateOfRule(Nullable(positiveRule()))
	if PredicateEqual(or, and) {
		t.Fatal("Either and Both predicates compared equal")
	}
	if PredicateEqual(or, PredicateOfRule(positiveRule())) {
		t.Fatal("Either predicate collided with an atom")
	}
	if PredicateEqual(opt, PredicateOfRule(positiveRule())) {
		t.Fatal("Optional predicate collided with its inner atom")
	}
	if PredicateEqual(opt, and) || PredicateEqual(opt, or) {
		t.Fatal("Optional predicate collided with a binary connective")
	}
}

func TestNullableAbsentIsVacuouslyValid(t *testing.T) {
	rule := Nullable(positiveRule())
	if !IsValid(rule, nil) {
		t.Fatal("nil must satisfy a nullable rule vacuously")
	}
	if failures := Check(rule, nil); len(failures) != 0 {
		t.Fatalf("nil nullable surfaced failures: %v", failures)
	}
}

func TestNullablePresentReportsInnerFailuresOnly(t *testing.T) {
	rule := Nullable(positiveRule())
	// A present, valid value passes; the nil-guard must not leak a failure.
	ok := 5
	if !IsValid(rule, &ok) {
		t.Fatal("present valid value must satisfy nullable rule")
	}
	// A present, invalid value reports ONLY the inner failure — never a
	// spurious "should be absent". This is the nil-biased-disjunction contract
	// that separates Nullable from a symmetric Or.
	bad := -1
	failures := Check(rule, &bad)
	if len(failures) != 1 || failures[0].Code != "positive" {
		t.Fatalf("nullable(present-invalid) failures = %#v, want exactly [positive]", failures)
	}
}
