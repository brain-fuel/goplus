package smt

// rewriteSymbolicIntegerToReal lowers the complete conjunctive fragment in
// which symbolic to_real terms are compared with another to_real term or an
// exact rational constant. The resulting integer relations preserve SMT-LIB
// semantics exactly; unsupported mixed expressions are left untouched.
func rewriteSymbolicIntegerToReal(assertions []Term[BoolSort]) []Term[BoolSort] {
	var rewritten []Term[BoolSort]
	for index, assertion := range assertions {
		next, changed := rewriteSymbolicIntegerToRealAssertion(assertion)
		if !changed {
			if rewritten != nil {
				rewritten = append(rewritten, assertion)
			}
			continue
		}
		if rewritten == nil {
			rewritten = make([]Term[BoolSort], 0, len(assertions))
			rewritten = append(rewritten, assertions[:index]...)
		}
		rewritten = append(rewritten, next)
	}
	if rewritten == nil {
		return assertions
	}
	return rewritten
}

func rewriteSymbolicIntegerToRealAssertion(term Term[BoolSort]) (Term[BoolSort], bool) {
	switch value := term.(type) {
	case BooleanConjunction:
		terms := value.InlineTerms[:min(value.Count, len(value.InlineTerms))]
		negated := value.InlineNegated[:len(terms)]
		if value.OverflowTerms != nil {
			terms = value.OverflowTerms[:value.Count]
			negated = value.OverflowNegated[:value.Count]
		}
		var rewritten []Term[BoolSort]
		for index, item := range terms {
			next, changed := rewriteSymbolicIntegerToRealAssertion(item)
			if !changed {
				if rewritten != nil {
					if negated[index] {
						rewritten = append(rewritten, Not{Value: item})
					} else {
						rewritten = append(rewritten, item)
					}
				}
				continue
			}
			if rewritten == nil {
				rewritten = make([]Term[BoolSort], 0, len(terms))
				for previous := 0; previous < index; previous++ {
					if negated[previous] {
						rewritten = append(rewritten, Not{Value: terms[previous]})
					} else {
						rewritten = append(rewritten, terms[previous])
					}
				}
			}
			if negated[index] {
				next = Not{Value: next}
			}
			rewritten = append(rewritten, next)
		}
		if rewritten == nil {
			return term, false
		}
		return And{Values: rewritten}, true
	case And:
		var rewritten []Term[BoolSort]
		for index, item := range value.Values {
			next, changed := rewriteSymbolicIntegerToRealAssertion(item)
			if !changed {
				if rewritten != nil {
					rewritten = append(rewritten, item)
				}
				continue
			}
			if rewritten == nil {
				rewritten = make([]Term[BoolSort], 0, len(value.Values))
				rewritten = append(rewritten, value.Values[:index]...)
			}
			rewritten = append(rewritten, next)
		}
		if rewritten == nil {
			return term, false
		}
		return And{Values: rewritten}, true
	case Equal:
		left, leftOK := value.Left.(Term[RealSort])
		right, rightOK := value.Right.(Term[RealSort])
		if !leftOK || !rightOK {
			return term, false
		}
		return rewriteIntegerRealEquality(left, right)
	case RealLessEqual:
		return rewriteIntegerRealOrder(value.Left, value.Right, false)
	case RealLess:
		return rewriteIntegerRealOrder(value.Left, value.Right, true)
	case Not:
		rewritten, changed := rewriteSymbolicIntegerToRealAssertion(value.Value)
		if !changed {
			return term, false
		}
		return Not{Value: rewritten}, true
	default:
		return term, false
	}
}

func integerToRealSource(term Term[RealSort]) (Term[IntSort], bool) {
	value, ok := term.(integerToReal)
	return value.value, ok
}

func rewriteIntegerRealEquality(left, right Term[RealSort]) (Term[BoolSort], bool) {
	leftInteger, leftOK := integerToRealSource(left)
	rightInteger, rightOK := integerToRealSource(right)
	switch {
	case leftOK && rightOK:
		return Equal{Left: leftInteger, Right: rightInteger}, true
	case leftOK:
		return integerEqualsRealConstant(leftInteger, right)
	case rightOK:
		return integerEqualsRealConstant(rightInteger, left)
	default:
		return nil, false
	}
}

func integerEqualsRealConstant(integer Term[IntSort], real Term[RealSort]) (Term[BoolSort], bool) {
	value, ok := ExactRealConstant(real)
	if !ok {
		return nil, false
	}
	if !value.IsInteger() {
		return Bool{Value: false}, true
	}
	return Equal{Left: integer, Right: IntegerTerm(FloorRational(value))}, true
}

func rewriteIntegerRealOrder(left, right Term[RealSort], strict bool) (Term[BoolSort], bool) {
	leftInteger, leftOK := integerToRealSource(left)
	rightInteger, rightOK := integerToRealSource(right)
	if leftOK && rightOK {
		if strict {
			return Less{Left: leftInteger, Right: rightInteger}, true
		}
		return LessEqual{Left: leftInteger, Right: rightInteger}, true
	}
	if leftOK {
		constant, ok := ExactRealConstant(right)
		if !ok {
			return nil, false
		}
		floor := FloorRational(constant)
		if strict && constant.IsInteger() {
			return Less{Left: leftInteger, Right: IntegerTerm(floor)}, true
		}
		return LessEqual{Left: leftInteger, Right: IntegerTerm(floor)}, true
	}
	if rightOK {
		constant, ok := ExactRealConstant(left)
		if !ok {
			return nil, false
		}
		floor := FloorRational(constant)
		if strict {
			return Less{Left: IntegerTerm(floor), Right: rightInteger}, true
		}
		ceiling := floor
		if !constant.IsInteger() {
			ceiling = AddIntegerValue(ceiling, NewIntegerValue(1))
		}
		return LessEqual{Left: IntegerTerm(ceiling), Right: rightInteger}, true
	}
	return nil, false
}
