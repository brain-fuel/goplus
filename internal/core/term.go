package core

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Term is the first-order dependent core: nat arithmetic, constructor
// data (enum tags and structured first-order values), calls to total
// functions, guarded conditionals, and one-level structural match.
// There are no term-level binders except match arms — total functions
// are top-level definitions — which keeps substitution and NbE simple.
type Term interface {
	isTerm()
	String() string
}

// Var references a parameter or match binder.
type Var struct{ Name string }

// Nat is a natural-number literal (arbitrary precision).
type Nat struct{ N *big.Int }

// Prim is built-in nat arithmetic: "+", "*", or guarded "-" (v - k,
// admissible only where the decider proves v ≥ k).
type Prim struct {
	Op   string
	Args []Term
}

// Ctor builds first-order data: an enum tag (no args) or a structured
// value. Type is the defining enum's name.
type Ctor struct {
	Type string
	Name string
	Args []Term
}

// Call invokes a total function.
type Call struct {
	Fn   string
	Args []Term
}

// If branches on a nat comparison: Op one of == != < <= > >=.
type If struct {
	Op         string
	L, R       Term
	Then, Else Term
}

// MatchT branches on first-order data one level deep. Arms are keyed by
// constructor name; a nil Body map entry is absent coverage (checked by
// the elaborator, not here).
type MatchT struct {
	Scrut Term
	Arms  []MatchArm
}

// MatchArm is one constructor case with its field binders.
type MatchArm struct {
	Ctor  string
	Binds []string
	Body  Term
}

func (Var) isTerm()    {}
func (Nat) isTerm()    {}
func (Prim) isTerm()   {}
func (Ctor) isTerm()   {}
func (Call) isTerm()   {}
func (If) isTerm()     {}
func (MatchT) isTerm() {}

func (t Var) String() string { return t.Name }
func (t Nat) String() string { return t.N.String() }
func (t Prim) String() string {
	parts := make([]string, len(t.Args))
	for i, a := range t.Args {
		parts[i] = a.String()
	}
	return "(" + strings.Join(parts, " "+t.Op+" ") + ")"
}
func (t Ctor) String() string {
	if len(t.Args) == 0 {
		return t.Name
	}
	parts := make([]string, len(t.Args))
	for i, a := range t.Args {
		parts[i] = a.String()
	}
	return t.Name + "(" + strings.Join(parts, ", ") + ")"
}
func (t Call) String() string {
	parts := make([]string, len(t.Args))
	for i, a := range t.Args {
		parts[i] = a.String()
	}
	return t.Fn + "(" + strings.Join(parts, ", ") + ")"
}
func (t If) String() string {
	return fmt.Sprintf("if %s %s %s then %s else %s", t.L, t.Op, t.R, t.Then, t.Else)
}
func (t MatchT) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "match %s {", t.Scrut)
	for i, a := range t.Arms {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s(%s) => %s", a.Ctor, strings.Join(a.Binds, ", "), a.Body)
	}
	b.WriteString("}")
	return b.String()
}

// Def is one total function definition.
type Def struct {
	Name   string
	Params []string
	Body   Term
}

// Defs is the definition environment (total functions by name).
type Defs map[string]*Def

// DataDef describes one first-order index data type (an enum usable as
// an index domain): constructor name -> arity.
type DataDef struct {
	Name  string
	Ctors map[string]int
}

// FreeVars returns the free variables of t in sorted order.
func FreeVars(t Term) []string {
	seen := map[string]bool{}
	var walk func(t Term, bound map[string]bool)
	walk = func(t Term, bound map[string]bool) {
		switch x := t.(type) {
		case Var:
			if !bound[x.Name] {
				seen[x.Name] = true
			}
		case Prim:
			for _, a := range x.Args {
				walk(a, bound)
			}
		case Ctor:
			for _, a := range x.Args {
				walk(a, bound)
			}
		case Call:
			for _, a := range x.Args {
				walk(a, bound)
			}
		case If:
			walk(x.L, bound)
			walk(x.R, bound)
			walk(x.Then, bound)
			walk(x.Else, bound)
		case MatchT:
			walk(x.Scrut, bound)
			for _, arm := range x.Arms {
				inner := map[string]bool{}
				for k := range bound {
					inner[k] = true
				}
				for _, b := range arm.Binds {
					inner[b] = true
				}
				walk(arm.Body, inner)
			}
		}
	}
	walk(t, map[string]bool{})
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
