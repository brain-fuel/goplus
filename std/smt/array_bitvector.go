package smt

// BitVectorArrayStoreReadValueRelation is the compact official-API form of a
// symbolic-address read from a one-store array compared with an exact value.
type BitVectorArrayStoreReadValueRelation struct {
	ArrayID, StoreIndexID, ReadIndexID int
	IndexWidth, ElementWidth           int
	StoredValue, ComparedValue         BitVectorValue
	Negated                            bool
}

func (BitVectorArrayStoreReadValueRelation) isTerm(BoolSort) {}

// BitVectorArrayEqualityExchange fuses an address equality with the array
// read that consumes it, avoiding general term allocation on the hot path.
type BitVectorArrayEqualityExchange struct {
	Equality BitVectorEUFRelation
	Read     BitVectorArrayStoreReadValueRelation
}

func (BitVectorArrayEqualityExchange) isTerm(BoolSort) {}

func solveCompactBitVectorArrayExchange(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if len(assertions) != 1 {
		return checkOutcome{}, false
	}
	value, ok := assertions[0].(BitVectorArrayEqualityExchange)
	if !ok {
		return checkOutcome{}, false
	}
	equality, read := value.Equality, value.Read
	if equality.Negated || equality.Left.Kind != 1 || equality.Right.Kind != 1 || equality.Left.Width != read.IndexWidth || equality.Right.Width != read.IndexWidth {
		return checkOutcome{}, false
	}
	reciprocal := equality.Left.SymbolID == read.StoreIndexID && equality.Right.SymbolID == read.ReadIndexID || equality.Right.SymbolID == read.StoreIndexID && equality.Left.SymbolID == read.ReadIndexID
	if !reciprocal {
		return checkOutcome{}, false
	}
	equal := EqualBitVectorValue(read.StoredValue, read.ComparedValue)
	if equal == read.Negated {
		return checkOutcome{status: checkUnsat}, true
	}
	return checkOutcome{status: checkSat}, true
}

// groundBitVectorArrayProblem is the allocation-free exact-index QF_AUFBV
// congruence layer. It complements the integer-array engine without erasing
// bit-vector widths or routing array reads through the general bit blaster.
type groundBitVectorArrayProblem struct {
	arrayCount    int
	arrayIDs      [16]int
	arrayParents  [16]int
	indexCount    int
	indexIDs      [16]int
	indexWidths   [16]int
	indexParents  [16]int
	readCount     int
	reads         [32]groundBitVectorArrayRead
	readParents   [32]int
	diseqCount    int
	diseqs        [16][2]int
	bitVectorSeen bool
}

type groundBitVectorArrayRead struct {
	constant bool
	arrayID  int
	index    groundBitVectorArrayIndex
	value    BitVectorValue
}

type groundBitVectorArrayIndex struct {
	symbol bool
	id     int
	width  int
	value  BitVectorValue
}

