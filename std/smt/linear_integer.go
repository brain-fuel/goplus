package smt

import "math/big"

const linearIntegerBranchLimit = 4096

// IntegerLinearEquality is the allocation-conscious normal form a
// compatibility layer can use for coefficient*x = value.
type IntegerLinearEquality struct {
	ID          int
	Coefficient int64
	Value       IntegerValue
}

func (IntegerLinearEquality) isTerm(BoolSort) {}

func CompactIntegerLinearEquality(left, right Term[IntSort]) (IntegerLinearEquality, bool) {
	form := linearInteger{valid: true}
	accumulateInteger(left, 1, &form)
	accumulateInteger(right, -1, &form)
	if !form.valid || len(form.overflow) != 0 {
		return IntegerLinearEquality{}, false
	}
	result := IntegerLinearEquality{Value: NegateIntegerValue(form.constant)}
	count := 0
	for index := 0; index < form.count; index++ {
		term := form.inline[index]
		if term.coefficient != 0 {
			result.ID, result.Coefficient, count = term.id, term.coefficient, count+1
		}
	}
	return result, count == 1
}

type integerLinearConstraint struct {
	coefficients map[int]IntegerValue
	bound        IntegerValue
}

type integerLinearProblem struct {
	constraints []integerLinearConstraint
	symbols     []int
	seen        map[int]struct{}
	unsat       bool
}

type integerAffine struct {
	constant     IntegerValue
	coefficients map[int]IntegerValue
	valid        bool
}

// solveLinearIntegerAssertions decides conjunctive QF_LIA with exact
// arbitrary-precision coefficients. It uses the existing exact rational
// simplex as a relaxation and branches only on fractional integer values.
func solveLinearIntegerAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if outcome, recognized := solveSingleVariableLinearIntegerEquality(assertions); recognized {
		return outcome, true
	}
	problem := integerLinearProblem{seen: make(map[int]struct{})}
	for _, assertion := range assertions {
		if !problem.boolean(assertion) {
			return checkOutcome{}, false
		}
	}
	if problem.unsat {
		return checkOutcome{status: checkUnsat}, true
	}
	nodes := 0
	outcome, exhausted := problem.branch(nil, &nodes)
	if exhausted {
		return checkOutcome{status: checkUnknown, reason: ResourceLimit{Limit: linearIntegerBranchLimit}}, true
	}
	return outcome, true
}

func solveSingleVariableLinearIntegerEquality(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if len(assertions) != 1 {
		return checkOutcome{}, false
	}
	equality, ok := assertions[0].(Equal)
	if compact, compactOK := assertions[0].(IntegerLinearEquality); compactOK {
		return solveCompactIntegerLinearEquality(compact), true
	}
	if !ok {
		return checkOutcome{}, false
	}
	left, leftOK := equality.Left.(Term[IntSort])
	right, rightOK := equality.Right.(Term[IntSort])
	if !leftOK || !rightOK {
		return checkOutcome{}, false
	}
	form := linearInteger{valid: true}
	accumulateInteger(left, 1, &form)
	accumulateInteger(right, -1, &form)
	if !form.valid || len(form.overflow) != 0 {
		return checkOutcome{}, false
	}
	id, coefficient, count := 0, int64(0), 0
	for index := 0; index < form.count; index++ {
		term := form.inline[index]
		if term.coefficient != 0 {
			id, coefficient, count = term.id, term.coefficient, count+1
		}
	}
	if count == 0 {
		if CompareIntegerValue(form.constant, IntegerValue{}) == 0 {
			return checkOutcome{status: checkSat}, true
		}
		return checkOutcome{status: checkUnsat}, true
	}
	if count != 1 {
		return checkOutcome{}, false
	}
	if form.constant.large == nil && form.constant.small != -1<<63 {
		numerator := -form.constant.small
		if numerator%coefficient != 0 {
			return checkOutcome{status: checkUnsat}, true
		}
		model := integerModel{}
		model.set(id, NewIntegerValue(numerator/coefficient))
		return checkOutcome{status: checkSat, integers: model}, true
	}
	numerator := NegateIntegerValue(form.constant).big()
	denominator := big.NewInt(coefficient)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if remainder.Sign() != 0 {
		return checkOutcome{status: checkUnsat}, true
	}
	model := integerModel{}
	model.set(id, integerValueFromBig(quotient))
	return checkOutcome{status: checkSat, integers: model}, true
}

