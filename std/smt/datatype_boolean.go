package smt

const datatypeBooleanBranchLimit = 256

type datatypeBooleanBranches struct {
	values    []datatypeBooleanBranch
	exhausted bool
}

type datatypeBooleanBranch struct {
	atoms []Term[BoolSort]
}

func solveBooleanDatatypeAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	branches := datatypeBooleanBranches{values: []datatypeBooleanBranch{{}}}
	for _, assertion := range assertions {
		next, ok := normalizeDatatypeBoolean(assertion, true)
		if !ok {
			return checkOutcome{}, false
		}
		branches = combineDatatypeBooleanBranches(branches, next)
		if branches.exhausted {
			return checkOutcome{status: checkUnknown, reason: ResourceLimit{Limit: datatypeBooleanBranchLimit}}, true
		}
	}
	var unknown checkOutcome
	unknownSeen := false
	for _, branch := range branches.values {
		outcome, recognized := solveDatatypeAssertions(branch.atoms)
		if !recognized {
			return checkOutcome{}, false
		}
		if outcome.status == checkSat {
			return outcome, true
		}
		if outcome.status == checkUnknown && !unknownSeen {
			unknown, unknownSeen = outcome, true
		}
	}
	if unknownSeen {
		return unknown, true
	}
	return checkOutcome{status: checkUnsat}, true
}

func containsBooleanDatatypeAssertions(assertions []Term[BoolSort]) bool {
	for _, assertion := range assertions {
		if containsBooleanDatatype(assertion) {
			return true
		}
	}
	return false
}

func containsBooleanDatatype(term Term[BoolSort]) bool {
	switch value := term.(type) {
	case Or, Implies, Iff, If[BoolSort]:
		return containsDatatypeTheory(term)
	case Not:
		return containsBooleanDatatype(value.Value)
	case And:
		for _, item := range value.Values {
			if containsBooleanDatatype(item) {
				return true
			}
		}
	case BooleanConjunction:
		items, _ := value.values()
		for _, item := range items {
			if containsBooleanDatatype(item) {
				return true
			}
		}
	case Equal:
		left, leftOK := value.Left.(Term[BoolSort])
		right, rightOK := value.Right.(Term[BoolSort])
		return leftOK && rightOK && (containsDatatypeTheory(left) || containsDatatypeTheory(right))
	}
	return false
}

func normalizeDatatypeBoolean(term Term[BoolSort], positive bool) (datatypeBooleanBranches, bool) {
	switch value := term.(type) {
	case Bool:
		if value.Value == positive {
			return datatypeBooleanBranches{values: []datatypeBooleanBranch{{}}}, true
		}
		return datatypeBooleanBranches{}, true
	case Not:
		return normalizeDatatypeBoolean(value.Value, !positive)
	case And:
		return normalizeDatatypeBooleanMany(value.Values, positive, positive)
	case Or:
		return normalizeDatatypeBooleanMany(value.Values, positive, !positive)
	case Implies:
		if positive {
			left, leftOK := normalizeDatatypeBoolean(value.Left, false)
			right, rightOK := normalizeDatatypeBoolean(value.Right, true)
			return unionDatatypeBooleanBranches(left, right), leftOK && rightOK
		}
		left, leftOK := normalizeDatatypeBoolean(value.Left, true)
		right, rightOK := normalizeDatatypeBoolean(value.Right, false)
		return combineDatatypeBooleanBranches(left, right), leftOK && rightOK
	case Iff:
		return normalizeDatatypeEquivalence(value.Left, value.Right, positive)
	case If[BoolSort]:
		conditionTrue, firstOK := normalizeDatatypeBoolean(value.Condition, true)
		conditionFalse, secondOK := normalizeDatatypeBoolean(value.Condition, false)
		thenBranch, thirdOK := normalizeDatatypeBoolean(value.Then, positive)
		elseBranch, fourthOK := normalizeDatatypeBoolean(value.Else, positive)
		return unionDatatypeBooleanBranches(
			combineDatatypeBooleanBranches(conditionTrue, thenBranch),
			combineDatatypeBooleanBranches(conditionFalse, elseBranch),
		), firstOK && secondOK && thirdOK && fourthOK
	case BooleanConjunction:
		items, negated := value.values()
		result := datatypeBooleanBranches{}
		if positive {
			result.values = append(result.values, datatypeBooleanBranch{})
		}
		for index, item := range items {
			part, ok := normalizeDatatypeBoolean(item, positive != negated[index])
			if !ok {
				return datatypeBooleanBranches{}, false
			}
			if positive {
				result = combineDatatypeBooleanBranches(result, part)
			} else {
				result = unionDatatypeBooleanBranches(result, part)
			}
		}
		return result, true
	case Equal:
		left, leftOK := value.Left.(Term[BoolSort])
		right, rightOK := value.Right.(Term[BoolSort])
		if leftOK && rightOK {
			return normalizeDatatypeEquivalence(left, right, positive)
		}
		return datatypeBooleanAtom(term, positive)
	default:
		return datatypeBooleanAtom(term, positive)
	}
}

