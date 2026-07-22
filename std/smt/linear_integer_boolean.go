package smt

const linearIntegerBooleanBranchLimit = 256

type integerBooleanBranches struct {
	count     int
	inline    [4]integerBooleanBranch
	overflow  []integerBooleanBranch
	exhausted bool
}

type integerBooleanBranch struct {
	count    int
	inline   [8]Term[BoolSort]
	overflow []Term[BoolSort]
}

func (branches *integerBooleanBranches) values() []integerBooleanBranch {
	if branches.overflow != nil {
		return branches.overflow[:branches.count]
	}
	return branches.inline[:branches.count]
}

func (branches *integerBooleanBranches) append(branch integerBooleanBranch) {
	if branches.count >= linearIntegerBooleanBranchLimit {
		branches.exhausted = true
		return
	}
	if branches.count < len(branches.inline) && branches.overflow == nil {
		branches.inline[branches.count] = branch
		branches.count++
		return
	}
	if branches.overflow == nil {
		branches.overflow = make([]integerBooleanBranch, branches.count, branches.count*2)
		copy(branches.overflow, branches.inline[:branches.count])
	}
	branches.overflow = append(branches.overflow, branch)
	branches.count++
}

func (branch *integerBooleanBranch) values() []Term[BoolSort] {
	if branch.overflow != nil {
		return branch.overflow[:branch.count]
	}
	return branch.inline[:branch.count]
}

func appendIntegerBooleanAtoms(first, second integerBooleanBranch) integerBooleanBranch {
	result := integerBooleanBranch{}
	total := first.count + second.count
	if total > len(result.inline) {
		result.overflow = make([]Term[BoolSort], 0, total)
	}
	for _, term := range first.values() {
		if result.overflow != nil {
			result.overflow = append(result.overflow, term)
		} else {
			result.inline[result.count] = term
		}
		result.count++
	}
	for _, term := range second.values() {
		if result.overflow != nil {
			result.overflow = append(result.overflow, term)
		} else {
			result.inline[result.count] = term
		}
		result.count++
	}
	return result
}

// solveBooleanLinearIntegerAssertions extends the conjunctive QF_LIA kernel
// to exact Boolean structure. It normalizes lazily into bounded branches and
// returns unknown rather than silently dropping an exponential expansion.
func solveBooleanLinearIntegerAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if outcome, recognized := solveCompactBooleanLinearIntegerAssertions(assertions); recognized {
		return outcome, true
	}
	branches := integerBooleanBranches{}
	branches.append(integerBooleanBranch{})
	for _, assertion := range assertions {
		next, ok := normalizeIntegerBoolean(assertion, true)
		if !ok {
			return checkOutcome{}, false
		}
		branches = combineIntegerBranches(branches, next)
		if branches.exhausted {
			return checkOutcome{status: checkUnknown, reason: ResourceLimit{Limit: linearIntegerBooleanBranchLimit}}, true
		}
	}
	var unknown *checkOutcome
	for _, branch := range branches.values() {
		outcome, recognized := solveLinearIntegerAssertions(branch.values())
		if !recognized {
			return checkOutcome{}, false
		}
		if outcome.status == checkSat {
			return outcome, true
		}
		if outcome.status == checkUnknown && unknown == nil {
			copy := outcome
			unknown = &copy
		}
	}
	if unknown != nil {
		return *unknown, true
	}
	return checkOutcome{status: checkUnsat}, true
}

func solveCompactBooleanLinearIntegerAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if len(assertions) != 1 {
		return checkOutcome{}, false
	}
	var terms []Term[BoolSort]
	switch value := assertions[0].(type) {
	case BooleanConjunction:
		values, negated := value.values()
		for _, item := range negated {
			if item {
				return checkOutcome{}, false
			}
		}
		terms = values
	case And:
		terms = value.Values
	default:
		return checkOutcome{}, false
	}
	var atoms [16]compactIntegerBooleanAtom
	return solveCompactBooleanLinearTerms(terms, 0, atoms[:], 0)
}

type compactIntegerBooleanAtom struct {
	equality IntegerLinearEquality
	strict   bool
	reverse  bool
}

func solveCompactBooleanLinearTerms(terms []Term[BoolSort], index int, atoms []compactIntegerBooleanAtom, count int) (checkOutcome, bool) {
	if index == len(terms) {
		return solveCompactIntegerBooleanAtoms(atoms[:count]), true
	}
	if count >= len(atoms) {
		return checkOutcome{}, false
	}
	switch value := terms[index].(type) {
	case IntegerLinearEquality:
		atoms[count] = compactIntegerBooleanAtom{equality: value}
		return solveCompactBooleanLinearTerms(terms, index+1, atoms, count+1)
	case IntegerLinearChoice:
		atoms[count] = compactIntegerBooleanAtom{equality: value.First}
		first, firstOK := solveCompactBooleanLinearTerms(terms, index+1, atoms, count+1)
		if firstOK && first.status == checkSat {
			return first, true
		}
		atoms[count] = compactIntegerBooleanAtom{equality: value.Second}
		second, secondOK := solveCompactBooleanLinearTerms(terms, index+1, atoms, count+1)
		return mergeIntegerBooleanAlternatives(first, firstOK, second, secondOK)
	case IntegerLinearDisequality:
		atoms[count] = compactIntegerBooleanAtom{equality: value.Equality, strict: true}
		first, firstOK := solveCompactBooleanLinearTerms(terms, index+1, atoms, count+1)
		if firstOK && first.status == checkSat {
			return first, true
		}
		atoms[count] = compactIntegerBooleanAtom{equality: value.Equality, strict: true, reverse: true}
		second, secondOK := solveCompactBooleanLinearTerms(terms, index+1, atoms, count+1)
		return mergeIntegerBooleanAlternatives(first, firstOK, second, secondOK)
	default:
		return checkOutcome{}, false
	}
}

