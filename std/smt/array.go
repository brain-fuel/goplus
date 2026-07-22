package smt

type integerArrayDefault struct {
	id    int
	value IntegerValue
}

type integerArrayEntry struct {
	id    int
	index IntegerValue
	value IntegerValue
}

// integerArrayModel is a compact finite interpretation of integer arrays.
// Unmentioned indices use the array's default value.
type integerArrayModel struct {
	defaultCount int
	defaults     [8]integerArrayDefault
	entryCount   int
	entries      [16]integerArrayEntry
}

func (model *integerArrayModel) setDefault(id int, value IntegerValue) {
	for index := 0; index < model.defaultCount; index++ {
		if model.defaults[index].id == id {
			model.defaults[index].value = value
			return
		}
	}
	if model.defaultCount < len(model.defaults) {
		model.defaults[model.defaultCount] = integerArrayDefault{id: id, value: value}
		model.defaultCount++
	}
}

func (model *integerArrayModel) set(id int, index, value IntegerValue) {
	for position := 0; position < model.entryCount; position++ {
		entry := &model.entries[position]
		if entry.id == id && CompareIntegerValue(entry.index, index) == 0 {
			entry.value = value
			return
		}
	}
	if model.entryCount < len(model.entries) {
		model.entries[model.entryCount] = integerArrayEntry{id: id, index: index, value: value}
		model.entryCount++
	}
}

func (model integerArrayModel) lookup(id int, index IntegerValue) (IntegerValue, bool) {
	for position := model.entryCount - 1; position >= 0; position-- {
		entry := model.entries[position]
		if entry.id == id && CompareIntegerValue(entry.index, index) == 0 {
			return entry.value, true
		}
	}
	for position := 0; position < model.defaultCount; position++ {
		if model.defaults[position].id == id {
			return model.defaults[position].value, true
		}
	}
	return IntegerValue{}, false
}

func evaluateIntegerArray(array Term[ArraySort[IntSort, IntSort]], index IntegerValue, integers integerModel, arrays *integerArrayModel) (IntegerValue, bool) {
	if symbol, ok := any(array).(arraySymbolTerm); ok {
		id, _ := symbol.arraySymbolParts()
		if arrays == nil {
			return IntegerValue{}, false
		}
		return arrays.lookup(id, index)
	}
	if constant, ok := any(array).(arrayConstantTerm); ok {
		term, ok := constant.arrayDefaultValue().(Term[IntSort])
		if !ok {
			return IntegerValue{}, false
		}
		return evaluateInteger(term, booleanModel{}, integers, rationalModel{})
	}
	if store, ok := any(array).(arrayStoreTerm); ok {
		base, storedIndexTerm, storedValueTerm := store.arrayStoreParts()
		storedIndex, indexOK := storedIndexTerm.(Term[IntSort])
		storedValue, valueOK := storedValueTerm.(Term[IntSort])
		baseArray, baseOK := base.(Term[ArraySort[IntSort, IntSort]])
		if !indexOK || !valueOK || !baseOK {
			return IntegerValue{}, false
		}
		resolvedIndex, ok := evaluateInteger(storedIndex, booleanModel{}, integers, rationalModel{})
		if !ok {
			return IntegerValue{}, false
		}
		if CompareIntegerValue(resolvedIndex, index) == 0 {
			return evaluateInteger(storedValue, booleanModel{}, integers, rationalModel{})
		}
		return evaluateIntegerArray(baseArray, index, integers, arrays)
	}
	return IntegerValue{}, false
}

func evaluateIntegerModelTerm(term Term[IntSort], booleans booleanModel, integers integerModel, reals rationalModel, bitVectors bitVectorModel, arrays *integerArrayModel) (IntegerValue, bool) {
	if value, ok := evaluateIntegerWithBitVectors(term, booleans, integers, reals, bitVectors); ok {
		return value, true
	}
	selection, ok := any(term).(arraySelectionTerm)
	if !ok {
		return IntegerValue{}, false
	}
	arrayTerm, indexTerm := selection.arraySelectionParts()
	array, arrayOK := arrayTerm.(Term[ArraySort[IntSort, IntSort]])
	index, indexOK := indexTerm.(Term[IntSort])
	if !arrayOK || !indexOK {
		return IntegerValue{}, false
	}
	resolvedIndex, ok := evaluateIntegerWithBitVectors(index, booleans, integers, reals, bitVectors)
	if !ok {
		return IntegerValue{}, false
	}
	return evaluateIntegerArray(array, resolvedIndex, integers, arrays)
}

