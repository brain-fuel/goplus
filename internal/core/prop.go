package core

import "math/big"

// Propositions (v0.151.0). A proof parameter carries a proposition about
// index terms, erased at quantity 0 and discharged before erasure.
//
// `Eq` shipped first because equality is what a cast needs. But the more
// common obligation is a BOUND — an index below a length — and the
// decider already settles those: FactGe sits beside FactEq, and MkGe
// beside MkEq. Ordering was decidable long before it was statable; these
// names are what make it sayable.

// PropOp is the relation a proposition asserts between its two terms.
type PropOp int

const (
	PropEq PropOp = iota // Eq[a, b] — a equals b
	PropLe               // Le[a, b] — a is at most b
	PropLt               // Lt[a, b] — a is strictly below b
)

// propNames maps each proposition's type name to its relation.
var propNames = map[string]PropOp{"Eq": PropEq, "Le": PropLe, "Lt": PropLt}

// PropFor reports the relation a proposition type name asserts.
func PropFor(name string) (PropOp, bool) {
	op, ok := propNames[name]
	return op, ok
}

// IsProp reports whether name is a proposition type.
func IsProp(name string) bool {
	_, ok := propNames[name]
	return ok
}

// Symbol renders the relation as it reads in a diagnostic.
func (op PropOp) Symbol() string {
	switch op {
	case PropLe:
		return "<="
	case PropLt:
		return "<"
	default:
		return "="
	}
}

// Witness names the proof term that discharges this proposition by
// decision. `refl` is reflexivity, which is true of equality and
// meaningless for a strict inequality, so ordering is discharged by
// `decide` — which also works for equality.
func (op PropOp) Witness() string {
	if op == PropEq {
		return "refl"
	}
	return "decide"
}

// fact builds the decider goal for this relation over two values.
func (op PropOp) fact(a, b Value) Fact {
	switch op {
	case PropLe:
		return MkGe(b, a) // a <= b is b >= a
	case PropLt:
		// a < b is b >= a+1. Indices are naturals, so the step is exact
		// and no strict-inequality fact shape is needed.
		return Fact{Op: FactGe, L: linAdd(linAdd(asLin(b), asLin(a), -1), linConst(big.NewInt(1)), -1)}
	default:
		return MkEq(a, b)
	}
}
