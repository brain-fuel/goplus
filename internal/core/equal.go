package core

// Equal is definitional equality on normal forms: canonical linear
// forms compare by constant and coefficient maps; data compares
// structurally; neutrals compare by shape. Symbolic facts beyond this
// (n+m vs Plus(n, m), hypotheses) are the decider's business.
func Equal(a, b Value) bool {
	switch x := a.(type) {
	case VLin:
		y, ok := b.(VLin)
		if !ok {
			y2, lift := b.(VNeu)
			if !lift {
				return false
			}
			y = linAtom(y2)
		}
		return linEqual(x, y)
	case VCtor:
		y, ok := b.(VCtor)
		if !ok || x.Type != y.Type || x.Name != y.Name || len(x.Args) != len(y.Args) {
			return false
		}
		for i := range x.Args {
			if !Equal(x.Args[i], y.Args[i]) {
				return false
			}
		}
		return true
	case VNeu:
		if y, ok := b.(VLin); ok {
			return linEqual(linAtom(x), y)
		}
		y, ok := b.(VNeu)
		return ok && x.N.String() == y.N.String()
	}
	return false
}

func linEqual(a, b VLin) bool {
	if a.Const.Cmp(b.Const) != 0 || len(a.Coef) != len(b.Coef) {
		return false
	}
	for k, c := range a.Coef {
		d, ok := b.Coef[k]
		if !ok || c.Cmp(d) != 0 {
			return false
		}
	}
	return true
}
