package smt

import "strings"

// CompactStringReplaceEquality is the allocation-light representation of
// str.replace(x, source, replacement) = target for a direct symbol x and
// ground string operands.
type CompactStringReplaceEquality struct {
	SymbolID    int
	SymbolName  string
	Source      string
	Replacement string
	Target      string
	All         bool
}

func (CompactStringReplaceEquality) isTerm(BoolSort) {}

type groundStringReplaceConstraint struct {
	id                int
	equalityCount     int
	equalities        [4]CompactStringReplaceEquality
	overflow          []CompactStringReplaceEquality
	indexedCount      int
	indexed           [4]CompactStringIndexedEquality
	indexedOverflow   []CompactStringIndexedEquality
	predicateCount    int
	predicates        [4]Term[BoolSort]
	predicateOverflow []Term[BoolSort]
}

type groundStringReplaceConstraints struct {
	count    int
	inline   [4]groundStringReplaceConstraint
	overflow []groundStringReplaceConstraint
}

func (constraints *groundStringReplaceConstraints) at(index int) *groundStringReplaceConstraint {
	if constraints.overflow != nil {
		return &constraints.overflow[index]
	}
	return &constraints.inline[index]
}

func (constraints *groundStringReplaceConstraints) findOrAppend(id int) *groundStringReplaceConstraint {
	for index := 0; index < constraints.count; index++ {
		if constraints.at(index).id == id {
			return constraints.at(index)
		}
	}
	if constraints.overflow != nil {
		constraints.overflow = append(constraints.overflow, groundStringReplaceConstraint{id: id})
		constraints.count++
		return &constraints.overflow[constraints.count-1]
	}
	if constraints.count < len(constraints.inline) {
		constraints.inline[constraints.count].id = id
		constraints.count++
		return &constraints.inline[constraints.count-1]
	}
	constraints.overflow = make(
		[]groundStringReplaceConstraint, constraints.count, constraints.count*2,
	)
	copy(constraints.overflow, constraints.inline[:])
	constraints.overflow = append(constraints.overflow, groundStringReplaceConstraint{id: id})
	constraints.count++
	return &constraints.overflow[constraints.count-1]
}

func (constraint *groundStringReplaceConstraint) equalityAt(index int) CompactStringReplaceEquality {
	if constraint.overflow != nil {
		return constraint.overflow[index]
	}
	return constraint.equalities[index]
}

func (constraint *groundStringReplaceConstraint) append(equality CompactStringReplaceEquality) {
	if constraint.overflow != nil {
		constraint.overflow = append(constraint.overflow, equality)
		constraint.equalityCount++
		return
	}
	if constraint.equalityCount < len(constraint.equalities) {
		constraint.equalities[constraint.equalityCount] = equality
		constraint.equalityCount++
		return
	}
	constraint.overflow = make(
		[]CompactStringReplaceEquality, constraint.equalityCount, constraint.equalityCount*2,
	)
	copy(constraint.overflow, constraint.equalities[:])
	constraint.overflow = append(constraint.overflow, equality)
	constraint.equalityCount++
}

func (constraint *groundStringReplaceConstraint) indexedAt(index int) CompactStringIndexedEquality {
	if constraint.indexedOverflow != nil {
		return constraint.indexedOverflow[index]
	}
	return constraint.indexed[index]
}

func (constraint *groundStringReplaceConstraint) appendIndexed(equality CompactStringIndexedEquality) {
	if constraint.indexedOverflow != nil {
		constraint.indexedOverflow = append(constraint.indexedOverflow, equality)
		constraint.indexedCount++
		return
	}
	if constraint.indexedCount < len(constraint.indexed) {
		constraint.indexed[constraint.indexedCount] = equality
		constraint.indexedCount++
		return
	}
	constraint.indexedOverflow = make(
		[]CompactStringIndexedEquality, constraint.indexedCount, constraint.indexedCount*2,
	)
	copy(constraint.indexedOverflow, constraint.indexed[:])
	constraint.indexedOverflow = append(constraint.indexedOverflow, equality)
	constraint.indexedCount++
}