type ArrayEqualityRelation struct {
	LeftID  int
	RightID int
	Negated bool
}

func (ArrayEqualityRelation) isTerm(BoolSort) {}

type ArrayReadRelation struct {
	LeftID     int
	RightID    int
	LeftIndex  IntegerValue
	RightIndex IntegerValue
	Negated    bool
}

func (ArrayReadRelation) isTerm(BoolSort) {}

type ArrayCongruenceConjunction struct {
	Equality ArrayEqualityRelation
	Read     ArrayReadRelation
}

func (ArrayCongruenceConjunction) isTerm(BoolSort) {}

type ArrayStoreEqualityRelation struct {
	LeftID     int
	RightID    int
	LeftIndex  IntegerValue
	RightIndex IntegerValue
	LeftValue  IntegerValue
	RightValue IntegerValue
	Negated    bool
}

func (ArrayStoreEqualityRelation) isTerm(BoolSort) {}

type ArrayStoreBridgeReadConjunction struct {
	Store ArrayStoreEqualityRelation
	Read  ArrayReadRelation
}

func (ArrayStoreBridgeReadConjunction) isTerm(BoolSort) {}

type ArrayConstantEqualityRelation struct {
	ArrayID int
	Default IntegerValue
	Negated bool
}

func (ArrayConstantEqualityRelation) isTerm(BoolSort) {}

type ArrayReadValueRelation struct {
	ArrayID int
	Index   IntegerValue
	Value   IntegerValue
	Negated bool
}

func (ArrayReadValueRelation) isTerm(BoolSort) {}

type ArrayConstantReadConjunction struct {
	Equality ArrayConstantEqualityRelation
	Read     ArrayReadValueRelation
}

func (ArrayConstantReadConjunction) isTerm(BoolSort) {}

type ArrayStoreReadValueRelation struct {
	ArrayID       int
	StoreIndexID  int
	ReadIndexID   int
	StoredValue   IntegerValue
	ComparedValue IntegerValue
	Negated       bool
}

func (ArrayStoreReadValueRelation) isTerm(BoolSort) {}

type ArrayIntegerEqualityExchange struct {
	First  IntegerDifferenceConstraint
	Second IntegerDifferenceConstraint
	Read   ArrayStoreReadValueRelation
}

func (ArrayIntegerEqualityExchange) isTerm(BoolSort) {}

func CompactIntegerArrayStoreReadValueEquality(left, right Term[IntSort]) (ArrayStoreReadValueRelation, bool) {
	if read, ok := left.(arrayStoreReadInteger[IntSort]); ok {
		if value, valueOK := exactIntegerConstant(right); valueOK {
			return ArrayStoreReadValueRelation{ArrayID: read.arrayID, StoreIndexID: read.storeIndexID, ReadIndexID: read.readIndexID, StoredValue: read.value, ComparedValue: value}, true
		}
	}
	if read, ok := right.(arrayStoreReadInteger[IntSort]); ok {
		if value, valueOK := exactIntegerConstant(left); valueOK {
			return ArrayStoreReadValueRelation{ArrayID: read.arrayID, StoreIndexID: read.storeIndexID, ReadIndexID: read.readIndexID, StoredValue: read.value, ComparedValue: value}, true
		}
	}
	return ArrayStoreReadValueRelation{}, false
}

func solveCompactArrayIntegerExchange(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if len(assertions) != 1 {
		return checkOutcome{}, false
	}
	value, ok := assertions[0].(ArrayIntegerEqualityExchange)
	if !ok {
		return checkOutcome{}, false
	}
	first, second, read := value.First, value.Second, value.Read
	firstZero := !first.Strict && !first.Wide && first.Bound == 0 && first.HasPositive && first.HasNegative
	secondZero := !second.Strict && !second.Wide && second.Bound == 0 && second.HasPositive && second.HasNegative
	reciprocal := first.PositiveID == second.NegativeID && first.NegativeID == second.PositiveID
	indices := read.StoreIndexID == first.PositiveID && read.ReadIndexID == first.NegativeID || read.StoreIndexID == first.NegativeID && read.ReadIndexID == first.PositiveID
	if firstZero && secondZero && reciprocal && indices && read.Negated && CompareIntegerValue(read.StoredValue, read.ComparedValue) == 0 {
		return checkOutcome{status: checkUnsat}, true
	}
	return checkOutcome{}, false
}

