package smt

// groundArrayCongruence implements the first symbolic QF_ALIA layer: equal
// array symbols have congruent reads at each exact integer index.
type groundArrayCongruence struct {
	arrayCount             int
	arrayIDs               [8]int
	arrayParents           [8]int
	indexCount             int
	indexIDs               [8]int
	indexParents           [8]int
	indexHasValue          [8]bool
	indexValues            [8]IntegerValue
	indexConflict          bool
	observedCount          int
	observed               [16]groundArrayIndex
	bridgeCount            int
	bridges                [8][2]int
	constantBridgeCount    int
	constantBridges        [8]groundArrayConstantBridge
	constantBridgeConflict bool
	readCount              int
	reads                  [16]groundArrayReadKey
	readParents            [16]int
	disequalityCount       int
	disequalities          [8][2]int
	extensionalCount       int
	extensional            [8]groundArrayExtensionalDisequality
	extensionalPairs       [32][2]int
	extensionalPairsCount  int
}

type groundArrayConstantBridge struct {
	arrayID int
	value   IntegerValue
}

type groundArrayExtensionalDisequality struct {
	start int
	count int
}

type groundArrayReadKey struct {
	constant bool
	arrayID  int
	index    groundArrayIndex
	value    IntegerValue
}

type groundArrayIndex struct {
	symbol bool
	id     int
	value  IntegerValue
}

func solveGroundArrayCongruence(assertions []Term[BoolSort]) (checkOutcome, bool) {
	return solveGroundArrayCongruenceWithIntegers(assertions, integerModel{})
}