func (constraint *groundStringReplaceConstraint) predicateAt(index int) Term[BoolSort] {
	if constraint.predicateOverflow != nil {
		return constraint.predicateOverflow[index]
	}
	return constraint.predicates[index]
}

func (constraint *groundStringReplaceConstraint) appendPredicate(predicate Term[BoolSort]) {
	if constraint.predicateOverflow != nil {
		constraint.predicateOverflow = append(constraint.predicateOverflow, predicate)
		constraint.predicateCount++
		return
	}
	if constraint.predicateCount < len(constraint.predicates) {
		constraint.predicates[constraint.predicateCount] = predicate
		constraint.predicateCount++
		return
	}
	constraint.predicateOverflow = make(
		[]Term[BoolSort], constraint.predicateCount, constraint.predicateCount*2,
	)
	copy(constraint.predicateOverflow, constraint.predicates[:])
	constraint.predicateOverflow = append(constraint.predicateOverflow, predicate)
	constraint.predicateCount++
}

func solveGroundStringReplaceEqualities(assertions []Term[BoolSort]) (checkOutcome, bool) {
	var storage boundedWordEquationConjuncts
	for _, assertion := range assertions {
		appendBoundedWordEquationConjunct(assertion, &storage)
	}
	conjuncts := storage.values()
	if len(conjuncts) == 0 {
		return checkOutcome{}, false
	}
	var constraints groundStringReplaceConstraints
	for _, conjunct := range conjuncts {
		if ground, known := evaluateStringBoolean(conjunct, stringModel{}, integerModel{}); known {
			if !ground {
				return checkOutcome{status: checkUnsat}, true
			}
			continue
		}
		equality, ok := groundStringReplaceEquality(conjunct)
		if ok {
			constraints.findOrAppend(equality.SymbolID).append(equality)
			continue
		}
		indexed, indexedOK := compactGroundIndexedStringEquality(conjunct)
		if indexedOK {
			constraints.findOrAppend(indexed.SymbolID).appendIndexed(indexed)
			continue
		}
		if !isBoundedWordEquationPredicate(conjunct) {
			return checkOutcome{}, false
		}
		var symbols stringSymbols
		collectStringSymbolsBoolean(conjunct, &symbols)
		if symbols.count != 1 || len(symbols.overflow) != 0 {
			return checkOutcome{}, false
		}
		constraints.findOrAppend(symbols.inline[0]).appendPredicate(conjunct)
	}
	if constraints.count == 0 {
		return checkOutcome{}, false
	}
	for index := 0; index < constraints.count; index++ {
		if constraints.at(index).equalityCount == 0 {
			return checkOutcome{}, false
		}
	}
	var model stringModel
	for index := 0; index < constraints.count; index++ {
		constraint := constraints.at(index)
		candidate, found, complete := groundStringReplacePreimage(constraint)
		if !complete {
			return checkOutcome{
				status: checkUnknown,
				reason: ResourceLimit{Limit: compactStringWordEquationSearchLimit},
			}, true
		}
		if !found {
			return checkOutcome{status: checkUnsat}, true
		}
		model.set(constraint.id, candidate)
	}
	return checkOutcome{status: checkSat, strings: model}, true
}

func groundStringReplaceEquality(term Term[BoolSort]) (CompactStringReplaceEquality, bool) {
	if compact, ok := term.(CompactStringReplaceEquality); ok {
		return compact, true
	}
	equality, ok := term.(Equal)
	if !ok {
		return CompactStringReplaceEquality{}, false
	}
	if result, ok := groundStringReplaceEqualitySides(equality.Left, equality.Right); ok {
		return result, true
	}
	return groundStringReplaceEqualitySides(equality.Right, equality.Left)
}