func solveCompactIntegerLinearEquality(value IntegerLinearEquality) checkOutcome {
	if value.Coefficient == 0 {
		if CompareIntegerValue(value.Value, IntegerValue{}) == 0 {
			return checkOutcome{status: checkSat}
		}
		return checkOutcome{status: checkUnsat}
	}
	if value.Value.large == nil && value.Value.small != -1<<63 {
		if value.Value.small%value.Coefficient != 0 {
			return checkOutcome{status: checkUnsat}
		}
		model := integerModel{}
		model.set(value.ID, NewIntegerValue(value.Value.small/value.Coefficient))
		return checkOutcome{status: checkSat, integers: model}
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Value.big(), big.NewInt(value.Coefficient), remainder)
	if remainder.Sign() != 0 {
		return checkOutcome{status: checkUnsat}
	}
	model := integerModel{}
	model.set(value.ID, integerValueFromBig(quotient))
	return checkOutcome{status: checkSat, integers: model}
}

func (problem *integerLinearProblem) boolean(term Term[BoolSort]) bool {
	switch value := term.(type) {
	case Bool:
		problem.unsat = problem.unsat || !value.Value
		return true
	case And:
		for _, item := range value.Values {
			if !problem.boolean(item) {
				return false
			}
		}
		return true
	case BooleanConjunction:
		terms, negated := value.values()
		for index, item := range terms {
			if negated[index] || !problem.boolean(item) {
				return false
			}
		}
		return true
	case LessEqual:
		return problem.relation(value.Left, value.Right, false)
	case Less:
		return problem.relation(value.Left, value.Right, true)
	case Equal:
		left, leftOK := value.Left.(Term[IntSort])
		right, rightOK := value.Right.(Term[IntSort])
		return leftOK && rightOK && problem.relation(left, right, false) && problem.relation(right, left, false)
	case IntegerDifferenceConstraint:
		return problem.compactDifference(value)
	case IntegerDifferenceSystem:
		for _, constraint := range value.values() {
			if !problem.compactDifference(constraint) {
				return false
			}
		}
		return true
	case IntegerLinearEquality:
		coefficient := NewIntegerValue(value.Coefficient)
		problem.constraints = append(problem.constraints,
			integerLinearConstraint{coefficients: map[int]IntegerValue{value.ID: coefficient}, bound: value.Value},
			integerLinearConstraint{coefficients: map[int]IntegerValue{value.ID: NegateIntegerValue(coefficient)}, bound: NegateIntegerValue(value.Value)},
		)
		if _, exists := problem.seen[value.ID]; !exists {
			problem.seen[value.ID] = struct{}{}
			problem.symbols = append(problem.symbols, value.ID)
		}
		return true
	default:
		return false
	}
}

func (problem *integerLinearProblem) compactDifference(value IntegerDifferenceConstraint) bool {
	bound := NewIntegerValue(value.Bound)
	if value.Wide {
		bound = value.WideBound
	}
	if value.Strict {
		bound = AddIntegerValue(bound, NewIntegerValue(-1))
	}
	coefficients := make(map[int]IntegerValue, 2)
	if value.HasPositive {
		coefficients[value.PositiveID] = NewIntegerValue(1)
	}
	if value.HasNegative {
		coefficients[value.NegativeID] = NewIntegerValue(-1)
	}
	if len(coefficients) == 0 {
		problem.unsat = problem.unsat || CompareIntegerValue(IntegerValue{}, bound) > 0
		return true
	}
	for id := range coefficients {
		if _, exists := problem.seen[id]; !exists {
			problem.seen[id] = struct{}{}
			problem.symbols = append(problem.symbols, id)
		}
	}
	problem.constraints = append(problem.constraints, integerLinearConstraint{coefficients: coefficients, bound: bound})
	return true
}