func solveGroundBitVectorArrays(assertions []Term[BoolSort]) (checkOutcome, bool) {
	problem := groundBitVectorArrayProblem{}
	for _, assertion := range assertions {
		if !problem.collectArrays(assertion, false) {
			return checkOutcome{}, false
		}
	}
	for _, assertion := range assertions {
		if !problem.collectElements(assertion, false) {
			return checkOutcome{}, false
		}
	}
	if !problem.bitVectorSeen {
		return checkOutcome{}, false
	}
	for _, pair := range problem.diseqs[:problem.diseqCount] {
		if problem.readRoot(pair[0]) == problem.readRoot(pair[1]) {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	for left := 0; left < problem.readCount; left++ {
		if !problem.reads[left].constant {
			continue
		}
		for right := left + 1; right < problem.readCount; right++ {
			if problem.reads[right].constant && problem.readRoot(left) == problem.readRoot(right) && !EqualBitVectorValue(problem.reads[left].value, problem.reads[right].value) {
				return checkOutcome{status: checkUnsat}, true
			}
		}
	}
	return checkOutcome{status: checkSat}, true
}

// solveSharedArrayBitVector exchanges equalities entailed by QF_BV into the
// array congruence closure. This is the Nelson-Oppen seam needed for symbolic
// bit-vector addresses: bit blasting decides address equality, then array
// read-over-write consumes only the resulting equality facts.
func solveSharedArrayBitVector(assertions []Term[BoolSort]) (checkOutcome, bool) {
	arrays := make([]Term[BoolSort], 0, len(assertions))
	bitVectors := make([]Term[BoolSort], 0, len(assertions))
	var indexIDs [16]int
	var indexWidths [16]int
	indexCount := 0
	addIndex := func(id, width int) {
		for index := 0; index < indexCount; index++ {
			if indexIDs[index] == id && indexWidths[index] == width {
				return
			}
		}
		if indexCount < len(indexIDs) {
			indexIDs[indexCount], indexWidths[indexCount] = id, width
			indexCount++
		}
	}
	for _, assertion := range assertions {
		if !splitSharedArrayBitVector(assertion, false, &arrays, &bitVectors, addIndex) {
			return checkOutcome{}, false
		}
	}
	if len(arrays) == 0 || indexCount == 0 {
		return checkOutcome{}, false
	}
	bitVectorOutcome := checkOutcome{status: checkSat}
	if len(bitVectors) != 0 {
		solverBitVectors := make([]Term[BoolSort], len(bitVectors))
		for index, term := range bitVectors {
			solverBitVectors[index] = expandSharedBitVectorTerm(term)
		}
		var recognized bool
		bitVectorOutcome, recognized = solveBitVectorAssertions(solverBitVectors)
		if !recognized {
			return checkOutcome{}, false
		}
		if bitVectorOutcome.status == checkUnsat {
			return bitVectorOutcome, true
		}
		for _, term := range bitVectors {
			appendDirectBitVectorIndexEqualities(term, &arrays)
		}
		bitVectors = solverBitVectors
	}
	for left := 0; left < indexCount; left++ {
		for right := left + 1; right < indexCount; right++ {
			if indexWidths[left] != indexWidths[right] {
				continue
			}
			first := bitVectorSymbol[BitVecSort]{width: indexWidths[left], iD: indexIDs[left]}
			second := bitVectorSymbol[BitVecSort]{width: indexWidths[right], iD: indexIDs[right]}
			probe := append(make([]Term[BoolSort], 0, len(bitVectors)+1), bitVectors...)
			probe = append(probe, Not{Value: Equal{Left: first, Right: second}})
			outcome, recognized := solveBitVectorAssertions(probe)
			if !recognized {
				continue
			}
			if outcome.status == checkUnsat {
				arrays = append(arrays, Equal{Left: first, Right: second})
			}
		}
	}
	arrayOutcome, recognized := solveGroundBitVectorArrays(arrays)
	if !recognized {
		return checkOutcome{}, false
	}
	if arrayOutcome.status == checkSat {
		arrayOutcome.bitVectors = bitVectorOutcome.bitVectors
	}
	return arrayOutcome, true
}

func expandSharedBitVectorTerm(term Term[BoolSort]) Term[BoolSort] {
	switch value := term.(type) {
	case BitVectorEUFRelation:
		if value.Left.Kind == 1 && value.Right.Kind == 1 && value.Left.Width == value.Right.Width {
			var equality Term[BoolSort] = Equal{
				Left:  bitVectorSymbol[BitVecSort]{width: value.Left.Width, iD: value.Left.SymbolID},
				Right: bitVectorSymbol[BitVecSort]{width: value.Right.Width, iD: value.Right.SymbolID},
			}
			if value.Negated {
				equality = Not{Value: equality}
			}
			return equality
		}
	case BitVectorEUFConjunction:
		relations := value.values()
		terms := make([]Term[BoolSort], len(relations))
		for index, relation := range relations {
			terms[index] = expandSharedBitVectorTerm(relation)
		}
		return And{Values: terms}
	}
	return term
}

func appendDirectBitVectorIndexEqualities(term Term[BoolSort], arrays *[]Term[BoolSort]) {
	switch value := term.(type) {
	case BitVectorEUFRelation:
		if !value.Negated && value.Left.Kind == 1 && value.Right.Kind == 1 && value.Left.Width == value.Right.Width {
			*arrays = append(*arrays, Equal{
				Left:  bitVectorSymbol[BitVecSort]{width: value.Left.Width, iD: value.Left.SymbolID},
				Right: bitVectorSymbol[BitVecSort]{width: value.Right.Width, iD: value.Right.SymbolID},
			})
		}
	case BitVectorEUFConjunction:
		var relations []BitVectorEUFRelation
		if value.Overflow != nil {
			relations = value.Overflow
		} else {
			relations = value.Inline[:value.Count]
		}
		for _, relation := range relations {
			appendDirectBitVectorIndexEqualities(relation, arrays)
		}
	}
}

func splitSharedArrayBitVector(term Term[BoolSort], negated bool, arrays, bitVectors *[]Term[BoolSort], addIndex func(int, int)) bool {
	switch value := term.(type) {
	case And:
		if negated {
			return false
		}
		for _, item := range value.Values {
			if !splitSharedArrayBitVector(item, false, arrays, bitVectors, addIndex) {
				return false
			}
		}
		return true
	case BooleanConjunction:
		if negated {
			return false
		}
		items, polarities := value.values()
		for index, item := range items {
			if !splitSharedArrayBitVector(item, polarities[index], arrays, bitVectors, addIndex) {
				return false
			}
		}
		return true
	case Not:
		return splitSharedArrayBitVector(value.Value, !negated, arrays, bitVectors, addIndex)
	}
	effective := term
	if negated {
		effective = Not{Value: term}
	}
	if containsArrayTheory(effective) {
		*arrays = append(*arrays, effective)
		collectBitVectorArrayIndexSymbols(effective, addIndex)
		return true
	}
	if containsBitVectorTheory(effective) {
		*bitVectors = append(*bitVectors, effective)
		return true
	}
	return false
}

func collectBitVectorArrayIndexSymbols(term any, add func(int, int)) {
	switch value := term.(type) {
	case And:
		for _, item := range value.Values {
			collectBitVectorArrayIndexSymbols(item, add)
		}
	case BooleanConjunction:
		items, _ := value.values()
		for _, item := range items {
			collectBitVectorArrayIndexSymbols(item, add)
		}
	case Not:
		collectBitVectorArrayIndexSymbols(value.Value, add)
	case Equal:
		collectBitVectorArrayIndexSymbols(value.Left, add)
		collectBitVectorArrayIndexSymbols(value.Right, add)
	case arraySelectionTerm:
		array, index := value.arraySelectionParts()
		if symbol, ok := index.(bitVectorSymbol[BitVecSort]); ok {
			add(symbol.iD, symbol.width)
		}
		collectBitVectorArrayIndexSymbols(array, add)
	case arrayStoreTerm:
		array, index, stored := value.arrayStoreParts()
		if symbol, ok := index.(bitVectorSymbol[BitVecSort]); ok {
			add(symbol.iD, symbol.width)
		}
		collectBitVectorArrayIndexSymbols(array, add)
		collectBitVectorArrayIndexSymbols(stored, add)
	}
}

func containsSymbolicBitVectorArrayIndex(term any) bool {
	found := false
	collectBitVectorArrayIndexSymbols(term, func(int, int) { found = true })
	return found
}

func containsSymbolicBitVectorArrayIndices(assertions []Term[BoolSort]) bool {
	for _, assertion := range assertions {
		if containsSymbolicBitVectorArrayIndex(assertion) {
			return true
		}
	}
	return false
}

func (problem *groundBitVectorArrayProblem) collectArrays(term Term[BoolSort], negated bool) bool {
	switch value := term.(type) {
	case Bool:
		return true
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
		items, polarities := value.values()
		for index, item := range items {
			if !problem.collectArrays(item, polarities[index]) {
				return false
			}
		}
		return true
	case Not:
		return problem.collectArrays(value.Value, !negated)
	case Equal:
		leftArray, leftOK := arraySymbolID(value.Left)
		rightArray, rightOK := arraySymbolID(value.Right)
		if leftOK || rightOK {
			if !leftOK || !rightOK || negated {
				return false
			}
			problem.ensureArray(leftArray)
			problem.ensureArray(rightArray)
			problem.unionArray(leftArray, rightArray)
			return true
		}
		leftIndex, leftIndexOK := problem.index(value.Left)
		rightIndex, rightIndexOK := problem.index(value.Right)
		if leftIndexOK && rightIndexOK && !negated {
			problem.unionIndex(leftIndex, rightIndex)
		}
		return true
	}
	return false
}

func (problem *groundBitVectorArrayProblem) collectElements(term Term[BoolSort], negated bool) bool {
	switch value := term.(type) {
	case Bool:
		return value.Value != negated
	case And:
		if negated {
			return false
		}
		for _, item := range value.Values {
			if !problem.collectElements(item, false) {
				return false
			}
		}
		return true
	case BooleanConjunction:
		if negated {
			return false
		}
		items, polarities := value.values()
		for index, item := range items {
			if !problem.collectElements(item, polarities[index]) {
				return false
			}
		}
		return true
	case Not:
		return problem.collectElements(value.Value, !negated)
	case Equal:
		if isArrayTerm(value.Left) || isArrayTerm(value.Right) {
			return !negated
		}
		if _, leftIndex := problem.index(value.Left); leftIndex {
			if _, rightIndex := problem.index(value.Right); rightIndex {
				// Pure address equalities are discharged by the bit-vector
				// partition and consumed during the first (array) pass.
				return true
			}
		}
		left, leftOK := problem.elementKey(value.Left)
		right, rightOK := problem.elementKey(value.Right)
		if !leftOK || !rightOK {
			return false
		}
		if negated {
			if problem.diseqCount == len(problem.diseqs) {
				return false
			}
			problem.diseqs[problem.diseqCount] = [2]int{left, right}
			problem.diseqCount++
		} else {
			problem.unionRead(left, right)
		}
		return true
	}
	return false
}

func (problem *groundBitVectorArrayProblem) elementKey(term any) (int, bool) {
	if value, ok := exactBitVectorArrayValue(term); ok {
		problem.bitVectorSeen = true
		return problem.ensureRead(groundBitVectorArrayRead{constant: true, value: value})
	}
	selection, ok := term.(arraySelectionTerm)
	if !ok {
		return 0, false
	}
	array, indexTerm := selection.arraySelectionParts()
	index, ok := problem.index(indexTerm)
	if !ok {
		return 0, false
	}
	problem.bitVectorSeen = true
	for {
		if stored, ok := array.(arrayStoreTerm); ok {
			base, storedIndexTerm, storedValue := stored.arrayStoreParts()
			storedIndex, indexOK := problem.index(storedIndexTerm)
			if !indexOK {
				return 0, false
			}
			if problem.indexEqual(index, storedIndex) {
				return problem.elementKey(storedValue)
			}
			array = base
			continue
		}
		if constant, ok := array.(arrayConstantTerm); ok {
			return problem.elementKey(constant.arrayDefaultValue())
		}
		id, ok := arraySymbolID(array)
		if !ok {
			return 0, false
		}
		problem.ensureArray(id)
		return problem.ensureRead(groundBitVectorArrayRead{arrayID: problem.arrayRoot(id), index: index})
	}
}

func exactBitVectorArrayValue(term any) (BitVectorValue, bool) {
	value, ok := term.(bitVector[BitVecSort])
	if !ok {
		return BitVectorValue{}, false
	}
	return value.value, true
}

func (problem *groundBitVectorArrayProblem) index(term any) (groundBitVectorArrayIndex, bool) {
	if value, ok := exactBitVectorArrayValue(term); ok {
		problem.bitVectorSeen = true
		return groundBitVectorArrayIndex{width: value.Width(), value: value}, true
	}
	if value, ok := term.(bitVectorSymbol[BitVecSort]); ok {
		problem.bitVectorSeen = true
		return problem.ensureIndex(value.iD, value.width)
	}
	return groundBitVectorArrayIndex{}, false
}

func (problem *groundBitVectorArrayProblem) ensureIndex(id, width int) (groundBitVectorArrayIndex, bool) {
	for index := 0; index < problem.indexCount; index++ {
		if problem.indexIDs[index] == id && problem.indexWidths[index] == width {
			return groundBitVectorArrayIndex{symbol: true, id: id, width: width}, true
		}
	}
	if problem.indexCount == len(problem.indexIDs) {
		return groundBitVectorArrayIndex{}, false
	}
	problem.indexIDs[problem.indexCount] = id
	problem.indexWidths[problem.indexCount] = width
	problem.indexParents[problem.indexCount] = problem.indexCount
	problem.indexCount++
	return groundBitVectorArrayIndex{symbol: true, id: id, width: width}, true
}

func (problem *groundBitVectorArrayProblem) indexPosition(value groundBitVectorArrayIndex) int {
	for index := 0; index < problem.indexCount; index++ {
		if problem.indexIDs[index] == value.id && problem.indexWidths[index] == value.width {
			return index
		}
	}
	return -1
}

func (problem *groundBitVectorArrayProblem) indexRoot(position int) int {
	for problem.indexParents[position] != position {
		problem.indexParents[position] = problem.indexParents[problem.indexParents[position]]
		position = problem.indexParents[position]
	}
	return position
}

func (problem *groundBitVectorArrayProblem) normalizeIndex(value groundBitVectorArrayIndex) groundBitVectorArrayIndex {
	if !value.symbol {
		return value
	}
	position := problem.indexPosition(value)
	if position < 0 {
		return value
	}
	root := problem.indexRoot(position)
	return groundBitVectorArrayIndex{symbol: true, id: problem.indexIDs[root], width: problem.indexWidths[root]}
}

func (problem *groundBitVectorArrayProblem) indexEqual(left, right groundBitVectorArrayIndex) bool {
	left, right = problem.normalizeIndex(left), problem.normalizeIndex(right)
	if left.symbol || right.symbol {
		return left.symbol && right.symbol && left.id == right.id && left.width == right.width
	}
	return EqualBitVectorValue(left.value, right.value)
}

func (problem *groundBitVectorArrayProblem) unionIndex(left, right groundBitVectorArrayIndex) {
	left, right = problem.normalizeIndex(left), problem.normalizeIndex(right)
	if !left.symbol || !right.symbol || left.width != right.width {
		return
	}
	leftPosition, rightPosition := problem.indexRoot(problem.indexPosition(left)), problem.indexRoot(problem.indexPosition(right))
	if leftPosition != rightPosition {
		problem.indexParents[rightPosition] = leftPosition
	}
}

func (problem *groundBitVectorArrayProblem) ensureArray(id int) {
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

func (problem *groundBitVectorArrayProblem) arrayPosition(id int) int {
	for index := 0; index < problem.arrayCount; index++ {
		if problem.arrayIDs[index] == id {
			return index
		}
	}
	return -1
}

func (problem *groundBitVectorArrayProblem) arrayRoot(id int) int {
	position := problem.arrayPosition(id)
	if position < 0 {
		return id
	}
	for problem.arrayParents[position] != position {
		problem.arrayParents[position] = problem.arrayParents[problem.arrayParents[position]]
		position = problem.arrayParents[position]
	}
	return problem.arrayIDs[position]
}

func (problem *groundBitVectorArrayProblem) unionArray(left, right int) {
	leftRoot, rightRoot := problem.arrayRoot(left), problem.arrayRoot(right)
	if leftRoot == rightRoot {
		return
	}
	leftPosition, rightPosition := problem.arrayPosition(leftRoot), problem.arrayPosition(rightRoot)
	if leftPosition >= 0 && rightPosition >= 0 {
		problem.arrayParents[rightPosition] = leftPosition
	}
}

func (problem *groundBitVectorArrayProblem) ensureRead(key groundBitVectorArrayRead) (int, bool) {
	for index := 0; index < problem.readCount; index++ {
		prior := problem.reads[index]
		if prior.constant != key.constant {
			continue
		}
		if prior.constant && EqualBitVectorValue(prior.value, key.value) || !prior.constant && problem.arrayRoot(prior.arrayID) == problem.arrayRoot(key.arrayID) && problem.indexEqual(prior.index, key.index) {
			return index, true
		}
	}
	if problem.readCount == len(problem.reads) {
		return 0, false
	}
	position := problem.readCount
	problem.reads[position] = key
	problem.readParents[position] = position
	problem.readCount++
	return position, true
}

func (problem *groundBitVectorArrayProblem) readRoot(position int) int {
	for problem.readParents[position] != position {
		problem.readParents[position] = problem.readParents[problem.readParents[position]]
		position = problem.readParents[position]
	}
	return position
}

func (problem *groundBitVectorArrayProblem) unionRead(left, right int) {
	left, right = problem.readRoot(left), problem.readRoot(right)
	if left != right {
		problem.readParents[right] = left
	}
}
