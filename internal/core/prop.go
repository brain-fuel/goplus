package core

import (
	"math/big"
	"strings"
)

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
	PropEq  PropOp = iota // Eq[a, b] — a equals b
	PropLe                // Le[a, b] — a is at most b
	PropLt                // Lt[a, b] — a is strictly below b
	PropAnd               // And[P, Q] — both propositions hold
)

// propNames maps each proposition's type name to its relation.
var propNames = map[string]PropOp{"Eq": PropEq, "Le": PropLe, "Lt": PropLt, "And": PropAnd}

// Nested reports whether the proposition's arguments are themselves
// propositions rather than index terms.
func (op PropOp) Nested() bool { return op == PropAnd }

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
	case PropAnd:
		return "and"
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

// PropFactsFor builds the decider facts a proposition asserts. A relation
// contributes one; a conjunction contributes its parts, which is what
// makes `And` cost nothing at the decider — the solver already takes a
// list of facts, and a conjunction is exactly that.
func PropFactsFor(op PropOp, aText, bText string, defs Defs, resolve CallResolver) ([]Fact, bool) {
	if op.Nested() {
		left, okL := propTextFacts(aText, defs, resolve)
		right, okR := propTextFacts(bText, defs, resolve)
		if !okL || !okR {
			return nil, false
		}
		return append(left, right...), true
	}
	f, ok := PropFact(op, aText, bText, defs, resolve)
	if !ok {
		return nil, false
	}
	return []Fact{f}, true
}

// propTextFacts reads one proposition written as a type text.
func propTextFacts(text string, defs Defs, resolve CallResolver) ([]Fact, bool) {
	base, args := SplitProp(text)
	op, isProp := PropFor(base)
	if !isProp || len(args) != 2 {
		return nil, false
	}
	return PropFactsFor(op, args[0], args[1], defs, resolve)
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

// SplitProp splits a proposition type text into its name and arguments,
// respecting nesting so `And[Lt[0, n], Lt[n, m]]` yields two arguments
// rather than three.
func SplitProp(text string) (string, []string) {
	text = strings.TrimSpace(text)
	open := strings.IndexByte(text, '[')
	if open <= 0 || !strings.HasSuffix(text, "]") {
		return "", nil
	}
	name := strings.TrimSpace(text[:open])
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		name = name[i+1:]
	}
	var args []string
	depth, start := 0, 0
	inner := text[open+1 : len(text)-1]
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(inner[start:]))
	return name, args
}
