package core

import (
	"math/big"
)

// The linear-arithmetic decider. Facts and goals are linear forms over
// nat atoms; every nat atom (variable, stuck call, product) is
// implicitly ≥ 0. The procedure is SOUND and incomplete: "true" is a
// proof, "false" means unknown — obligations that come back false go
// stuck with a guided diagnostic, never wrongly accepted.
//
// Method: substitute equality hypotheses (unit-coefficient elimination),
// then check the goal — an equality must cancel to 0 = 0; an inequality
// d ≥ 0 holds when all coefficients and the constant are non-negative,
// alone or after adding one or two inequality hypotheses.

// FactOp distinguishes fact shapes.
type FactOp int

const (
	FactEq FactOp = iota // L == 0
	FactGe               // L ≥ 0
)

// Fact is one hypothesis or goal in normalized form.
type Fact struct {
	Op FactOp
	L  VLin
}

// MkEq builds the fact a == b; MkGe builds a ≥ b.
func MkEq(a, b Value) Fact { return Fact{Op: FactEq, L: linAdd(asLin(a), asLin(b), -1)} }
func MkGe(a, b Value) Fact { return Fact{Op: FactGe, L: linAdd(asLin(a), asLin(b), -1)} }

// Decide reports whether the hypotheses entail the goal.
func Decide(goal Fact, hyps []Fact) bool {
	// Copy everything; elimination mutates.
	facts := make([]Fact, len(hyps))
	for i, h := range hyps {
		facts[i] = Fact{Op: h.Op, L: linCopy(h.L)}
	}
	g := Fact{Op: goal.Op, L: linCopy(goal.L)}

	// Contradictory hypotheses entail anything.
	if contradiction(facts) {
		return true
	}

	// Equality elimination: any equality with a ±1-coefficient atom
	// defines that atom; substitute it everywhere.
	for changed := true; changed; {
		changed = false
		for i, f := range facts {
			if f.Op != FactEq {
				continue
			}
			k, c := unitAtom(f.L)
			if k == "" {
				continue
			}
			// atom = -(rest)/c with c = ±1: atom := s·rest where s = -c.
			rest := linCopy(f.L)
			delete(rest.Coef, k)
			delete(rest.Atoms, k)
			s := new(big.Int).Neg(c) // ±1
			repl := linScale(rest, s)
			for j := range facts {
				if j == i {
					continue
				}
				facts[j].L = substAtom(facts[j].L, k, repl)
			}
			g.L = substAtom(g.L, k, repl)
			// The eliminated atom was a nat (≥ 0): its definition
			// survives as repl ≥ 0 or the entailment weakens.
			facts[i] = Fact{Op: FactGe, L: repl}
			changed = true
		}
	}
	if contradiction(facts) {
		return true
	}

	switch g.Op {
	case FactEq:
		if ground(g.L) && g.L.Const.Sign() == 0 {
			return true
		}
		// a == b iff a ≥ b and b ≥ a.
		return Decide(Fact{Op: FactGe, L: linCopy(g.L)}, facts) &&
			Decide(Fact{Op: FactGe, L: linScale(g.L, big.NewInt(-1))}, facts)
	case FactGe:
		if nonneg(g.L) {
			return true
		}
		var ineqs []VLin
		for _, f := range facts {
			switch f.Op {
			case FactGe:
				ineqs = append(ineqs, f.L)
			case FactEq:
				// Residual equalities contribute both directions.
				ineqs = append(ineqs, f.L, linScale(f.L, big.NewInt(-1)))
			}
		}
		// g ≥ 0 follows when g minus a non-negative combination of
		// hypotheses is manifestly non-negative: g = residual + Σh.
		for i := range ineqs {
			if nonneg(linAdd(g.L, ineqs[i], -1)) {
				return true
			}
			for j := i + 1; j < len(ineqs); j++ {
				if nonneg(linAdd(linAdd(g.L, ineqs[i], -1), ineqs[j], -1)) {
					return true
				}
			}
		}
	}
	return false
}

// nonneg: every atom is ≥ 0, so all-nonnegative coefficients and a
// non-negative constant prove L ≥ 0.
func nonneg(l VLin) bool {
	if l.Const.Sign() < 0 {
		return false
	}
	for _, c := range l.Coef {
		if c.Sign() < 0 {
			return false
		}
	}
	return true
}

// contradiction: a ground equality with non-zero constant, or a ground
// inequality with negative constant, among the hypotheses.
func contradiction(facts []Fact) bool {
	for _, f := range facts {
		if !ground(f.L) {
			continue
		}
		if f.Op == FactEq && f.L.Const.Sign() != 0 {
			return true
		}
		if f.Op == FactGe && f.L.Const.Sign() < 0 {
			return true
		}
	}
	return false
}

// unitAtom finds an atom with coefficient ±1.
func unitAtom(l VLin) (string, *big.Int) {
	for k, c := range l.Coef {
		if c.CmpAbs(bigOne) == 0 {
			return k, c
		}
	}
	return "", nil
}

// substAtom replaces atom k with repl in l.
func substAtom(l VLin, k string, repl VLin) VLin {
	c, ok := l.Coef[k]
	if !ok {
		return l
	}
	out := linCopy(l)
	delete(out.Coef, k)
	delete(out.Atoms, k)
	return linAdd(out, linScale(repl, c), 1)
}

func linCopy(l VLin) VLin {
	out := linConst(l.Const)
	for k, c := range l.Coef {
		out.Coef[k] = new(big.Int).Set(c)
		out.Atoms[k] = l.Atoms[k]
	}
	return out
}

func linScale(l VLin, s *big.Int) VLin {
	out := linConst(new(big.Int).Mul(l.Const, s))
	if s.Sign() == 0 {
		return out
	}
	for k, c := range l.Coef {
		out.Coef[k] = new(big.Int).Mul(c, s)
		out.Atoms[k] = l.Atoms[k]
	}
	return out
}
