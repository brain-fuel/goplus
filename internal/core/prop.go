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

// PropDefs is the unfolding table for named propositions: a name maps to
// its comma-joined parameters and its body. A named proposition is not a
// new kind of fact — it is an abbreviation, so unfolding is all it needs.
type PropDefs map[string][2]string

// Unfold expands a named proposition applied to arguments into its body.
// It reports false when the name is not declared or the arity is wrong.
func (p PropDefs) Unfold(name string, args []string) (string, bool) {
	def, ok := p[name]
	if !ok {
		return "", false
	}
	var params []string
	for _, s := range strings.Split(def[0], ",") {
		if s = strings.TrimSpace(s); s != "" {
			params = append(params, s)
		}
	}
	if len(params) != len(args) {
		return "", false
	}
	body := def[1]
	// Substitute in one pass over placeholders so a parameter named like
	// another's argument cannot be re-substituted.
	sub := make(map[string]string, len(params))
	for i, name := range params {
		sub[name] = args[i]
	}
	return substIdents(body, sub), true
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
	return PropTextFactsUnder(nil, text, defs, resolve)
}

// PropTextFactsUnder reads one proposition, unfolding any named ones it
// meets. Unfolding is bounded: a name may not expand into itself.
func PropTextFactsUnder(props PropDefs, text string, defs Defs, resolve CallResolver) ([]Fact, bool) {
	return propTextFactsDepth(props, text, defs, resolve, 0)
}

func propTextFactsDepth(props PropDefs, text string, defs Defs, resolve CallResolver, depth int) ([]Fact, bool) {
	if depth > maxPropUnfold {
		return nil, false
	}
	base, args := SplitProp(text)
	if base == "" {
		return nil, false
	}
	if op, isProp := PropFor(base); isProp {
		if len(args) != 2 {
			return nil, false
		}
		if op.Nested() {
			left, okL := propTextFactsDepth(props, args[0], defs, resolve, depth+1)
			right, okR := propTextFactsDepth(props, args[1], defs, resolve, depth+1)
			if !okL || !okR {
				return nil, false
			}
			return append(left, right...), true
		}
		f, ok := PropFact(op, args[0], args[1], defs, resolve)
		if !ok {
			return nil, false
		}
		return []Fact{f}, true
	}
	// A named proposition: unfold and try again.
	body, ok := props.Unfold(base, args)
	if !ok {
		return nil, false
	}
	return propTextFactsDepth(props, body, defs, resolve, depth+1)
}

// maxPropUnfold bounds unfolding so a self-referential declaration ends
// as a refusal rather than a hang.
const maxPropUnfold = 32

// substIdents replaces whole-identifier occurrences, all in one pass.
func substIdents(s string, sub map[string]string) string {
	identByte := func(b byte) bool {
		return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if !identByte(s[i]) || (i > 0 && identByte(s[i-1])) {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && identByte(s[j]) {
			j++
		}
		if with, ok := sub[s[i:j]]; ok {
			b.WriteString(with)
		} else {
			b.WriteString(s[i:j])
		}
		i = j
	}
	return b.String()
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