func (problem *integerLinearProblem) relation(left, right Term[IntSort], strict bool) bool {
	form := integerAffine{coefficients: make(map[int]IntegerValue), valid: true}
	accumulateIntegerAffine(left, NewIntegerValue(1), &form)
	accumulateIntegerAffine(right, NewIntegerValue(-1), &form)
	if !form.valid {
		return false
	}
	bound := NegateIntegerValue(form.constant)
	if strict {
		bound = AddIntegerValue(bound, NewIntegerValue(-1))
	}
	for id, coefficient := range form.coefficients {
		if CompareIntegerValue(coefficient, IntegerValue{}) == 0 {
			delete(form.coefficients, id)
			continue
		}
		if _, exists := problem.seen[id]; !exists {
			problem.seen[id] = struct{}{}
			problem.symbols = append(problem.symbols, id)
		}
	}
	if len(form.coefficients) == 0 {
		problem.unsat = problem.unsat || CompareIntegerValue(IntegerValue{}, bound) > 0
		return true
	}
	problem.constraints = append(problem.constraints, integerLinearConstraint{coefficients: form.coefficients, bound: bound})
	return true
}

func accumulateIntegerAffine(term Term[IntSort], multiplier IntegerValue, form *integerAffine) {
	if !form.valid {
		return
	}
	switch value := term.(type) {
	case Integer:
		form.constant = AddIntegerValue(form.constant, MultiplyIntegerValue(multiplier, NewIntegerValue(value.Value)))
	case integerExact[IntSort]:
		form.constant = AddIntegerValue(form.constant, MultiplyIntegerValue(multiplier, value.value))
	case IntSymbol:
		form.add(value.ID, multiplier)
	case integerVariable[IntSort]:
		form.add(value.iD, multiplier)
	case Add:
		for _, item := range value.Values {
			accumulateIntegerAffine(item, multiplier, form)
		}
	case Subtract:
		accumulateIntegerAffine(value.Left, multiplier, form)
		accumulateIntegerAffine(value.Right, NegateIntegerValue(multiplier), form)
	case IntegerScale:
		accumulateIntegerAffine(value.Value, MultiplyIntegerValue(multiplier, value.Coefficient), form)
	default:
		form.valid = false
	}
}

func (form *integerAffine) add(id int, coefficient IntegerValue) {
	form.coefficients[id] = AddIntegerValue(form.coefficients[id], coefficient)
}

func (problem *integerLinearProblem) branch(extra []integerLinearConstraint, nodes *int) (checkOutcome, bool) {
	*nodes++
	if *nodes > linearIntegerBranchLimit {
		return checkOutcome{}, true
	}
	relaxation := rationalProblem{}
	for _, id := range problem.symbols {
		relaxation.appendSymbol(id)
	}
	appendConstraint := func(constraint integerLinearConstraint) {
		coefficients := rationalCoefficients{}
		for id, coefficient := range constraint.coefficients {
			coefficients.add(id, rationalFromInteger(coefficient))
		}
		coefficients.compact()
		relaxation.appendConstraint(rationalConstraint{coefficients: coefficients, bound: rationalFromInteger(constraint.bound)})
	}
	for _, constraint := range problem.constraints {
		appendConstraint(constraint)
	}
	for _, constraint := range extra {
		appendConstraint(constraint)
	}
	result, _ := relaxation.solve()
	if result.status == checkUnsat {
		return result, false
	}
	for _, id := range problem.symbols {
		value, _ := result.reals.lookup(id)
		if value.IsInteger() {
			continue
		}
		floor := floorRational(value)
		ceil := AddIntegerValue(floor, NewIntegerValue(1))
		left := integerLinearConstraint{coefficients: map[int]IntegerValue{id: NewIntegerValue(1)}, bound: floor}
		if outcome, exhausted := problem.branch(append(extra, left), nodes); exhausted || outcome.status == checkSat {
			return outcome, exhausted
		}
		right := integerLinearConstraint{coefficients: map[int]IntegerValue{id: NewIntegerValue(-1)}, bound: NegateIntegerValue(ceil)}
		return problem.branch(append(extra, right), nodes)
	}
	model := integerModel{}
	model.reserve(len(problem.symbols))
	for _, id := range problem.symbols {
		value, _ := result.reals.lookup(id)
		model.set(id, integerValueFromBig(value.big().Num()))
	}
	return checkOutcome{status: checkSat, integers: model}, false
}

func rationalFromInteger(value IntegerValue) Rational {
	return rationalFromBig(new(big.Rat).SetInt(value.big()))
}

func floorRational(value Rational) IntegerValue {
	fraction := value.big()
	return integerValueFromBig(new(big.Int).Div(fraction.Num(), fraction.Denom()))
}