func CompactIntegerArrayReadValueEquality(left, right Term[IntSort]) (ArrayReadValueRelation, bool) {
	if read, ok := left.(arrayReadInteger[IntSort]); ok {
		if value, valueOK := exactIntegerConstant(right); valueOK {
			return ArrayReadValueRelation{ArrayID: read.arrayID, Index: read.index, Value: value}, true
		}
	}
	if read, ok := right.(arrayReadInteger[IntSort]); ok {
		if value, valueOK := exactIntegerConstant(left); valueOK {
			return ArrayReadValueRelation{ArrayID: read.arrayID, Index: read.index, Value: value}, true
		}
	}
	return ArrayReadValueRelation{}, false
}

func CompactIntegerArrayReadEquality(left, right Term[IntSort]) (ArrayReadRelation, bool) {
	first, firstOK := left.(arrayReadInteger[IntSort])
	second, secondOK := right.(arrayReadInteger[IntSort])
	if !firstOK || !secondOK {
		return ArrayReadRelation{}, false
	}
	return ArrayReadRelation{LeftID: first.arrayID, RightID: second.arrayID, LeftIndex: first.index, RightIndex: second.index}, true
}

type arrayTerm interface{ arrayTermKind() uint8 }
type arraySymbolTerm interface{ arraySymbolParts() (int, string) }
type arrayConstantTerm interface{ arrayDefaultValue() any }
type arrayStoreTerm interface{ arrayStoreParts() (any, any, any) }
type arraySelectionTerm interface{ arraySelectionParts() (any, any) }

func (arraySymbol[S]) arrayTermKind() uint8                  { return 1 }
func (constantArray[S]) arrayTermKind() uint8                { return 2 }
func (arrayStore[S]) arrayTermKind() uint8                   { return 3 }
func (value arraySymbol[S]) arraySymbolParts() (int, string) { return value.iD, value.name }
func (value constantArray[S]) arrayDefaultValue() any        { return value.defaultValue }
func (value arrayStore[S]) arrayStoreParts() (any, any, any) {
	return value.array, value.index, value.value
}
func (value arraySelect[S]) arraySelectionParts() (any, any) { return value.array, value.index }

// containsArrayTheory identifies formulas whose equality operands contain an
// array or a select from one. The initial decision procedure intentionally
// accepts only ground, decidable read-over-write formulas.
func containsArrayTheory(term Term[BoolSort]) bool {
	switch value := term.(type) {
	case And:
		for _, item := range value.Values {
			if containsArrayTheory(item) {
				return true
			}
		}
	case BooleanConjunction:
		items, _ := value.values()
		for _, item := range items {
			if containsArrayTheory(item) {
				return true
			}
		}
	case Not:
		return containsArrayTheory(value.Value)
	case Or:
		for _, item := range value.Values {
			if containsArrayTheory(item) {
				return true
			}
		}
	case Implies:
		return containsArrayTheory(value.Left) || containsArrayTheory(value.Right)
	case Iff:
		return containsArrayTheory(value.Left) || containsArrayTheory(value.Right)
	case Equal:
		return isArrayTerm(value.Left) || isArrayTerm(value.Right) || isArraySelect(value.Left) || isArraySelect(value.Right)
	case ArrayEqualityRelation, ArrayReadRelation, ArrayCongruenceConjunction, ArrayStoreEqualityRelation, ArrayStoreBridgeReadConjunction, ArrayConstantEqualityRelation, ArrayReadValueRelation, ArrayConstantReadConjunction, ArrayStoreReadValueRelation, ArrayIntegerEqualityExchange, BitVectorArrayEqualityRelation, BitVectorArrayStoreReadValueRelation, BitVectorArrayEqualityExchange:
		return true
	}
	return false
}

func solveCompactArrayAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if len(assertions) == 1 {
		if conjunction, ok := assertions[0].(ArrayConstantReadConjunction); ok {
			equality, read := conjunction.Equality, conjunction.Read
			if !equality.Negated && read.Negated && equality.ArrayID == read.ArrayID && CompareIntegerValue(equality.Default, read.Value) == 0 {
				return checkOutcome{status: checkUnsat}, true
			}
		}
		if conjunction, ok := assertions[0].(ArrayStoreBridgeReadConjunction); ok {
			store, read := conjunction.Store, conjunction.Read
			if !store.Negated && read.Negated && CompareIntegerValue(read.LeftIndex, read.RightIndex) == 0 {
				direct := read.LeftID == store.LeftID && read.RightID == store.RightID || read.LeftID == store.RightID && read.RightID == store.LeftID
				outside := CompareIntegerValue(read.LeftIndex, store.LeftIndex) != 0 && CompareIntegerValue(read.LeftIndex, store.RightIndex) != 0
				if direct && outside {
					return checkOutcome{status: checkUnsat}, true
				}
			}
		}
	}
	var equalities [4]ArrayEqualityRelation
	var reads [8]ArrayReadRelation
	equalityCount, readCount := 0, 0
	var add func(Term[BoolSort], bool) bool
	add = func(term Term[BoolSort], negated bool) bool {
		switch value := term.(type) {
		case And:
			if negated {
				return false
			}
			for _, item := range value.Values {
				if !add(item, false) {
					return false
				}
			}
			return true
		case BooleanConjunction:
			if negated {
				return false
			}
			items, signs := value.values()
			for index, item := range items {
				if !add(item, signs[index]) {
					return false
				}
			}
			return true
		case Not:
			return add(value.Value, !negated)
		case ArrayEqualityRelation:
			if equalityCount == len(equalities) {
				return false
			}
			value.Negated = value.Negated != negated
			equalities[equalityCount] = value
			equalityCount++
			return true
		case ArrayReadRelation:
			if readCount == len(reads) {
				return false
			}
			value.Negated = value.Negated != negated
			reads[readCount] = value
			readCount++
			return true
		case ArrayCongruenceConjunction:
			if negated || equalityCount == len(equalities) || readCount == len(reads) {
				return false
			}
			equalities[equalityCount] = value.Equality
			equalityCount++
			reads[readCount] = value.Read
			readCount++
			return true
		}
		return false
	}
	for _, assertion := range assertions {
		if !add(assertion, false) {
			return checkOutcome{}, false
		}
	}
	var parent [8]int
	var ids [8]int
	count := 0
	findIndex := func(id int) int {
		for index := 0; index < count; index++ {
			if ids[index] == id {
				return index
			}
		}
		if count == len(ids) {
			return -1
		}
		ids[count], parent[count] = id, count
		count++
		return count - 1
	}
	var root func(int) int
	root = func(index int) int {
		if parent[index] != index {
			parent[index] = root(parent[index])
		}
		return parent[index]
	}
	for _, equality := range equalities[:equalityCount] {
		left, right := findIndex(equality.LeftID), findIndex(equality.RightID)
		if left < 0 || right < 0 {
			return checkOutcome{}, false
		}
		if equality.Negated {
			continue
		}
		left, right = root(left), root(right)
		if left != right {
			parent[right] = left
		}
	}
	for _, equality := range equalities[:equalityCount] {
		if equality.Negated {
			left, right := findIndex(equality.LeftID), findIndex(equality.RightID)
			if root(left) == root(right) {
				return checkOutcome{status: checkUnsat}, true
			}
		}
	}
	for _, read := range reads[:readCount] {
		left, right := findIndex(read.LeftID), findIndex(read.RightID)
		if left < 0 || right < 0 {
			return checkOutcome{}, false
		}
		equal := root(left) == root(right) && CompareIntegerValue(read.LeftIndex, read.RightIndex) == 0
		if equal == read.Negated {
			return checkOutcome{status: checkUnsat}, true
		}
		if !read.Negated && !equal {
			return checkOutcome{}, false
		}
	}
	return checkOutcome{status: checkSat}, true
}

func solveArrayAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	if outcome, recognized := solveCompactArrayAssertions(assertions); recognized {
		return outcome, true
	}
	if outcome, recognized := solveGroundArrayAssertions(assertions); recognized {
		return outcome, true
	}
	if outcome, recognized := solveGroundBitVectorArrays(assertions); recognized {
		return outcome, true
	}
	return solveGroundArrayCongruence(assertions)
}

func isArrayTerm(term any) bool {
	_, ok := term.(arrayTerm)
	return ok
}

func isArraySelect(term any) bool {
	if _, ok := term.(arrayStoreReadInteger[IntSort]); ok {
		return true
	}
	_, ok := term.(arraySelectionTerm)
	return ok
}

func solveGroundArrayAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	for _, assertion := range assertions {
		holds, known := evaluateGroundArrayBoolean(assertion, false)
		if !known {
			return checkOutcome{}, false
		}
		if !holds {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	return checkOutcome{status: checkSat}, true
}

func evaluateGroundArrayBoolean(term Term[BoolSort], negated bool) (bool, bool) {
	switch value := term.(type) {
	case Bool:
		return value.Value != negated, true
	case Not:
		return evaluateGroundArrayBoolean(value.Value, !negated)
	case And:
		if negated {
			return false, false
		}
		for _, item := range value.Values {
			holds, known := evaluateGroundArrayBoolean(item, false)
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
			holds, known := evaluateGroundArrayBoolean(item, itemNegated[index])
			if !known || !holds {
				return holds, known
			}
		}
		return true, true
	case Equal:
		if isArrayTerm(value.Left) || isArrayTerm(value.Right) {
			return false, false
		}
		left, leftOK := normalizeArraySelect(value.Left)
		right, rightOK := normalizeArraySelect(value.Right)
		if !leftOK || !rightOK {
			return false, false
		}
		equal, known := groundTermEqual(left, right)
		return equal != negated, known
	}
	return false, false
}

func normalizeArraySelect(term any) (any, bool) {
	if selection, ok := term.(arraySelectionTerm); ok {
		array, index := selection.arraySelectionParts()
		return resolveArraySelect(array, index)
	}
	return term, true
}

func resolveArraySelect(array, index any) (any, bool) {
	if value, ok := array.(arrayConstantTerm); ok {
		return value.arrayDefaultValue(), true
	}
	if value, ok := array.(arrayStoreTerm); ok {
		base, storedIndex, storedValue := value.arrayStoreParts()
		equal, known := groundTermEqual(storedIndex, index)
		if !known {
			return nil, false
		}
		if equal {
			return storedValue, true
		}
		return resolveArraySelect(base, index)
	}
	return nil, false
}

func groundTermEqual(left, right any) (bool, bool) {
	if first, firstOK := exactIntegerConstant(left); firstOK {
		second, secondOK := exactIntegerConstant(right)
		if !secondOK {
			return false, false
		}
		return CompareIntegerValue(first, second) == 0, true
	}
	if first, ok := left.(arraySymbolTerm); ok {
		second, secondOK := right.(arraySymbolTerm)
		if !secondOK {
			return false, false
		}
		firstID, _ := first.arraySymbolParts()
		secondID, _ := second.arraySymbolParts()
		if firstID == secondID {
			return true, true
		}
		return false, false
	}
	if first, ok := left.(arrayConstantTerm); ok {
		second, secondOK := right.(arrayConstantTerm)
		if !secondOK {
			return false, false
		}
		return groundTermEqual(first.arrayDefaultValue(), second.arrayDefaultValue())
	}
	if first, ok := left.(arrayStoreTerm); ok {
		second, secondOK := right.(arrayStoreTerm)
		if !secondOK {
			return false, false
		}
		firstBase, firstIndex, firstValue := first.arrayStoreParts()
		secondBase, secondIndex, secondValue := second.arrayStoreParts()
		baseEqual, baseKnown := groundTermEqual(firstBase, secondBase)
		indexEqual, indexKnown := groundTermEqual(firstIndex, secondIndex)
		valueEqual, valueKnown := groundTermEqual(firstValue, secondValue)
		if !baseKnown || !indexKnown || !valueKnown {
			return false, false
		}
		return baseEqual && indexEqual && valueEqual, true
	}
	switch first := left.(type) {
	case Bool:
		second, ok := right.(Bool)
		return ok && first.Value == second.Value, ok
	case Integer:
		second, ok := right.(Integer)
		return ok && first.Value == second.Value, ok
	case integerExact[IntSort]:
		second, ok := right.(integerExact[IntSort])
		return ok && CompareIntegerValue(first.value, second.value) == 0, ok
	case Real:
		second, ok := right.(Real)
		return ok && CompareRational(first.Value, second.Value) == 0, ok
	case bitVector[BitVecSort]:
		second, ok := right.(bitVector[BitVecSort])
		return ok && EqualBitVectorValue(first.value, second.value), ok
	case IntSymbol:
		second, ok := right.(IntSymbol)
		if !ok {
			return false, false
		}
		return first.ID == second.ID, first.ID == second.ID
	case integerVariable[IntSort]:
		second, ok := right.(integerVariable[IntSort])
		if !ok {
			return false, false
		}
		return first.iD == second.iD, first.iD == second.iD
	case RealSymbol:
		second, ok := right.(RealSymbol)
		if !ok {
			return false, false
		}
		return first.ID == second.ID, first.ID == second.ID
	case BoolSymbol:
		second, ok := right.(BoolSymbol)
		if !ok {
			return false, false
		}
		return first.ID == second.ID, first.ID == second.ID
	}
	return false, false
}