func solveCompactIntegerBooleanAtoms(atoms []compactIntegerBooleanAtom) checkOutcome {
	problem := integerLinearProblem{}
	for _, atom := range atoms {
		coefficient := NewIntegerValue(atom.equality.Coefficient)
		if atom.strict {
			bound := AddIntegerValue(atom.equality.Value, NewIntegerValue(-1))
			if atom.reverse {
				coefficient = NegateIntegerValue(coefficient)
				bound = AddIntegerValue(NegateIntegerValue(atom.equality.Value), NewIntegerValue(-1))
			}
			coefficients := integerCoefficients{}
			coefficients.add(atom.equality.ID, coefficient)
			problem.appendConstraint(integerLinearConstraint{coefficients: coefficients, bound: bound})
			problem.addSymbol(atom.equality.ID)
			continue
		}
		first, second := integerCoefficients{}, integerCoefficients{}
		first.add(atom.equality.ID, coefficient)
		second.add(atom.equality.ID, NegateIntegerValue(coefficient))
		problem.appendConstraint(integerLinearConstraint{coefficients: first, bound: atom.equality.Value})
		problem.appendConstraint(integerLinearConstraint{coefficients: second, bound: NegateIntegerValue(atom.equality.Value)})
		problem.addSymbol(atom.equality.ID)
	}
	nodes := 0
	outcome, exhausted := problem.branch(nil, &nodes)
	if exhausted {
		return checkOutcome{status: checkUnknown, reason: ResourceLimit{Limit: linearIntegerBranchLimit}}
	}
	return outcome
}

func mergeIntegerBooleanAlternatives(first checkOutcome, firstOK bool, second checkOutcome, secondOK bool) (checkOutcome, bool) {
	if !firstOK || !secondOK {
		return checkOutcome{}, false
	}
	if second.status == checkSat {
		return second, true
	}
	if first.status == checkUnknown {
		return first, true
	}
	if second.status == checkUnknown {
		return second, true
	}
	return checkOutcome{status: checkUnsat}, true
}

func containsBooleanLinearInteger(term any) bool {
	switch value := term.(type) {
	case Or, Implies, Iff, IntegerLinearDisequality, IntegerLinearChoice:
		return true
	case Not:
		return containsIntegerTheory(value.Value)
	case And:
		for _, item := range value.Values {
			if containsBooleanLinearInteger(item) {
				return true
			}
		}
	case BooleanConjunction:
		terms, negated := value.values()
		for index, item := range terms {
			if negated[index] || containsBooleanLinearInteger(item) {
				return true
			}
		}
	}
	return false
}

func containsBooleanLinearIntegerAssertions(assertions []Term[BoolSort]) bool {
	for _, assertion := range assertions {
		if containsBooleanLinearInteger(assertion) {
			return true
		}
	}
	return false
}