func groundStringReplaceEqualitySides(derived, target any) (CompactStringReplaceEquality, bool) {
	replacement, ok := derived.(stringReplace[StringSort])
	all := false
	if !ok {
		replacementAll, replaceAll := derived.(stringReplaceAll[StringSort])
		if !replaceAll {
			return CompactStringReplaceEquality{}, false
		}
		replacement = stringReplace[StringSort]{
			value:       replacementAll.value,
			source:      replacementAll.source,
			replacement: replacementAll.replacement,
		}
		all = true
	}
	id, symbol := stringSymbolID(replacement.value)
	source, sourceGround := evaluateString(replacement.source, stringModel{}, integerModel{})
	newValue, replacementGround := evaluateString(replacement.replacement, stringModel{}, integerModel{})
	targetTerm, targetString := target.(Term[StringSort])
	if !symbol || !sourceGround || !replacementGround || !targetString {
		return CompactStringReplaceEquality{}, false
	}
	targetValue, targetGround := evaluateString(targetTerm, stringModel{}, integerModel{})
	if !targetGround {
		return CompactStringReplaceEquality{}, false
	}
	return CompactStringReplaceEquality{
		SymbolID:    id,
		Source:      source,
		Replacement: newValue,
		Target:      targetValue,
		All:         all,
	}, true
}

func groundStringReplacePreimage(
	constraint *groundStringReplaceConstraint,
) (string, bool, bool) {
	anchor := constraint.equalityAt(0)
	steps := 0
	try := func(candidate string) (string, bool, bool) {
		steps++
		if steps > compactStringWordEquationSearchLimit {
			return "", false, false
		}
		// The anchor candidate is constructed from its exact inverse rule, so
		// only the remaining equalities need evaluation. Besides avoiding
		// redundant work, this prevents strings.Replace from allocating a
		// throwaway copy on the common single-constraint path.
		for index := 1; index < constraint.equalityCount; index++ {
			equality := constraint.equalityAt(index)
			if !compactStringReplacementEquals(candidate, equality) {
				return "", false, true
			}
		}
		for index := 0; index < constraint.indexedCount; index++ {
			if !evaluateCompactIndexedStringEquality(
				constraint.indexedAt(index), candidate,
			) {
				return "", false, true
			}
		}
		if constraint.predicateCount > 0 {
			var model stringModel
			model.set(constraint.id, candidate)
			for index := 0; index < constraint.predicateCount; index++ {
				accepted, known := evaluateStringBoolean(
					constraint.predicateAt(index), model, integerModel{},
				)
				if !known {
					return "", false, false
				}
				if !accepted {
					return "", false, true
				}
			}
		}
		return candidate, true, true
	}
	if anchor.All {
		return groundStringReplaceAllPreimage(anchor, try)
	}
	if anchor.Source == "" {
		if !strings.HasPrefix(anchor.Target, anchor.Replacement) {
			return "", false, true
		}
		return try(anchor.Target[len(anchor.Replacement):])
	}
	if !strings.Contains(anchor.Target, anchor.Source) {
		if candidate, found, complete := try(anchor.Target); found || !complete {
			return candidate, found, complete
		}
	}
	if anchor.Replacement == "" {
		for offset := 0; offset <= len(anchor.Target); offset++ {
			if !stringWordEquationBoundary(anchor.Target, offset) {
				continue
			}
			prefix := anchor.Target[:offset]
			if strings.Contains(prefix, anchor.Source) {
				continue
			}
			candidate := prefix + anchor.Source + anchor.Target[offset:]
			if result, found, complete := try(candidate); found || !complete {
				return result, found, complete
			}
		}
		return "", false, true
	}
	for search := 0; search <= len(anchor.Target); {
		relative := strings.Index(anchor.Target[search:], anchor.Replacement)
		if relative < 0 {
			break
		}
		offset := search + relative
		prefix := anchor.Target[:offset]
		if !strings.Contains(prefix, anchor.Source) {
			candidate := prefix + anchor.Source + anchor.Target[offset+len(anchor.Replacement):]
			if result, found, complete := try(candidate); found || !complete {
				return result, found, complete
			}
		}
		search = offset + 1
	}
	return "", false, true
}