func normalizeDatatypeEquivalence(left, right Term[BoolSort], positive bool) (datatypeBooleanBranches, bool) {
	leftTrue, firstOK := normalizeDatatypeBoolean(left, true)
	rightTrue, secondOK := normalizeDatatypeBoolean(right, true)
	leftFalse, thirdOK := normalizeDatatypeBoolean(left, false)
	rightFalse, fourthOK := normalizeDatatypeBoolean(right, false)
	if positive {
		return unionDatatypeBooleanBranches(
			combineDatatypeBooleanBranches(leftTrue, rightTrue),
			combineDatatypeBooleanBranches(leftFalse, rightFalse),
		), firstOK && secondOK && thirdOK && fourthOK
	}
	return unionDatatypeBooleanBranches(
		combineDatatypeBooleanBranches(leftTrue, rightFalse),
		combineDatatypeBooleanBranches(leftFalse, rightTrue),
	), firstOK && secondOK && thirdOK && fourthOK
}

func normalizeDatatypeBooleanMany(terms []Term[BoolSort], childPositive, conjunction bool) (datatypeBooleanBranches, bool) {
	result := datatypeBooleanBranches{}
	if conjunction {
		result.values = append(result.values, datatypeBooleanBranch{})
	}
	for _, term := range terms {
		part, ok := normalizeDatatypeBoolean(term, childPositive)
		if !ok {
			return datatypeBooleanBranches{}, false
		}
		if conjunction {
			result = combineDatatypeBooleanBranches(result, part)
		} else {
			result = unionDatatypeBooleanBranches(result, part)
		}
		if result.exhausted {
			break
		}
	}
	return result, true
}

func datatypeBooleanAtom(term Term[BoolSort], positive bool) (datatypeBooleanBranches, bool) {
	if !containsDatatypeTheory(term) {
		return datatypeBooleanBranches{}, false
	}
	if !positive {
		term = Not{Value: term}
	}
	return datatypeBooleanBranches{values: []datatypeBooleanBranch{{atoms: []Term[BoolSort]{term}}}}, true
}

func unionDatatypeBooleanBranches(left, right datatypeBooleanBranches) datatypeBooleanBranches {
	if left.exhausted || right.exhausted || len(left.values)+len(right.values) > datatypeBooleanBranchLimit {
		return datatypeBooleanBranches{exhausted: true}
	}
	values := make([]datatypeBooleanBranch, 0, len(left.values)+len(right.values))
	values = append(values, left.values...)
	values = append(values, right.values...)
	return datatypeBooleanBranches{values: values}
}

func combineDatatypeBooleanBranches(left, right datatypeBooleanBranches) datatypeBooleanBranches {
	if left.exhausted || right.exhausted || len(left.values) != 0 && len(right.values) > datatypeBooleanBranchLimit/len(left.values) {
		return datatypeBooleanBranches{exhausted: true}
	}
	values := make([]datatypeBooleanBranch, 0, len(left.values)*len(right.values))
	for _, first := range left.values {
		for _, second := range right.values {
			atoms := make([]Term[BoolSort], 0, len(first.atoms)+len(second.atoms))
			atoms = append(atoms, first.atoms...)
			atoms = append(atoms, second.atoms...)
			values = append(values, datatypeBooleanBranch{atoms: atoms})
		}
	}
	return datatypeBooleanBranches{values: values}
}