func normalizeIntegerBoolean(term Term[BoolSort], positive bool) (integerBooleanBranches, bool) {
	switch value := term.(type) {
	case Bool:
		if value.Value == positive {
			result := integerBooleanBranches{}
			result.append(integerBooleanBranch{})
			return result, true
		}
		return integerBooleanBranches{}, true
	case Not:
		return normalizeIntegerBoolean(value.Value, !positive)
	case And:
		return normalizeIntegerBooleanMany(value.Values, positive, positive)
	case Or:
		return normalizeIntegerBooleanMany(value.Values, positive, !positive)
	case Implies:
		if positive {
			left, leftOK := normalizeIntegerBoolean(value.Left, false)
			right, rightOK := normalizeIntegerBoolean(value.Right, true)
			return unionIntegerBranches(left, right), leftOK && rightOK
		}
		left, leftOK := normalizeIntegerBoolean(value.Left, true)
		right, rightOK := normalizeIntegerBoolean(value.Right, false)
		return combineIntegerBranches(left, right), leftOK && rightOK
	case Iff:
		leftTrue, firstOK := normalizeIntegerBoolean(value.Left, true)
		rightTrue, secondOK := normalizeIntegerBoolean(value.Right, true)
		leftFalse, thirdOK := normalizeIntegerBoolean(value.Left, false)
		rightFalse, fourthOK := normalizeIntegerBoolean(value.Right, false)
		if positive {
			return unionIntegerBranches(combineIntegerBranches(leftTrue, rightTrue), combineIntegerBranches(leftFalse, rightFalse)), firstOK && secondOK && thirdOK && fourthOK
		}
		return unionIntegerBranches(combineIntegerBranches(leftTrue, rightFalse), combineIntegerBranches(leftFalse, rightTrue)), firstOK && secondOK && thirdOK && fourthOK
	case BooleanConjunction:
		terms, negated := value.values()
		result := integerBooleanBranches{}
		if positive {
			result.append(integerBooleanBranch{})
		}
		for index, item := range terms {
			part, ok := normalizeIntegerBoolean(item, positive != negated[index])
			if !ok {
				return integerBooleanBranches{}, false
			}
			if positive {
				result = combineIntegerBranches(result, part)
			} else {
				result = unionIntegerBranches(result, part)
			}
		}
		return result, true
	case LessEqual:
		if positive {
			return integerAtom(term), true
		}
		return integerAtom(Less{Left: value.Right, Right: value.Left}), true
	case Less:
		if positive {
			return integerAtom(term), true
		}
		return integerAtom(LessEqual{Left: value.Right, Right: value.Left}), true
	case Equal:
		left, leftOK := value.Left.(Term[IntSort])
		right, rightOK := value.Right.(Term[IntSort])
		if !leftOK || !rightOK {
			return integerBooleanBranches{}, false
		}
		if positive {
			return integerAtom(term), true
		}
		return unionIntegerBranches(integerAtom(Less{Left: left, Right: right}), integerAtom(Less{Left: right, Right: left})), true
	case IntegerDifferenceConstraint:
		return normalizeIntegerBoolean(integerDifferenceAsTerm(value), positive)
	case IntegerDifferenceSystem:
		terms := make([]Term[BoolSort], len(value.values()))
		for index, constraint := range value.values() {
			terms[index] = integerDifferenceAsTerm(constraint)
		}
		return normalizeIntegerBooleanMany(terms, positive, positive)
	case IntegerLinearEquality:
		if positive {
			return integerAtom(value), true
		}
		return unionIntegerBranches(integerAtom(integerLinearStrictBound{Equality: value}), integerAtom(integerLinearStrictBound{Equality: value, Reverse: true})), true
	case IntegerLinearDisequality:
		return normalizeIntegerBoolean(value.Equality, !positive)
	case IntegerLinearChoice:
		if positive {
			return unionIntegerBranches(integerAtom(value.First), integerAtom(value.Second)), true
		}
		first, _ := normalizeIntegerBoolean(value.First, false)
		second, _ := normalizeIntegerBoolean(value.Second, false)
		return combineIntegerBranches(first, second), true
	default:
		return integerBooleanBranches{}, false
	}
}

func normalizeIntegerBooleanMany(terms []Term[BoolSort], childPositive, conjunction bool) (integerBooleanBranches, bool) {
	result := integerBooleanBranches{}
	if conjunction {
		result.append(integerBooleanBranch{})
	}
	for _, term := range terms {
		part, ok := normalizeIntegerBoolean(term, childPositive)
		if !ok {
			return integerBooleanBranches{}, false
		}
		if conjunction {
			result = combineIntegerBranches(result, part)
		} else {
			result = unionIntegerBranches(result, part)
		}
		if result.exhausted {
			break
		}
	}
	return result, true
}

func integerAtom(term Term[BoolSort]) integerBooleanBranches {
	branch := integerBooleanBranch{count: 1}
	branch.inline[0] = term
	result := integerBooleanBranches{}
	result.append(branch)
	return result
}

func unionIntegerBranches(left, right integerBooleanBranches) integerBooleanBranches {
	if left.exhausted || right.exhausted || left.count+right.count > linearIntegerBooleanBranchLimit {
		return integerBooleanBranches{exhausted: true}
	}
	result := integerBooleanBranches{}
	for _, branch := range left.values() {
		result.append(branch)
	}
	for _, branch := range right.values() {
		result.append(branch)
	}
	return result
}

func combineIntegerBranches(left, right integerBooleanBranches) integerBooleanBranches {
	if left.exhausted || right.exhausted || left.count != 0 && right.count > linearIntegerBooleanBranchLimit/left.count {
		return integerBooleanBranches{exhausted: true}
	}
	result := integerBooleanBranches{}
	for _, first := range left.values() {
		for _, second := range right.values() {
			result.append(appendIntegerBooleanAtoms(first, second))
		}
	}
	return result
}

func integerDifferenceAsTerm(value IntegerDifferenceConstraint) Term[BoolSort] {
	left := Term[IntSort](Integer{Value: 0})
	if value.HasPositive {
		left = IntegerVariable(value.PositiveID)
	}
	if value.HasNegative {
		left = Subtract{Left: left, Right: IntegerVariable(value.NegativeID)}
	}
	bound := NewIntegerValue(value.Bound)
	if value.Wide {
		bound = value.WideBound
	}
	if value.Strict {
		return Less{Left: left, Right: IntegerTerm(bound)}
	}
	return LessEqual{Left: left, Right: IntegerTerm(bound)}
}