func solveGroundArrayCongruenceWithIntegers(assertions []Term[BoolSort], seed integerModel) (checkOutcome, bool) {
	problem := groundArrayCongruence{}
	for _, assertion := range assertions {
		if !problem.collectArrays(assertion, false) {
			return checkOutcome{}, false
		}
	}
	if problem.indexConflict {
		return checkOutcome{status: checkUnsat}, true
	}
	for _, assertion := range assertions {
		holds, known := problem.collectElements(assertion, false)
		if !known {
			return checkOutcome{}, false
		}
		if !holds {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	if problem.constantBridgeConflict {
		return checkOutcome{status: checkUnsat}, true
	}
	for _, pair := range problem.disequalities[:problem.disequalityCount] {
		if problem.readRoot(pair[0]) == problem.readRoot(pair[1]) {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	for _, group := range problem.extensional[:problem.extensionalCount] {
		different := false
		for _, pair := range problem.extensionalPairs[group.start : group.start+group.count] {
			if problem.readRoot(pair[0]) != problem.readRoot(pair[1]) {
				different = true
				break
			}
		}
		if !different {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	for left := 0; left < problem.readCount; left++ {
		if !problem.reads[left].constant {
			continue
		}
		for right := left + 1; right < problem.readCount; right++ {
			if problem.reads[right].constant && problem.readRoot(left) == problem.readRoot(right) && CompareIntegerValue(problem.reads[left].value, problem.reads[right].value) != 0 {
				return checkOutcome{status: checkUnsat}, true
			}
		}
	}
	integers, arrays := problem.model(seed)
	return checkOutcome{status: checkSat, integers: integers, arrays: arrays}, true
}

func (problem *groundArrayCongruence) model(seed integerModel) (integerModel, *integerArrayModel) {
	integers := seed
	for position := 0; position < problem.indexCount; position++ {
		id := problem.indexIDs[position]
		normalized := problem.normalizeIndex(groundArrayIndex{symbol: true, id: id})
		value := normalized.value
		if normalized.symbol {
			rootPosition := problem.indexRootPosition(problem.indexPosition(normalized.id))
			value = NewIntegerValue(int64(1000 + rootPosition))
		}
		if _, exists := integers.lookup(id); !exists {
			integers.set(id, value)
		}
	}

	arrays := &integerArrayModel{}
	for position := 0; position < problem.arrayCount; position++ {
		id := problem.arrayIDs[position]
		root := problem.modelArrayClass(id)
		rootPosition := 0
		for candidate := 0; candidate < problem.arrayCount; candidate++ {
			if problem.modelArrayClass(problem.arrayIDs[candidate]) == root {
				rootPosition = candidate
				break
			}
		}
		defaultValue := NewIntegerValue(int64(rootPosition + 1))
		if constrained, ok := problem.constantBridgeValue(id); ok {
			defaultValue = constrained
		}
		arrays.setDefault(id, defaultValue)
	}

	for position := 0; position < problem.readCount; position++ {
		key := problem.reads[position]
		if key.constant {
			continue
		}
		root := problem.readRoot(position)
		value := NewIntegerValue(int64(10000 + root))
		for candidate := 0; candidate < problem.readCount; candidate++ {
			if problem.readRoot(candidate) == root && problem.reads[candidate].constant {
				value = problem.reads[candidate].value
				break
			}
		}
		index := problem.normalizeIndex(key.index).value
		if normalized := problem.normalizeIndex(key.index); normalized.symbol {
			resolved, ok := integers.lookup(normalized.id)
			if !ok {
				continue
			}
			index = resolved
		}
		for arrayPosition := 0; arrayPosition < problem.arrayCount; arrayPosition++ {
			id := problem.arrayIDs[arrayPosition]
			if problem.arrayRoot(id) == key.arrayID {
				arrays.set(id, index, value)
			}
		}
	}
	return integers, arrays
}

func (problem *groundArrayCongruence) modelArrayClass(id int) int {
	class := problem.arrayRoot(id)
	changed := true
	for changed {
		changed = false
		for _, bridge := range problem.bridges[:problem.bridgeCount] {
			left, right := problem.arrayRoot(bridge[0]), problem.arrayRoot(bridge[1])
			if left == class && right < class {
				class, changed = right, true
			}
			if right == class && left < class {
				class, changed = left, true
			}
		}
	}
	return class
}

func (problem *groundArrayCongruence) collectArrays(term Term[BoolSort], negated bool) bool {
	switch value := term.(type) {
	case And:
		if negated {
			return false
		}
		for _, item := range value.Values {
			if !problem.collectArrays(item, false) {
				return false
			}
		}
		return true
	case BooleanConjunction:
		if negated {
			return false
		}
		items, itemNegated := value.values()
		for index, item := range items {
			if !problem.collectArrays(item, itemNegated[index]) {
				return false
			}
		}
		return true
	case Not:
		return problem.collectArrays(value.Value, !negated)
	case Equal:
		if !problem.collectArrayObservations(value.Left) || !problem.collectArrayObservations(value.Right) {
			return false
		}
		if !isArrayTerm(value.Left) && !isArrayTerm(value.Right) {
			left, leftOK := problem.arrayIndex(value.Left)
			right, rightOK := problem.arrayIndex(value.Right)
			if leftOK && rightOK && !negated {
				problem.unionIndex(left, right)
			}
			return true
		}
		left, leftOK := arraySymbolID(value.Left)
		right, rightOK := arraySymbolID(value.Right)
		if leftOK && rightOK {
			problem.ensureArray(left)
			problem.ensureArray(right)
			if !negated {
				problem.unionArray(left, right)
			}
			return true
		}
		leftExpression, leftExpressionOK := problem.arrayExpression(value.Left)
		rightExpression, rightExpressionOK := problem.arrayExpression(value.Right)
		if !leftExpressionOK || !rightExpressionOK {
			return false
		}
		if !negated {
			problem.recordExpressionBaseConstraint(leftExpression, rightExpression)
		}
		return true
	case ArrayEqualityRelation:
		effectiveNegated := value.Negated != negated
		problem.ensureArray(value.LeftID)
		problem.ensureArray(value.RightID)
		if !effectiveNegated {
			problem.unionArray(value.LeftID, value.RightID)
		}
		return true
	case ArrayReadRelation:
		problem.ensureArray(value.LeftID)
		problem.ensureArray(value.RightID)
		return problem.observeIndex(groundArrayIndex{value: value.LeftIndex}) && problem.observeIndex(groundArrayIndex{value: value.RightIndex})
	case ArrayCongruenceConjunction:
		if negated {
			return false
		}
		return problem.collectArrays(value.Equality, false) && problem.collectArrays(value.Read, false)
	case ArrayStoreEqualityRelation:
		problem.ensureArray(value.LeftID)
		problem.ensureArray(value.RightID)
		effectiveNegated := value.Negated != negated
		if !effectiveNegated {
			problem.recordBridge(value.LeftID, value.RightID)
		}
		return problem.observeIndex(groundArrayIndex{value: value.LeftIndex}) && problem.observeIndex(groundArrayIndex{value: value.RightIndex})
	case ArrayStoreBridgeReadConjunction:
		if negated {
			return false
		}
		return problem.collectArrays(value.Store, false) && problem.collectArrays(value.Read, false)
	case ArrayConstantEqualityRelation:
		problem.ensureArray(value.ArrayID)
		effectiveNegated := value.Negated != negated
		if !effectiveNegated {
			problem.recordConstantBridge(value.ArrayID, value.Default)
		}
		return true
	case ArrayReadValueRelation:
		problem.ensureArray(value.ArrayID)
		return problem.observeIndex(groundArrayIndex{value: value.Index})
	case ArrayConstantReadConjunction:
		if negated {
			return false
		}
		return problem.collectArrays(value.Equality, false) && problem.collectArrays(value.Read, false)
	case ArrayStoreReadValueRelation:
		problem.ensureArray(value.ArrayID)
		storeIndex, storeOK := problem.arrayIndex(integerVariable[IntSort]{iD: value.StoreIndexID})
		readIndex, readOK := problem.arrayIndex(integerVariable[IntSort]{iD: value.ReadIndexID})
		return storeOK && readOK && problem.observeIndex(storeIndex) && problem.observeIndex(readIndex)
	case Bool:
		return true
	}
	return false
}

func (problem *groundArrayCongruence) collectElements(term Term[BoolSort], negated bool) (bool, bool) {
	switch value := term.(type) {
	case Bool:
		return value.Value != negated, true
	case And:
		if negated {
			return false, false
		}
		for _, item := range value.Values {
			holds, known := problem.collectElements(item, false)
			if !known || !holds {
				return holds, known
			}
		}
		return true, true
	case BooleanConjunction:
		if negated {
			return false, false
		}
		items, itemNegated := value.values()
		for index, item := range items {
			holds, known := problem.collectElements(item, itemNegated[index])
			if !known || !holds {
				return holds, known
			}
		}
		return true, true
	case Not:
		return problem.collectElements(value.Value, !negated)
	case Equal:
		if isArrayTerm(value.Left) || isArrayTerm(value.Right) {
			left, leftOK := arraySymbolID(value.Left)
			right, rightOK := arraySymbolID(value.Right)
			if leftOK && rightOK {
				equal := problem.arrayRoot(left) == problem.arrayRoot(right)
				if negated && equal {
					return false, true
				}
				return true, true
			}
			return problem.arrayExpressionEquality(value.Left, value.Right, negated)
		}
		if left, leftOK := problem.arrayIndex(value.Left); leftOK {
			if right, rightOK := problem.arrayIndex(value.Right); rightOK {
				equal := problem.indexEqual(left, right)
				return equal != negated, true
			}
		}
		return problem.elementEquality(value.Left, value.Right, negated)
	case ArrayEqualityRelation:
		effectiveNegated := value.Negated != negated
		equal := problem.arrayRoot(value.LeftID) == problem.arrayRoot(value.RightID)
		if effectiveNegated && equal {
			return false, true
		}
		return true, true
	case ArrayReadRelation:
		effectiveNegated := value.Negated != negated
		left := arrayReadInteger[IntSort]{arrayID: value.LeftID, index: value.LeftIndex}
		right := arrayReadInteger[IntSort]{arrayID: value.RightID, index: value.RightIndex}
		return problem.elementEquality(left, right, effectiveNegated)
	case ArrayCongruenceConjunction:
		if negated {
			return false, false
		}
		holds, known := problem.collectElements(value.Equality, false)
		if !known || !holds {
			return holds, known
		}
		return problem.collectElements(value.Read, false)
	case ArrayStoreEqualityRelation:
		left := groundArrayExpression{baseKind: 1, baseID: value.LeftID, updateCount: 1}
		left.indices[0], left.values[0] = groundArrayIndex{value: value.LeftIndex}, integerExact[IntSort]{value: value.LeftValue}
		right := groundArrayExpression{baseKind: 1, baseID: value.RightID, updateCount: 1}
		right.indices[0], right.values[0] = groundArrayIndex{value: value.RightIndex}, integerExact[IntSort]{value: value.RightValue}
		return problem.arrayExpressionsEquality(left, right, value.Negated != negated)
	case ArrayStoreBridgeReadConjunction:
		if negated {
			return false, false
		}
		holds, known := problem.collectElements(value.Store, false)
		if !known || !holds {
			return holds, known
		}
		return problem.collectElements(value.Read, false)
	case ArrayConstantEqualityRelation:
		symbol := groundArrayExpression{baseKind: 1, baseID: value.ArrayID}
		constant := groundArrayExpression{baseKind: 2, baseDefault: integerExact[IntSort]{value: value.Default}}
		return problem.arrayExpressionsEquality(symbol, constant, value.Negated != negated)
	case ArrayReadValueRelation:
		read := arrayReadInteger[IntSort]{arrayID: value.ArrayID, index: value.Index}
		constant := integerExact[IntSort]{value: value.Value}
		return problem.elementEquality(read, constant, value.Negated != negated)
	case ArrayConstantReadConjunction:
		if negated {
			return false, false
		}
		holds, known := problem.collectElements(value.Equality, false)
		if !known || !holds {
			return holds, known
		}
		return problem.collectElements(value.Read, false)
	case ArrayStoreReadValueRelation:
		read := arrayStoreReadInteger[IntSort]{arrayID: value.ArrayID, storeIndexID: value.StoreIndexID, readIndexID: value.ReadIndexID, value: value.StoredValue}
		constant := integerExact[IntSort]{value: value.ComparedValue}
		return problem.elementEquality(read, constant, value.Negated != negated)
	}
	return false, false
}

type groundArrayRead struct {
	arrayID int
	index   groundArrayIndex
}

type groundArrayExpression struct {
	baseKind    uint8
	baseID      int
	baseDefault any
	updateCount int
	indices     [8]groundArrayIndex
	values      [8]any
}

func (problem *groundArrayCongruence) collectArrayExpression(term any) bool {
	_, ok := problem.arrayExpression(term)
	return ok
}

func (problem *groundArrayCongruence) arrayExpression(term any) (groundArrayExpression, bool) {
	expression := groundArrayExpression{}
	for {
		if store, ok := term.(arrayStoreTerm); ok {
			if expression.updateCount == len(expression.indices) {
				return groundArrayExpression{}, false
			}
			base, indexTerm, value := store.arrayStoreParts()
			index, indexOK := problem.arrayIndex(indexTerm)
			if !indexOK {
				return groundArrayExpression{}, false
			}
			if !problem.observeIndex(index) {
				return groundArrayExpression{}, false
			}
			expression.indices[expression.updateCount] = index
			expression.values[expression.updateCount] = value
			expression.updateCount++
			term = base
			continue
		}
		if symbol, ok := term.(arraySymbolTerm); ok {
			expression.baseKind = 1
			expression.baseID, _ = symbol.arraySymbolParts()
			problem.ensureArray(expression.baseID)
			return expression, true
		}
		if constant, ok := term.(arrayConstantTerm); ok {
			expression.baseKind = 2
			expression.baseDefault = constant.arrayDefaultValue()
			return expression, true
		}
		return groundArrayExpression{}, false
	}
}

func (problem *groundArrayCongruence) arrayExpressionEquality(leftTerm, rightTerm any, negated bool) (bool, bool) {
	left, leftOK := problem.arrayExpression(leftTerm)
	right, rightOK := problem.arrayExpression(rightTerm)
	if !leftOK || !rightOK {
		return false, false
	}
	return problem.arrayExpressionsEquality(left, right, negated)
}

func (problem *groundArrayCongruence) arrayExpressionsEquality(left, right groundArrayExpression, negated bool) (bool, bool) {
	baseSame := false
	baseBridged := false
	if left.baseKind == 1 && right.baseKind == 1 {
		baseSame = problem.arrayRoot(left.baseID) == problem.arrayRoot(right.baseID)
		baseBridged = problem.modelArrayClass(left.baseID) == problem.modelArrayClass(right.baseID)
	} else if left.baseKind == 2 && right.baseKind == 2 {
		baseSame, _ = groundTermEqual(left.baseDefault, right.baseDefault)
		if !baseSame {
			if negated {
				return true, true
			}
			return false, true
		}
	} else if left.baseKind == 1 && right.baseKind == 2 {
		value, valueOK := exactIntegerConstant(right.baseDefault)
		if !valueOK {
			return false, false
		}
		if existing, ok := problem.constantBridgeValue(left.baseID); ok && CompareIntegerValue(existing, value) == 0 {
			baseBridged = true
		} else if !negated {
			baseBridged = problem.recordConstantBridge(left.baseID, value)
		}
	} else if left.baseKind == 2 && right.baseKind == 1 {
		value, valueOK := exactIntegerConstant(left.baseDefault)
		if !valueOK {
			return false, false
		}
		if existing, ok := problem.constantBridgeValue(right.baseID); ok && CompareIntegerValue(existing, value) == 0 {
			baseBridged = true
		} else if !negated {
			baseBridged = problem.recordConstantBridge(right.baseID, value)
		}
	}
	if !baseSame && !baseBridged {
		if negated {
			// Finite stores cannot hide a distinct base over the infinite Int domain.
			return true, true
		}
		if left.baseKind != 1 || right.baseKind != 1 {
			return false, false
		}
		if !problem.recordBridge(left.baseID, right.baseID) {
			return false, false
		}
		baseBridged = true
	}

	var indices [16]groundArrayIndex
	indexCount := 0
	addIndex := func(value groundArrayIndex) {
		for position := 0; position < indexCount; position++ {
			if problem.indexEqual(indices[position], value) {
				return
			}
		}
		indices[indexCount] = value
		indexCount++
	}
	for _, index := range left.indices[:left.updateCount] {
		addIndex(index)
	}
	for _, index := range right.indices[:right.updateCount] {
		addIndex(index)
	}
	if baseBridged {
		for _, index := range problem.observed[:problem.observedCount] {
			addIndex(index)
		}
	}
	if !negated {
		for _, index := range indices[:indexCount] {
			leftValue, leftRead := problem.arrayExpressionValue(left, index)
			rightValue, rightRead := problem.arrayExpressionValue(right, index)
			holds, known := problem.resolvedElementEquality(leftValue, leftRead, rightValue, rightRead, false)
			if !known || !holds {
				return holds, known
			}
		}
		return true, true
	}
	if indexCount == 0 {
		return false, true
	}
	if problem.extensionalCount == len(problem.extensional) {
		return false, false
	}
	start := problem.extensionalPairsCount
	for _, index := range indices[:indexCount] {
		leftValue, leftRead := problem.arrayExpressionValue(left, index)
		rightValue, rightRead := problem.arrayExpressionValue(right, index)
		if leftRead == nil {
			var ok bool
			leftValue, leftRead, ok = problem.groundArrayElement(leftValue)
			if !ok {
				problem.extensionalPairsCount = start
				return false, false
			}
		}
		if rightRead == nil {
			var ok bool
			rightValue, rightRead, ok = problem.groundArrayElement(rightValue)
			if !ok {
				problem.extensionalPairsCount = start
				return false, false
			}
		}
		leftConstant, leftConstantOK := exactIntegerConstant(leftValue)
		rightConstant, rightConstantOK := exactIntegerConstant(rightValue)
		if leftConstantOK && rightConstantOK {
			if CompareIntegerValue(leftConstant, rightConstant) != 0 {
				problem.extensionalPairsCount = start
				return true, true
			}
			continue
		}
		leftKey, leftKeyOK := problem.elementKey(leftValue, leftRead)
		rightKey, rightKeyOK := problem.elementKey(rightValue, rightRead)
		if !leftKeyOK || !rightKeyOK || problem.extensionalPairsCount == len(problem.extensionalPairs) {
			problem.extensionalPairsCount = start
			return false, false
		}
		if leftKey != rightKey {
			problem.extensionalPairs[problem.extensionalPairsCount] = [2]int{leftKey, rightKey}
			problem.extensionalPairsCount++
		}
	}
	count := problem.extensionalPairsCount - start
	if count == 0 {
		return false, true
	}
	problem.extensional[problem.extensionalCount] = groundArrayExtensionalDisequality{start: start, count: count}
	problem.extensionalCount++
	return true, true
}

func (problem *groundArrayCongruence) arrayExpressionValue(expression groundArrayExpression, index groundArrayIndex) (any, *groundArrayRead) {
	for position := 0; position < expression.updateCount; position++ {
		if problem.indexEqual(expression.indices[position], index) {
			return expression.values[position], nil
		}
	}
	if expression.baseKind == 1 {
		return nil, &groundArrayRead{arrayID: expression.baseID, index: index}
	}
	return expression.baseDefault, nil
}

func (problem *groundArrayCongruence) collectArrayObservations(term any) bool {
	if read, ok := term.(arrayReadInteger[IntSort]); ok {
		problem.ensureArray(read.arrayID)
		return problem.observeIndex(groundArrayIndex{value: read.index})
	}
	if read, ok := term.(arrayStoreReadInteger[IntSort]); ok {
		problem.ensureArray(read.arrayID)
		storeIndex, storeOK := problem.arrayIndex(integerVariable[IntSort]{iD: read.storeIndexID})
		readIndex, readOK := problem.arrayIndex(integerVariable[IntSort]{iD: read.readIndexID})
		return storeOK && readOK && problem.observeIndex(storeIndex) && problem.observeIndex(readIndex)
	}
	if selection, ok := term.(arraySelectionTerm); ok {
		array, indexTerm := selection.arraySelectionParts()
		index, indexOK := problem.arrayIndex(indexTerm)
		return indexOK && problem.observeIndex(index) && problem.collectArrayExpression(array)
	}
	if isArrayTerm(term) {
		return problem.collectArrayExpression(term)
	}
	return true
}

func (problem *groundArrayCongruence) observeIndex(value groundArrayIndex) bool {
	for position := 0; position < problem.observedCount; position++ {
		if problem.indexEqual(problem.observed[position], value) {
			return true
		}
	}
	if problem.observedCount == len(problem.observed) {
		return false
	}
	problem.observed[problem.observedCount] = value
	problem.observedCount++
	return true
}

func (problem *groundArrayCongruence) recordBridge(left, right int) bool {
	left, right = problem.arrayRoot(left), problem.arrayRoot(right)
	for _, bridge := range problem.bridges[:problem.bridgeCount] {
		if bridge == [2]int{left, right} || bridge == [2]int{right, left} {
			return true
		}
	}
	if problem.bridgeCount == len(problem.bridges) {
		return false
	}
	leftValue, leftHasValue := problem.constantBridgeValue(left)
	rightValue, rightHasValue := problem.constantBridgeValue(right)
	problem.bridges[problem.bridgeCount] = [2]int{left, right}
	problem.bridgeCount++
	if leftHasValue && rightHasValue && CompareIntegerValue(leftValue, rightValue) != 0 {
		problem.constantBridgeConflict = true
	}
	return true
}

func (problem *groundArrayCongruence) recordExpressionBaseConstraint(left, right groundArrayExpression) {
	if left.baseKind == 1 && right.baseKind == 1 {
		problem.recordBridge(left.baseID, right.baseID)
		return
	}
	if left.baseKind == 1 && right.baseKind == 2 {
		if value, ok := exactIntegerConstant(right.baseDefault); ok {
			problem.recordConstantBridge(left.baseID, value)
		}
		return
	}
	if left.baseKind == 2 && right.baseKind == 1 {
		if value, ok := exactIntegerConstant(left.baseDefault); ok {
			problem.recordConstantBridge(right.baseID, value)
		}
	}
}

func (problem *groundArrayCongruence) recordConstantBridge(id int, value IntegerValue) bool {
	class := problem.modelArrayClass(id)
	for _, bridge := range problem.constantBridges[:problem.constantBridgeCount] {
		if problem.modelArrayClass(bridge.arrayID) == class {
			if CompareIntegerValue(bridge.value, value) != 0 {
				problem.constantBridgeConflict = true
			}
			return true
		}
	}
	if problem.constantBridgeCount == len(problem.constantBridges) {
		return false
	}
	problem.constantBridges[problem.constantBridgeCount] = groundArrayConstantBridge{arrayID: id, value: value}
	problem.constantBridgeCount++
	return true
}

func (problem *groundArrayCongruence) constantBridgeValue(id int) (IntegerValue, bool) {
	class := problem.modelArrayClass(id)
	for _, bridge := range problem.constantBridges[:problem.constantBridgeCount] {
		if problem.modelArrayClass(bridge.arrayID) == class {
			return bridge.value, true
		}
	}
	return IntegerValue{}, false
}

func (problem *groundArrayCongruence) elementEquality(leftTerm, rightTerm any, negated bool) (bool, bool) {
	left, leftRead, leftOK := problem.groundArrayElement(leftTerm)
	right, rightRead, rightOK := problem.groundArrayElement(rightTerm)
	if !leftOK || !rightOK {
		return false, false
	}
	return problem.resolvedElementEquality(left, leftRead, right, rightRead, negated)
}

func (problem *groundArrayCongruence) resolvedElementEquality(left any, leftRead *groundArrayRead, right any, rightRead *groundArrayRead, negated bool) (bool, bool) {
	if leftRead == nil {
		var ok bool
		left, leftRead, ok = problem.groundArrayElement(left)
		if !ok {
			return false, false
		}
	}
	if rightRead == nil {
		var ok bool
		right, rightRead, ok = problem.groundArrayElement(right)
		if !ok {
			return false, false
		}
	}
	leftConstant, leftConstantOK := exactIntegerConstant(left)
	rightConstant, rightConstantOK := exactIntegerConstant(right)
	if leftConstantOK && rightConstantOK {
		return (CompareIntegerValue(leftConstant, rightConstant) == 0) != negated, true
	}
	leftKey, leftKeyOK := problem.elementKey(left, leftRead)
	rightKey, rightKeyOK := problem.elementKey(right, rightRead)
	if !leftKeyOK || !rightKeyOK {
		return false, false
	}
	if negated {
		if problem.disequalityCount == len(problem.disequalities) {
			return false, false
		}
		problem.disequalities[problem.disequalityCount] = [2]int{leftKey, rightKey}
		problem.disequalityCount++
		return true, true
	}
	problem.unionRead(leftKey, rightKey)
	return true, true
}

func (problem *groundArrayCongruence) groundArrayElement(term any) (any, *groundArrayRead, bool) {
	if read, ok := term.(arrayReadInteger[IntSort]); ok {
		problem.ensureArray(read.arrayID)
		return nil, &groundArrayRead{arrayID: read.arrayID, index: groundArrayIndex{value: read.index}}, true
	}
	if specialized, ok := term.(arrayStoreReadInteger[IntSort]); ok {
		storeIndex, storeOK := problem.arrayIndex(integerVariable[IntSort]{iD: specialized.storeIndexID})
		readIndex, readOK := problem.arrayIndex(integerVariable[IntSort]{iD: specialized.readIndexID})
		if !storeOK || !readOK {
			return nil, nil, false
		}
		if problem.indexEqual(storeIndex, readIndex) {
			return IntegerTerm(specialized.value), nil, true
		}
		return nil, &groundArrayRead{arrayID: specialized.arrayID, index: readIndex}, true
	}
	selection, ok := term.(arraySelectionTerm)
	if !ok {
		return term, nil, true
	}
	array, indexTerm := selection.arraySelectionParts()
	index, indexOK := problem.arrayIndex(indexTerm)
	if !indexOK {
		return nil, nil, false
	}
	for {
		if stored, ok := array.(arrayStoreTerm); ok {
			base, storedIndexTerm, storedValue := stored.arrayStoreParts()
			storedIndex, storedOK := problem.arrayIndex(storedIndexTerm)
			if !storedOK {
				return nil, nil, false
			}
			if problem.indexEqual(index, storedIndex) {
				return storedValue, nil, true
			}
			array = base
			continue
		}
		if constant, ok := array.(arrayConstantTerm); ok {
			return constant.arrayDefaultValue(), nil, true
		}
		id, ok := arraySymbolID(array)
		if !ok {
			return nil, nil, false
		}
		return nil, &groundArrayRead{arrayID: id, index: index}, true
	}
}

func arraySymbolID(term any) (int, bool) {
	value, ok := term.(arraySymbolTerm)
	if !ok {
		return 0, false
	}
	id, _ := value.arraySymbolParts()
	return id, true
}

func (problem *groundArrayCongruence) ensureArray(id int) {
	for index := 0; index < problem.arrayCount; index++ {
		if problem.arrayIDs[index] == id {
			return
		}
	}
	if problem.arrayCount == len(problem.arrayIDs) {
		return
	}
	problem.arrayIDs[problem.arrayCount] = id
	problem.arrayParents[problem.arrayCount] = problem.arrayCount
	problem.arrayCount++
}

func (problem *groundArrayCongruence) arrayRoot(id int) int {
	problem.ensureArray(id)
	index := -1
	for candidate := 0; candidate < problem.arrayCount; candidate++ {
		if problem.arrayIDs[candidate] == id {
			index = candidate
			break
		}
	}
	if index < 0 {
		return id
	}
	for problem.arrayParents[index] != index {
		problem.arrayParents[index] = problem.arrayParents[problem.arrayParents[index]]
		index = problem.arrayParents[index]
	}
	return problem.arrayIDs[index]
}

func (problem *groundArrayCongruence) unionArray(left, right int) {
	left, right = problem.arrayRoot(left), problem.arrayRoot(right)
	if left != right {
		leftIndex, rightIndex := -1, -1
		for index := 0; index < problem.arrayCount; index++ {
			if problem.arrayIDs[index] == left {
				leftIndex = index
			}
			if problem.arrayIDs[index] == right {
				rightIndex = index
			}
		}
		if leftIndex >= 0 && rightIndex >= 0 {
			problem.arrayParents[rightIndex] = leftIndex
		}
	}
}

func (problem *groundArrayCongruence) arrayIndex(term any) (groundArrayIndex, bool) {
	if value, ok := exactIntegerConstant(term); ok {
		return groundArrayIndex{value: value}, true
	}
	if symbol, ok := term.(IntSymbol); ok {
		return problem.ensureIndexSymbol(symbol.ID)
	}
	if symbol, ok := term.(integerVariable[IntSort]); ok {
		return problem.ensureIndexSymbol(symbol.iD)
	}
	return groundArrayIndex{}, false
}

func (problem *groundArrayCongruence) ensureIndexSymbol(id int) (groundArrayIndex, bool) {
	for index := 0; index < problem.indexCount; index++ {
		if problem.indexIDs[index] == id {
			return groundArrayIndex{symbol: true, id: id}, true
		}
	}
	if problem.indexCount == len(problem.indexIDs) {
		return groundArrayIndex{}, false
	}
	problem.indexIDs[problem.indexCount] = id
	problem.indexParents[problem.indexCount] = problem.indexCount
	problem.indexCount++
	return groundArrayIndex{symbol: true, id: id}, true
}

func (problem *groundArrayCongruence) indexPosition(id int) int {
	for index := 0; index < problem.indexCount; index++ {
		if problem.indexIDs[index] == id {
			return index
		}
	}
	return -1
}
func (problem *groundArrayCongruence) indexRootPosition(index int) int {
	for problem.indexParents[index] != index {
		problem.indexParents[index] = problem.indexParents[problem.indexParents[index]]
		index = problem.indexParents[index]
	}
	return index
}
func (problem *groundArrayCongruence) normalizeIndex(value groundArrayIndex) groundArrayIndex {
	if !value.symbol {
		return value
	}
	position := problem.indexPosition(value.id)
	if position < 0 {
		return value
	}
	root := problem.indexRootPosition(position)
	if problem.indexHasValue[root] {
		return groundArrayIndex{value: problem.indexValues[root]}
	}
	return groundArrayIndex{symbol: true, id: problem.indexIDs[root]}
}
func (problem *groundArrayCongruence) indexEqual(left, right groundArrayIndex) bool {
	left, right = problem.normalizeIndex(left), problem.normalizeIndex(right)
	if left.symbol || right.symbol {
		return left.symbol && right.symbol && left.id == right.id
	}
	return CompareIntegerValue(left.value, right.value) == 0
}
func (problem *groundArrayCongruence) unionIndex(left, right groundArrayIndex) {
	left, right = problem.normalizeIndex(left), problem.normalizeIndex(right)
	if !left.symbol && !right.symbol {
		return
	}
	if left.symbol && !right.symbol {
		position := problem.indexRootPosition(problem.indexPosition(left.id))
		if problem.indexHasValue[position] && CompareIntegerValue(problem.indexValues[position], right.value) != 0 {
			problem.indexConflict = true
			return
		}
		problem.indexHasValue[position], problem.indexValues[position] = true, right.value
		return
	}
	if !left.symbol && right.symbol {
		problem.unionIndex(right, left)
		return
	}
	leftPosition, rightPosition := problem.indexRootPosition(problem.indexPosition(left.id)), problem.indexRootPosition(problem.indexPosition(right.id))
	if leftPosition == rightPosition {
		return
	}
	problem.indexParents[rightPosition] = leftPosition
	if problem.indexHasValue[rightPosition] {
		if problem.indexHasValue[leftPosition] && CompareIntegerValue(problem.indexValues[leftPosition], problem.indexValues[rightPosition]) != 0 {
			problem.indexConflict = true
			return
		}
		problem.indexHasValue[leftPosition], problem.indexValues[leftPosition] = true, problem.indexValues[rightPosition]
	}
}

func (problem *groundArrayCongruence) elementKey(term any, read *groundArrayRead) (int, bool) {
	var key groundArrayReadKey
	if read != nil {
		key = groundArrayReadKey{arrayID: problem.arrayRoot(read.arrayID), index: problem.normalizeIndex(read.index)}
	} else if value, ok := exactIntegerConstant(term); ok {
		key = groundArrayReadKey{constant: true, value: value}
	} else {
		return 0, false
	}
	for index := 0; index < problem.readCount; index++ {
		prior := problem.reads[index]
		indicesEqual := prior.constant || problem.indexEqual(prior.index, key.index)
		if prior.constant == key.constant && prior.arrayID == key.arrayID && indicesEqual && CompareIntegerValue(prior.value, key.value) == 0 {
			return index, true
		}
	}
	if problem.readCount == len(problem.reads) {
		return 0, false
	}
	index := problem.readCount
	problem.reads[index] = key
	problem.readParents[index] = index
	problem.readCount++
	return index, true
}

func (problem *groundArrayCongruence) readRoot(key int) int {
	for problem.readParents[key] != key {
		problem.readParents[key] = problem.readParents[problem.readParents[key]]
		key = problem.readParents[key]
	}
	return key
}

func (problem *groundArrayCongruence) unionRead(left, right int) {
	left, right = problem.readRoot(left), problem.readRoot(right)
	if left != right {
		problem.readParents[right] = left
	}
}