func compactStringReplacementEquals(
	candidate string, equality CompactStringReplaceEquality,
) bool {
	if !equality.All {
		return strings.Replace(
			candidate, equality.Source, equality.Replacement, 1,
		) == equality.Target
	}
	if equality.Source == "" {
		return candidate == equality.Target
	}
	inputOffset := 0
	targetOffset := 0
	for {
		relative := strings.Index(candidate[inputOffset:], equality.Source)
		if relative < 0 {
			return targetOffset <= len(equality.Target) &&
				candidate[inputOffset:] == equality.Target[targetOffset:]
		}
		match := inputOffset + relative
		literal := candidate[inputOffset:match]
		if targetOffset > len(equality.Target) ||
			!strings.HasPrefix(equality.Target[targetOffset:], literal) {
			return false
		}
		targetOffset += len(literal)
		if targetOffset > len(equality.Target) ||
			!strings.HasPrefix(equality.Target[targetOffset:], equality.Replacement) {
			return false
		}
		targetOffset += len(equality.Replacement)
		inputOffset = match + len(equality.Source)
	}
}

// groundStringReplaceAllPreimage enumerates the finite inverse parses induced
// by a nonempty replacement. Every target boundary can either be copied
// literally or consume one replacement and emit the source. Exact forward
// evaluation rejects parses whose copied text accidentally contains source.
//
// An empty replacement can have an unbounded inverse language. The identity
// candidate remains useful and exact; broader inversion is deliberately
// reported incomplete until the deletion transducer is represented directly.
func groundStringReplaceAllPreimage(
	anchor CompactStringReplaceEquality,
	try func(string) (string, bool, bool),
) (string, bool, bool) {
	accept := func(candidate string) (string, bool, bool) {
		if !compactStringReplacementEquals(candidate, anchor) {
			return "", false, true
		}
		return try(candidate)
	}
	if anchor.Source == "" {
		return accept(anchor.Target)
	}
	if candidate, found, complete := accept(anchor.Target); found || !complete {
		return candidate, found, complete
	}
	if anchor.Replacement == "" {
		return "", false, false
	}
	// Replacing every visible output occurrence is the common inverse and
	// avoids constructing the search tree when it is already exact.
	direct := anchor.Source
	if anchor.Target != anchor.Replacement {
		direct = strings.ReplaceAll(anchor.Target, anchor.Replacement, anchor.Source)
	}
	if candidate, found, complete := accept(direct); found || !complete {
		return candidate, found, complete
	}
	states := 0
	return enumerateStringReplaceAllPreimages(anchor, accept, 0, "", &states)
}

func enumerateStringReplaceAllPreimages(
	anchor CompactStringReplaceEquality,
	try func(string) (string, bool, bool),
	offset int,
	prefix string,
	states *int,
) (string, bool, bool) {
	*states = *states + 1
	if *states > compactStringWordEquationSearchLimit {
		return "", false, false
	}
	if offset == len(anchor.Target) {
		return try(prefix)
	}
	if strings.HasPrefix(anchor.Target[offset:], anchor.Replacement) {
		if candidate, found, complete := enumerateStringReplaceAllPreimages(
			anchor,
			try,
			offset+len(anchor.Replacement),
			prefix+anchor.Source,
			states,
		); found || !complete {
			return candidate, found, complete
		}
	}
	width := stringCodePointWidth(anchor.Target, offset)
	return enumerateStringReplaceAllPreimages(
		anchor,
		try,
		offset+width,
		prefix+anchor.Target[offset:offset+width],
		states,
	)
}
