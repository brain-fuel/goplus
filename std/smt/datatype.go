package smt

import "goforge.dev/goplus/std/vec"

// NaryDatatypeSelectors is the compact erased representation of an indexed
// selector-name vector. Arity at most four requires no backing allocation.
type NaryDatatypeSelectors struct {
	Count    int
	Inline   [4]string
	Overflow []string
}

func compactNaryDatatypeSelectors(values vec.Vec[string]) NaryDatatypeSelectors {
	var result NaryDatatypeSelectors
	for {
		switch current := values.(type) {
		case vec.Nil[string]:
			return result
		case vec.Cons[string]:
			result.Append(current.Head)
			values = current.Tail
		default:
			panic("smt: invalid erased n-ary selector vector")
		}
	}
}

func (values *NaryDatatypeSelectors) Append(value string) {
	if values.Count < len(values.Inline) && values.Overflow == nil {
		values.Inline[values.Count] = value
		values.Count++
		return
	}
	if values.Overflow == nil {
		values.Overflow = append(make([]string, 0, values.Count+4), values.Inline[:]...)
	}
	values.Overflow = append(values.Overflow, value)
	values.Count++
}

func (values NaryDatatypeSelectors) Len() int { return values.Count }

func (values NaryDatatypeSelectors) At(index int) string {
	if index < 0 || index >= values.Count {
		panic("smt: n-ary selector outside arity")
	}
	if values.Overflow != nil {
		return values.Overflow[index]
	}
	return values.Inline[index]
}

// NaryDatatypeTerms is the compact erased representation of an indexed term
// vector. Arity at most four remains entirely inline.
type NaryDatatypeTerms struct {
	Count    int
	Inline   [4]Term[DatatypeSort]
	Overflow []Term[DatatypeSort]
}

func compactNaryDatatypeTerms[D any](values vec.Vec[Term[D]]) NaryDatatypeTerms {
	var result NaryDatatypeTerms
	for {
		switch current := values.(type) {
		case vec.Nil[Term[D]]:
			return result
		case vec.Cons[Term[D]]:
			term, ok := any(current.Head).(Term[DatatypeSort])
			if !ok {
				panic("smt: erased n-ary datatype term sort mismatch")
			}
			result.Append(term)
			values = current.Tail
		default:
			panic("smt: invalid erased n-ary datatype term vector")
		}
	}
}

func (values *NaryDatatypeTerms) Append(value Term[DatatypeSort]) {
	if values.Count < len(values.Inline) && values.Overflow == nil {
		values.Inline[values.Count] = value
		values.Count++
		return
	}
	if values.Overflow == nil {
		values.Overflow = append(make([]Term[DatatypeSort], 0, values.Count+4), values.Inline[:]...)
	}
	values.Overflow = append(values.Overflow, value)
	values.Count++
}

func (values NaryDatatypeTerms) Len() int { return values.Count }

func (values NaryDatatypeTerms) At(index int) Term[DatatypeSort] {
	if index < 0 || index >= values.Count {
		panic("smt: n-ary datatype term outside arity")
	}
	if values.Overflow != nil {
		return values.Overflow[index]
	}
	return values.Inline[index]
}

// DeclareNaryRecursiveDatatypeConstructorDynamic is the runtime-checked
// compatibility boundary for parsers and generated-Go façades that have
// already erased the arity index. Go+ callers should prefer the Vec-indexed
// DeclareNaryRecursiveDatatypeConstructor API.
func DeclareNaryRecursiveDatatypeConstructorDynamic(datatype, constructors, constructor int, name string, selectorNames []string) NaryRecursiveDatatypeConstructor {
	var names NaryDatatypeSelectors
	for _, selectorName := range selectorNames {
		names.Append(selectorName)
	}
	return DeclareNaryRecursiveDatatypeConstructorCompact(datatype, constructors, constructor, name, names)
}

// DeclareNaryRecursiveDatatypeConstructorCompact is the allocation-free
// erased boundary used by generated façades and SMT-LIB execution.
func DeclareNaryRecursiveDatatypeConstructorCompact(datatype, constructors, constructor int, name string, names NaryDatatypeSelectors) NaryRecursiveDatatypeConstructor {
	if constructors < 2 || constructor < 0 || constructor >= constructors {
		panic("smt: n-ary recursive constructor requires a possible base constructor inside datatype cardinality")
	}
	if names.Len() == 0 {
		panic("smt: n-ary recursive constructor requires at least one field")
	}
	for left := 0; left < names.Len(); left++ {
		for right := left + 1; right < names.Len(); right++ {
			if names.At(left) == names.At(right) {
				panic("smt: n-ary recursive constructor selectors must be distinct")
			}
		}
	}
	return naryRecursiveDatatypeConstructorValue{datatypeID: datatype, constructorCount: constructors, constructorID: constructor, arity: names.Len(), name: name, selectorNames: names}
}

// ApplyNaryRecursiveDatatypeConstructorDynamic checks erased arity and sort
// witnesses before retaining a normalized compact argument vector.
func ApplyNaryRecursiveDatatypeConstructorDynamic(declaration NaryRecursiveDatatypeConstructor, values []Term[DatatypeSort]) Term[DatatypeSort] {
	var terms NaryDatatypeTerms
	for _, value := range values {
		terms.Append(value)
	}
	return ApplyNaryRecursiveDatatypeConstructorCompact(declaration, terms)
}

// ApplyNaryRecursiveDatatypeConstructorCompact checks an erased compact term
// vector without introducing a backing slice for the common small arities.
func ApplyNaryRecursiveDatatypeConstructorCompact(declaration NaryRecursiveDatatypeConstructor, values NaryDatatypeTerms) Term[DatatypeSort] {
	witness := declaration
	if values.Len() != witness.arity {
		panic("smt: erased n-ary recursive datatype arity mismatch")
	}
	return datatypeNaryRecursiveApplication[DatatypeSort]{datatypeID: witness.datatypeID, constructorCount: witness.constructorCount, constructorID: witness.constructorID, arity: witness.arity, name: witness.name, selectorNames: witness.selectorNames, values: values}
}

// SelectNaryRecursiveDatatypeConstructorDynamic checks an erased field index.
func SelectNaryRecursiveDatatypeConstructorDynamic(field int, declaration NaryRecursiveDatatypeConstructor, value Term[DatatypeSort]) Term[DatatypeSort] {
	witness := declaration
	if field < 0 || field >= witness.arity {
		panic("smt: erased n-ary recursive datatype selector outside arity")
	}
	return datatypeNaryRecursiveSelector[DatatypeSort]{datatypeID: witness.datatypeID, constructorCount: witness.constructorCount, constructorID: witness.constructorID, arity: witness.arity, field: field, selectorName: witness.selectorNames.At(field), value: value}
}

// DatatypeValue is the exact model value of a supported algebraic datatype.
// IDs are declaration-local ordinals; ConstructorName is retained when the
// corresponding constructor appeared in the authored formula.
type DatatypeValue struct {
	DatatypeID       int
	ConstructorCount int
	ConstructorID    int
	ConstructorName  string
	Child            *DatatypeValue
	SecondChild      *DatatypeValue
	// Children is populated for arbitrary-arity recursive constructors. Unary
	// and binary values retain Child/SecondChild for source compatibility.
	Children *DatatypeChildren
}

// DatatypeChildren keeps the common arity-at-most-four case in one retained
// object while permitting arbitrary arity. DatatypeValue itself remains
// comparable because it contains only this pointer, preserving generated
// equality APIs in packages that embed model values.
type DatatypeChildren struct {
	Count    int
	Inline   [4]DatatypeValue
	Overflow []DatatypeValue
}

func newDatatypeChildren(values []DatatypeValue) *DatatypeChildren {
	children := &DatatypeChildren{Count: len(values)}
	if len(values) <= len(children.Inline) {
		copy(children.Inline[:], values)
	} else {
		children.Overflow = append([]DatatypeValue(nil), values...)
	}
	return children
}

func (children *DatatypeChildren) Len() int {
	if children == nil {
		return 0
	}
	return children.Count
}

func (children *DatatypeChildren) At(index int) (DatatypeValue, bool) {
	if children == nil || index < 0 || index >= children.Count {
		return DatatypeValue{}, false
	}
	if children.Overflow != nil {
		return children.Overflow[index], true
	}
	return children.Inline[index], true
}

type datatypeModelEntry struct {
	datatypeID int
	symbolID   int
	value      DatatypeValue
}

type datatypeModel struct {
	count                  int
	inline                 [8]datatypeModelEntry
	overflow               map[[2]int]DatatypeValue
	childCount             int
	children               *[8]DatatypeValue
	datatypeChildrenCount  int
	inlineDatatypeChildren [8]DatatypeChildren
}

func (model *datatypeModel) retainChild(value DatatypeValue) *DatatypeValue {
	if model.children == nil {
		model.children = new([8]DatatypeValue)
	}
	if model.childCount < len(*model.children) {
		child := &model.children[model.childCount]
		model.childCount++
		*child = value
		return child
	}
	child := new(DatatypeValue)
	*child = value
	return child
}

func (model *datatypeModel) retainDatatypeChildren(count int) *DatatypeChildren {
	if model.datatypeChildrenCount < len(model.inlineDatatypeChildren) && count <= len(model.inlineDatatypeChildren[0].Inline) {
		children := &model.inlineDatatypeChildren[model.datatypeChildrenCount]
		model.datatypeChildrenCount++
		children.Count = count
		return children
	}
	children := &DatatypeChildren{Count: count}
	if count > len(children.Inline) {
		children.Overflow = make([]DatatypeValue, count)
	}
	return children
}

func (children *DatatypeChildren) set(index int, value DatatypeValue) {
	if children.Overflow != nil {
		children.Overflow[index] = value
		return
	}
	children.Inline[index] = value
}

func (model *datatypeModel) set(datatypeID, symbolID int, value DatatypeValue) {
	for index := 0; index < model.count; index++ {
		if model.inline[index].datatypeID == datatypeID && model.inline[index].symbolID == symbolID {
			model.inline[index].value = value
			return
		}
	}
	if model.count < len(model.inline) {
		model.inline[model.count] = datatypeModelEntry{datatypeID: datatypeID, symbolID: symbolID, value: value}
		model.count++
		return
	}
	if model.overflow == nil {
		model.overflow = make(map[[2]int]DatatypeValue)
	}
	model.overflow[[2]int{datatypeID, symbolID}] = value
}

func (model *datatypeModel) lookup(datatypeID, symbolID int) (DatatypeValue, bool) {
	if model == nil {
		return DatatypeValue{}, false
	}
	for index := 0; index < model.count; index++ {
		entry := model.inline[index]
		if entry.datatypeID == datatypeID && entry.symbolID == symbolID {
			return entry.value, true
		}
	}
	value, ok := model.overflow[[2]int{datatypeID, symbolID}]
	return value, ok
}

func evaluateDatatype(term Term[DatatypeSort], model *datatypeModel) (DatatypeValue, bool) {
	switch value := term.(type) {
	case datatypeConstructor[DatatypeSort]:
		return DatatypeValue{DatatypeID: value.datatypeID, ConstructorCount: value.constructorCount, ConstructorID: value.constructorID, ConstructorName: value.name}, true
	case datatypeSymbol[DatatypeSort]:
		return model.lookup(value.datatypeID, value.iD)
	case datatypeRecursiveApplication[DatatypeSort]:
		child, ok := evaluateDatatype(value.value.(Term[DatatypeSort]), model)
		if !ok {
			return DatatypeValue{}, false
		}
		return DatatypeValue{DatatypeID: value.datatypeID, ConstructorCount: value.constructorCount, ConstructorID: value.constructorID, ConstructorName: value.name, Child: &child}, true
	case datatypeRecursiveSelector[DatatypeSort]:
		target, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return DatatypeValue{}, false
		}
		if application, direct := target.(datatypeRecursiveApplication[DatatypeSort]); direct && application.datatypeID == value.datatypeID && application.constructorCount == value.constructorCount && application.constructorID == value.constructorID {
			return evaluateDatatype(application.value.(Term[DatatypeSort]), model)
		}
		modelValue, found := evaluateDatatype(target, model)
		if !found || modelValue.ConstructorID != value.constructorID || modelValue.Child == nil {
			return DatatypeValue{}, false
		}
		return *modelValue.Child, true
	case datatypeBinaryRecursiveApplication[DatatypeSort]:
		first, firstOK := evaluateDatatype(value.first.(Term[DatatypeSort]), model)
		second, secondOK := evaluateDatatype(value.second.(Term[DatatypeSort]), model)
		if !firstOK || !secondOK {
			return DatatypeValue{}, false
		}
		return DatatypeValue{DatatypeID: value.datatypeID, ConstructorCount: value.constructorCount, ConstructorID: value.constructorID, ConstructorName: value.name, Child: &first, SecondChild: &second}, true
	case datatypeBinaryRecursiveSelector[DatatypeSort]:
		target, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return DatatypeValue{}, false
		}
		if application, direct := target.(datatypeBinaryRecursiveApplication[DatatypeSort]); direct && application.datatypeID == value.datatypeID && application.constructorCount == value.constructorCount && application.constructorID == value.constructorID {
			if value.field == 0 {
				return evaluateDatatype(application.first.(Term[DatatypeSort]), model)
			}
			return evaluateDatatype(application.second.(Term[DatatypeSort]), model)
		}
		modelValue, found := evaluateDatatype(target, model)
		if !found || modelValue.ConstructorID != value.constructorID || modelValue.Child == nil || modelValue.SecondChild == nil {
			return DatatypeValue{}, false
		}
		if value.field == 0 {
			return *modelValue.Child, true
		}
		return *modelValue.SecondChild, true
	case datatypeNaryRecursiveApplication[DatatypeSort]:
		children := make([]DatatypeValue, value.values.Len())
		for index := 0; index < value.values.Len(); index++ {
			item := value.values.At(index)
			child, ok := evaluateDatatype(item, model)
			if !ok {
				return DatatypeValue{}, false
			}
			children[index] = child
		}
		return DatatypeValue{DatatypeID: value.datatypeID, ConstructorCount: value.constructorCount, ConstructorID: value.constructorID, ConstructorName: value.name, Children: newDatatypeChildren(children)}, true
	case datatypeNaryRecursiveSelector[DatatypeSort]:
		target, ok := value.value.(Term[DatatypeSort])
		if !ok || value.field < 0 || value.field >= value.arity {
			return DatatypeValue{}, false
		}
		if application, direct := target.(datatypeNaryRecursiveApplication[DatatypeSort]); direct && application.datatypeID == value.datatypeID && application.constructorCount == value.constructorCount && application.constructorID == value.constructorID {
			if value.field < application.values.Len() {
				return evaluateDatatype(application.values.At(value.field), model)
			}
		}
		modelValue, found := evaluateDatatype(target, model)
		if !found || modelValue.ConstructorID != value.constructorID {
			return DatatypeValue{}, false
		}
		return modelValue.Children.At(value.field)
	default:
		return DatatypeValue{}, false
	}
}

func evaluateBoolWithDatatypes(term Term[BoolSort], booleans booleanModel, integers integerModel, reals rationalModel, datatypes *datatypeModel) (bool, bool) {
	switch value := term.(type) {
	case datatypeRecognizer:
		candidate, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return false, false
		}
		actual, found := evaluateDatatype(candidate, datatypes)
		return actual.ConstructorID == value.constructorID, found && actual.DatatypeID == value.datatypeID && actual.ConstructorCount == value.constructorCount
	case datatypeRecursiveRecognizer:
		candidate, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return false, false
		}
		actual, found := evaluateDatatype(candidate, datatypes)
		return actual.ConstructorID == value.constructorID && actual.Child != nil, found && actual.DatatypeID == value.datatypeID && actual.ConstructorCount == value.constructorCount
	case datatypeBinaryRecursiveRecognizer:
		candidate, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return false, false
		}
		actual, found := evaluateDatatype(candidate, datatypes)
		return actual.ConstructorID == value.constructorID && actual.Child != nil && actual.SecondChild != nil, found && actual.DatatypeID == value.datatypeID && actual.ConstructorCount == value.constructorCount
	case datatypeNaryRecursiveRecognizer:
		candidate, ok := value.value.(Term[DatatypeSort])
		if !ok {
			return false, false
		}
		actual, found := evaluateDatatype(candidate, datatypes)
		return actual.ConstructorID == value.constructorID && actual.Children.Len() == value.arity, found && actual.DatatypeID == value.datatypeID && actual.ConstructorCount == value.constructorCount
	case Equal:
		left, leftOK := value.Left.(Term[DatatypeSort])
		right, rightOK := value.Right.(Term[DatatypeSort])
		if leftOK && rightOK {
			leftValue, leftFound := evaluateDatatype(left, datatypes)
			rightValue, rightFound := evaluateDatatype(right, datatypes)
			return equalDatatypeValue(leftValue, rightValue), leftFound && rightFound
		}
	case Not:
		result, ok := evaluateBoolWithDatatypes(value.Value, booleans, integers, reals, datatypes)
		return !result, ok
	case And:
		for _, item := range value.Values {
			result, ok := evaluateBoolWithDatatypes(item, booleans, integers, reals, datatypes)
			if !ok || !result {
				return result, ok
			}
		}
		return true, true
	case BooleanConjunction:
		items, negated := value.values()
		for index, item := range items {
			result, ok := evaluateBoolWithDatatypes(item, booleans, integers, reals, datatypes)
			if !ok || result == negated[index] {
				return false, ok
			}
		}
		return true, true
	case Or:
		for _, item := range value.Values {
			result, ok := evaluateBoolWithDatatypes(item, booleans, integers, reals, datatypes)
			if !ok {
				return false, false
			}
			if result {
				return true, true
			}
		}
		return false, true
	case Implies:
		left, leftOK := evaluateBoolWithDatatypes(value.Left, booleans, integers, reals, datatypes)
		right, rightOK := evaluateBoolWithDatatypes(value.Right, booleans, integers, reals, datatypes)
		return !left || right, leftOK && rightOK
	case Iff:
		left, leftOK := evaluateBoolWithDatatypes(value.Left, booleans, integers, reals, datatypes)
		right, rightOK := evaluateBoolWithDatatypes(value.Right, booleans, integers, reals, datatypes)
		return left == right, leftOK && rightOK
	}
	return evaluateBool(term, booleans, integers, reals)
}

func equalDatatypeValue(left, right DatatypeValue) bool {
	if left.DatatypeID != right.DatatypeID || left.ConstructorCount != right.ConstructorCount || left.ConstructorID != right.ConstructorID || (left.Child == nil) != (right.Child == nil) || (left.SecondChild == nil) != (right.SecondChild == nil) || left.Children.Len() != right.Children.Len() {
		return false
	}
	if left.Child != nil && !equalDatatypeValue(*left.Child, *right.Child) {
		return false
	}
	if left.SecondChild != nil && !equalDatatypeValue(*left.SecondChild, *right.SecondChild) {
		return false
	}
	for index := 0; index < left.Children.Len(); index++ {
		leftChild, _ := left.Children.At(index)
		rightChild, _ := right.Children.At(index)
		if !equalDatatypeValue(leftChild, rightChild) {
			return false
		}
	}
	return true
}

type datatypeNode struct {
	datatypeID       int
	constructorCount int
	kind             uint8
	id               int
	name             string
	child            int
	second           int
	field            int
	children         datatypeNodeChildren
}

type datatypeNodeChildren struct {
	count    int
	inline   [4]int
	overflow []int
}

func (children *datatypeNodeChildren) append(value int) {
	if children.count < len(children.inline) && children.overflow == nil {
		children.inline[children.count] = value
		children.count++
		return
	}
	if children.overflow == nil {
		children.overflow = append(make([]int, 0, children.count+4), children.inline[:]...)
	}
	children.overflow = append(children.overflow, value)
	children.count++
}

func (children datatypeNodeChildren) len() int { return children.count }

func (children datatypeNodeChildren) at(index int) int {
	if children.overflow != nil {
		return children.overflow[index]
	}
	return children.inline[index]
}

type datatypePair struct{ left, right int }
type datatypeTagConstraint struct {
	node        int
	constructor int
	negated     bool
	recursive   bool
	nary        bool
	arity       int
	name        string
}

type datatypeProblem struct {
	nodes               []datatypeNode
	parents             []int
	ranks               []uint8
	disequalities       []datatypePair
	tags                []datatypeTagConstraint
	unsat               bool
	inlineNodes         [8]datatypeNode
	inlineParents       [8]int
	inlineRanks         [8]uint8
	inlineDisequalities [8]datatypePair
	inlineTags          [8]datatypeTagConstraint
	model               datatypeModel
}

func containsDatatypeTheory(term Term[BoolSort]) bool {
	switch value := term.(type) {
	case Equal:
		return isDatatypeTerm(value.Left) || isDatatypeTerm(value.Right)
	case datatypeRecognizer, datatypeRecursiveRecognizer, datatypeBinaryRecursiveRecognizer, datatypeNaryRecursiveRecognizer:
		return true
	case Not:
		return containsDatatypeTheory(value.Value)
	case And:
		for _, item := range value.Values {
			if containsDatatypeTheory(item) {
				return true
			}
		}
	case BooleanConjunction:
		items, _ := value.values()
		for _, item := range items {
			if containsDatatypeTheory(item) {
				return true
			}
		}
	}
	return false
}

func isDatatypeTerm(term any) bool {
	switch term.(type) {
	case datatypeSymbol[DatatypeSort], datatypeConstructor[DatatypeSort], datatypeRecursiveApplication[DatatypeSort], datatypeRecursiveSelector[DatatypeSort], datatypeBinaryRecursiveApplication[DatatypeSort], datatypeBinaryRecursiveSelector[DatatypeSort], datatypeNaryRecursiveApplication[DatatypeSort], datatypeNaryRecursiveSelector[DatatypeSort]:
		return true
	default:
		return false
	}
}

func solveDatatypeAssertions(assertions []Term[BoolSort]) (checkOutcome, bool) {
	problem := datatypeProblem{}
	problem.nodes = problem.inlineNodes[:0]
	problem.parents = problem.inlineParents[:0]
	problem.ranks = problem.inlineRanks[:0]
	problem.disequalities = problem.inlineDisequalities[:0]
	problem.tags = problem.inlineTags[:0]
	for _, assertion := range assertions {
		if !problem.boolean(assertion, false) {
			return checkOutcome{}, false
		}
	}
	if problem.unsat {
		return checkOutcome{status: checkUnsat}, true
	}
	return problem.solve()
}

func (problem *datatypeProblem) boolean(term Term[BoolSort], negated bool) bool {
	switch value := term.(type) {
	case Bool:
		problem.unsat = problem.unsat || value.Value == negated
		return true
	case And:
		if negated {
			return false
		}
		for _, item := range value.Values {
			if !problem.boolean(item, false) {
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
			if !problem.boolean(item, polarities[index]) {
				return false
			}
		}
		return true
	case Not:
		return problem.boolean(value.Value, !negated)
	case Equal:
		left, leftOK := problem.term(value.Left)
		right, rightOK := problem.term(value.Right)
		if !leftOK || !rightOK || !problem.compatible(left, right) {
			return false
		}
		if negated {
			problem.disequalities = append(problem.disequalities, datatypePair{left: left, right: right})
		} else {
			problem.union(left, right)
		}
		return true
	case datatypeRecognizer:
		candidate, ok := problem.term(value.value)
		if !ok || problem.nodes[candidate].datatypeID != value.datatypeID || problem.nodes[candidate].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return false
		}
		candidateNode := problem.nodes[candidate]
		if isDatatypeConstructorNode(candidateNode) {
			matches := candidateNode.id == value.constructorID
			problem.unsat = problem.unsat || matches == negated
			return true
		}
		problem.tags = append(problem.tags, datatypeTagConstraint{node: candidate, constructor: value.constructorID, negated: negated})
		return true
	case datatypeRecursiveRecognizer:
		candidate, ok := problem.term(value.value)
		if !ok || problem.nodes[candidate].datatypeID != value.datatypeID || problem.nodes[candidate].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return false
		}
		candidateNode := problem.nodes[candidate]
		if isDatatypeConstructorNode(candidateNode) {
			matches := candidateNode.kind == 2 && candidateNode.id == value.constructorID
			problem.unsat = problem.unsat || matches == negated
			return true
		}
		problem.tags = append(problem.tags, datatypeTagConstraint{node: candidate, constructor: value.constructorID, negated: negated, recursive: true, arity: 1, name: value.name})
		return true
	case datatypeBinaryRecursiveRecognizer:
		candidate, ok := problem.term(value.value)
		if !ok || problem.nodes[candidate].datatypeID != value.datatypeID || problem.nodes[candidate].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return false
		}
		candidateNode := problem.nodes[candidate]
		if isDatatypeConstructorNode(candidateNode) {
			matches := candidateNode.kind == 4 && candidateNode.id == value.constructorID
			problem.unsat = problem.unsat || matches == negated
			return true
		}
		problem.tags = append(problem.tags, datatypeTagConstraint{node: candidate, constructor: value.constructorID, negated: negated, recursive: true, arity: 2, name: value.name})
		return true
	case datatypeNaryRecursiveRecognizer:
		candidate, ok := problem.term(value.value)
		if !ok || problem.nodes[candidate].datatypeID != value.datatypeID || problem.nodes[candidate].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount || value.arity <= 0 {
			return false
		}
		candidateNode := problem.nodes[candidate]
		if isDatatypeConstructorNode(candidateNode) {
			matches := candidateNode.kind == 6 && candidateNode.id == value.constructorID && candidateNode.children.len() == value.arity
			problem.unsat = problem.unsat || matches == negated
			return true
		}
		problem.tags = append(problem.tags, datatypeTagConstraint{node: candidate, constructor: value.constructorID, negated: negated, recursive: true, nary: true, arity: value.arity, name: value.name})
		return true
	default:
		return false
	}
}

func (problem *datatypeProblem) term(term any) (int, bool) {
	switch value := term.(type) {
	case datatypeSymbol[DatatypeSort]:
		if value.constructorCount <= 0 {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, id: value.iD, name: value.name}), true
	case datatypeConstructor[DatatypeSort]:
		if value.constructorCount <= 0 || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 1, id: value.constructorID, name: value.name}), true
	case datatypeRecursiveApplication[DatatypeSort]:
		child, ok := problem.term(value.value)
		if !ok || problem.nodes[child].datatypeID != value.datatypeID || problem.nodes[child].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 2, id: value.constructorID, name: value.name, child: child}), true
	case datatypeRecursiveSelector[DatatypeSort]:
		target, ok := problem.term(value.value)
		if !ok || problem.nodes[target].datatypeID != value.datatypeID || problem.nodes[target].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 3, id: value.constructorID, name: value.selectorName, child: target}), true
	case datatypeBinaryRecursiveApplication[DatatypeSort]:
		first, firstOK := problem.term(value.first)
		second, secondOK := problem.term(value.second)
		if !firstOK || !secondOK || problem.nodes[first].datatypeID != value.datatypeID || problem.nodes[first].constructorCount != value.constructorCount || problem.nodes[second].datatypeID != value.datatypeID || problem.nodes[second].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 4, id: value.constructorID, name: value.name, child: first, second: second}), true
	case datatypeBinaryRecursiveSelector[DatatypeSort]:
		target, ok := problem.term(value.value)
		if !ok || problem.nodes[target].datatypeID != value.datatypeID || problem.nodes[target].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount || value.field < 0 || value.field >= 2 {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 5, id: value.constructorID, name: value.selectorName, child: target, field: value.field}), true
	case datatypeNaryRecursiveApplication[DatatypeSort]:
		if value.arity <= 0 || value.values.Len() != value.arity || value.selectorNames.Len() != value.arity || value.constructorID < 0 || value.constructorID >= value.constructorCount {
			return 0, false
		}
		var children datatypeNodeChildren
		for index := 0; index < value.values.Len(); index++ {
			item := value.values.At(index)
			child, ok := problem.term(item)
			if !ok || problem.nodes[child].datatypeID != value.datatypeID || problem.nodes[child].constructorCount != value.constructorCount {
				return 0, false
			}
			children.append(child)
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 6, id: value.constructorID, name: value.name, children: children}), true
	case datatypeNaryRecursiveSelector[DatatypeSort]:
		target, ok := problem.term(value.value)
		if !ok || problem.nodes[target].datatypeID != value.datatypeID || problem.nodes[target].constructorCount != value.constructorCount || value.constructorID < 0 || value.constructorID >= value.constructorCount || value.arity <= 0 || value.field < 0 || value.field >= value.arity {
			return 0, false
		}
		return problem.ensure(datatypeNode{datatypeID: value.datatypeID, constructorCount: value.constructorCount, kind: 7, id: value.constructorID, name: value.selectorName, child: target, field: value.field}), true
	default:
		return 0, false
	}
}

func (problem *datatypeProblem) ensure(node datatypeNode) int {
	for index, existing := range problem.nodes {
		samePayload := true
		switch node.kind {
		case 2, 3:
			samePayload = existing.child == node.child
		case 4:
			samePayload = existing.child == node.child && existing.second == node.second
		case 5:
			samePayload = existing.child == node.child && existing.field == node.field
		case 6:
			samePayload = existing.children.len() == node.children.len()
			if samePayload {
				for field := 0; field < node.children.len(); field++ {
					if existing.children.at(field) != node.children.at(field) {
						samePayload = false
						break
					}
				}
			}
		case 7:
			samePayload = existing.child == node.child && existing.field == node.field
		}
		if existing.datatypeID == node.datatypeID && existing.constructorCount == node.constructorCount && existing.kind == node.kind && existing.id == node.id && samePayload {
			if problem.nodes[index].name == "" {
				problem.nodes[index].name = node.name
			}
			return index
		}
	}
	index := len(problem.nodes)
	problem.nodes = append(problem.nodes, node)
	problem.parents = append(problem.parents, index)
	problem.ranks = append(problem.ranks, 0)
	return index
}

func (problem *datatypeProblem) compatible(left, right int) bool {
	return problem.nodes[left].datatypeID == problem.nodes[right].datatypeID && problem.nodes[left].constructorCount == problem.nodes[right].constructorCount
}

func (problem *datatypeProblem) find(node int) int {
	root := node
	for problem.parents[root] != root {
		root = problem.parents[root]
	}
	for problem.parents[node] != node {
		next := problem.parents[node]
		problem.parents[node] = root
		node = next
	}
	return root
}

func (problem *datatypeProblem) union(left, right int) {
	left, right = problem.find(left), problem.find(right)
	if left == right {
		return
	}
	leftNode, rightNode := problem.nodes[left], problem.nodes[right]
	leftConstructor, rightConstructor := isDatatypeConstructorNode(leftNode), isDatatypeConstructorNode(rightNode)
	if !problem.compatible(left, right) || leftConstructor && rightConstructor && (leftNode.id != rightNode.id || leftNode.kind != rightNode.kind) {
		problem.unsat = true
		return
	}
	if leftNode.kind == 2 && rightNode.kind == 2 {
		problem.union(leftNode.child, rightNode.child)
		if problem.unsat {
			return
		}
	}
	if leftNode.kind == 4 && rightNode.kind == 4 {
		problem.union(leftNode.child, rightNode.child)
		if problem.unsat {
			return
		}
		problem.union(leftNode.second, rightNode.second)
		if problem.unsat {
			return
		}
	}
	if leftNode.kind == 6 && rightNode.kind == 6 {
		if leftNode.children.len() != rightNode.children.len() {
			problem.unsat = true
			return
		}
		for field := 0; field < leftNode.children.len(); field++ {
			problem.union(leftNode.children.at(field), rightNode.children.at(field))
			if problem.unsat {
				return
			}
		}
	}
	if problem.ranks[left] < problem.ranks[right] {
		left, right = right, left
	}
	problem.parents[right] = left
	if problem.ranks[left] == problem.ranks[right] {
		problem.ranks[left]++
	}
}

func isDatatypeConstructorNode(node datatypeNode) bool {
	return node.kind == 1 || node.kind == 2 || node.kind == 4 || node.kind == 6
}

func (problem *datatypeProblem) solve() (checkOutcome, bool) {
	if problem.unsat {
		return checkOutcome{status: checkUnsat}, true
	}
	for {
		changed := false
		for selector, selectorNode := range problem.nodes {
			if selectorNode.kind != 3 && selectorNode.kind != 5 && selectorNode.kind != 7 {
				continue
			}
			targetRoot := problem.find(selectorNode.child)
			for application, applicationNode := range problem.nodes {
				selected := -1
				if selectorNode.kind == 3 && applicationNode.kind == 2 {
					selected = applicationNode.child
				} else if selectorNode.kind == 5 && applicationNode.kind == 4 {
					selected = applicationNode.child
					if selectorNode.field == 1 {
						selected = applicationNode.second
					}
				} else if selectorNode.kind == 7 && applicationNode.kind == 6 && selectorNode.field < applicationNode.children.len() {
					selected = applicationNode.children.at(selectorNode.field)
				}
				if selected >= 0 && applicationNode.id == selectorNode.id && problem.find(application) == targetRoot && problem.find(selector) != problem.find(selected) {
					problem.union(selector, selected)
					changed = true
					break
				}
			}
		}
		for left := 0; left < len(problem.nodes); left++ {
			if problem.nodes[left].kind != 2 && problem.nodes[left].kind != 4 && problem.nodes[left].kind != 6 {
				continue
			}
			for right := left + 1; right < len(problem.nodes); right++ {
				sameChildren := problem.find(problem.nodes[left].child) == problem.find(problem.nodes[right].child)
				if problem.nodes[left].kind == 4 {
					sameChildren = sameChildren && problem.find(problem.nodes[left].second) == problem.find(problem.nodes[right].second)
				} else if problem.nodes[left].kind == 6 {
					sameChildren = problem.nodes[left].children.len() == problem.nodes[right].children.len()
					if sameChildren {
						for field := 0; field < problem.nodes[left].children.len(); field++ {
							if problem.find(problem.nodes[left].children.at(field)) != problem.find(problem.nodes[right].children.at(field)) {
								sameChildren = false
								break
							}
						}
					}
				}
				if problem.nodes[right].kind == problem.nodes[left].kind && problem.nodes[left].datatypeID == problem.nodes[right].datatypeID && problem.nodes[left].constructorCount == problem.nodes[right].constructorCount && problem.nodes[left].id == problem.nodes[right].id && sameChildren && problem.find(left) != problem.find(right) {
					problem.union(left, right)
					changed = true
				}
			}
		}
		if !changed || problem.unsat {
			break
		}
	}
	if problem.unsat || problem.hasRecursiveCycle() {
		return checkOutcome{status: checkUnsat}, true
	}
	for _, pair := range problem.disequalities {
		if problem.find(pair.left) == problem.find(pair.right) {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	var inlineAssignment [8]int
	var assignment []int
	if len(problem.nodes) <= len(inlineAssignment) {
		assignment = inlineAssignment[:len(problem.nodes)]
	} else {
		assignment = make([]int, len(problem.nodes))
	}
	for index := range assignment {
		assignment[index] = -1
	}
	for index, node := range problem.nodes {
		if isDatatypeConstructorNode(node) {
			root := problem.find(index)
			if assignment[root] >= 0 && assignment[root] != node.id {
				return checkOutcome{status: checkUnsat}, true
			}
			assignment[root] = node.id
		}
	}
	for _, tag := range problem.tags {
		root := problem.find(tag.node)
		if !tag.negated {
			if assignment[root] >= 0 && assignment[root] != tag.constructor {
				return checkOutcome{status: checkUnsat}, true
			}
			assignment[root] = tag.constructor
		}
	}
	var inlineRoots [8]int
	roots := inlineRoots[:0]
	if len(problem.nodes) > cap(inlineRoots) {
		roots = make([]int, 0, len(problem.nodes))
	}
	for index := range problem.nodes {
		root := problem.find(index)
		if root == index && (problem.nodes[index].kind == 0 || problem.nodes[index].kind == 3 || problem.nodes[index].kind == 5 || problem.nodes[index].kind == 7) {
			roots = append(roots, root)
		}
	}
	if !problem.color(roots, 0, assignment) {
		return checkOutcome{status: checkUnsat}, true
	}
	model := &problem.model
	for index, node := range problem.nodes {
		if node.kind != 0 {
			continue
		}
		var inlineVisiting [8]bool
		var visiting []bool
		if len(problem.nodes) <= len(inlineVisiting) {
			visiting = inlineVisiting[:len(problem.nodes)]
		} else {
			visiting = make([]bool, len(problem.nodes))
		}
		value, ok := problem.valueForRoot(problem.find(index), assignment, visiting, model)
		if !ok {
			return checkOutcome{status: checkUnknown, reason: UnsupportedTheory{Name: "recursive datatype model construction"}}, true
		}
		model.set(node.datatypeID, node.id, value)
	}
	return checkOutcome{status: checkSat, datatypes: model}, true
}

func (problem *datatypeProblem) valueForRoot(root int, assignment []int, visiting []bool, model *datatypeModel) (DatatypeValue, bool) {
	root = problem.find(root)
	if visiting[root] {
		return DatatypeValue{}, false
	}
	visiting[root] = true
	defer func() { visiting[root] = false }()
	for index, node := range problem.nodes {
		if node.kind != 2 && node.kind != 4 && node.kind != 6 || problem.find(index) != root {
			continue
		}
		if node.kind == 6 {
			children := model.retainDatatypeChildren(node.children.len())
			for field := 0; field < node.children.len(); field++ {
				childNode := node.children.at(field)
				child, ok := problem.valueForRoot(problem.find(childNode), assignment, visiting, model)
				if !ok {
					return DatatypeValue{}, false
				}
				children.set(field, child)
			}
			return DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: node.id, ConstructorName: node.name, Children: children}, true
		}
		first, ok := problem.valueForRoot(problem.find(node.child), assignment, visiting, model)
		if !ok {
			return DatatypeValue{}, false
		}
		value := DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: node.id, ConstructorName: node.name, Child: model.retainChild(first)}
		if node.kind == 4 {
			second, secondOK := problem.valueForRoot(problem.find(node.second), assignment, visiting, model)
			if !secondOK {
				return DatatypeValue{}, false
			}
			value.SecondChild = model.retainChild(second)
		}
		return value, true
	}
	node := problem.nodes[root]
	constructorID := assignment[root]
	if constructorID < 0 {
		return DatatypeValue{}, false
	}
	value := DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: constructorID, ConstructorName: problem.constructorName(node.datatypeID, node.constructorCount, constructorID)}
	if arity := problem.recursiveConstructorArity(root, constructorID); arity > 0 {
		base := problem.baseConstructor(node.datatypeID, node.constructorCount)
		if base < 0 {
			return DatatypeValue{}, false
		}
		child := DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: base, ConstructorName: problem.constructorName(node.datatypeID, node.constructorCount, base)}
		if problem.naryRecursiveConstructor(root, constructorID) {
			children := make([]DatatypeValue, arity)
			for field := range children {
				children[field] = child
			}
			value.Children = newDatatypeChildren(children)
		} else if arity <= 2 {
			value.Child = model.retainChild(child)
		}
		if !problem.naryRecursiveConstructor(root, constructorID) && arity == 2 {
			value.SecondChild = model.retainChild(child)
		}
	}
	return value, true
}

func (problem *datatypeProblem) hasRecursiveCycle() bool {
	var inlineState [8]uint8
	var state []uint8
	if len(problem.nodes) <= len(inlineState) {
		state = inlineState[:len(problem.nodes)]
	} else {
		state = make([]uint8, len(problem.nodes))
	}
	for index := range problem.nodes {
		root := problem.find(index)
		if state[root] == 0 && problem.recursiveCycleFrom(root, state) {
			return true
		}
	}
	return false
}

func (problem *datatypeProblem) recursiveCycleFrom(root int, state []uint8) bool {
	root = problem.find(root)
	if state[root] == 1 {
		return true
	}
	if state[root] == 2 {
		return false
	}
	state[root] = 1
	for index, node := range problem.nodes {
		if problem.find(index) != root || node.kind != 2 && node.kind != 4 && node.kind != 6 {
			continue
		}
		if node.kind == 2 || node.kind == 4 {
			if problem.recursiveCycleFrom(node.child, state) {
				return true
			}
			if node.kind == 4 && problem.recursiveCycleFrom(node.second, state) {
				return true
			}
		} else if node.kind == 6 {
			for field := 0; field < node.children.len(); field++ {
				if problem.recursiveCycleFrom(node.children.at(field), state) {
					return true
				}
			}
		}
	}
	state[root] = 2
	return false
}

func (problem *datatypeProblem) color(roots []int, position int, assignment []int) bool {
	if position == len(roots) {
		return true
	}
	root := roots[position]
	if assignment[root] >= 0 {
		if problem.assignmentAllowed(root, assignment[root], assignment) {
			return problem.color(roots, position+1, assignment)
		}
		return false
	}
	for constructor := 0; constructor < problem.nodes[root].constructorCount; constructor++ {
		if problem.assignmentAllowed(root, constructor, assignment) {
			assignment[root] = constructor
			if problem.color(roots, position+1, assignment) {
				return true
			}
			assignment[root] = -1
		}
	}
	return false
}

func (problem *datatypeProblem) assignmentAllowed(root, constructor int, assignment []int) bool {
	for _, tag := range problem.tags {
		if problem.find(tag.node) == root && (constructor == tag.constructor) == tag.negated {
			return false
		}
	}
	for _, pair := range problem.disequalities {
		left, right := problem.find(pair.left), problem.find(pair.right)
		if (left == root && assignment[right] == constructor || right == root && assignment[left] == constructor) && !problem.recursiveConstructor(root, constructor) {
			return false
		}
	}
	return true
}

func (problem *datatypeProblem) recursiveConstructor(root, constructor int) bool {
	return problem.recursiveConstructorArity(root, constructor) > 0
}

func (problem *datatypeProblem) recursiveConstructorArity(root, constructor int) int {
	node := problem.nodes[root]
	for _, candidate := range problem.nodes {
		if (candidate.kind == 2 || candidate.kind == 4 || candidate.kind == 6) && candidate.datatypeID == node.datatypeID && candidate.constructorCount == node.constructorCount && candidate.id == constructor {
			if candidate.kind == 6 {
				return candidate.children.len()
			}
			if candidate.kind == 4 {
				return 2
			}
			return 1
		}
	}
	for _, tag := range problem.tags {
		if tag.recursive && tag.constructor == constructor && problem.nodes[tag.node].datatypeID == node.datatypeID && problem.nodes[tag.node].constructorCount == node.constructorCount {
			return int(tag.arity)
		}
	}
	return 0
}

func (problem *datatypeProblem) naryRecursiveConstructor(root, constructor int) bool {
	node := problem.nodes[root]
	for _, candidate := range problem.nodes {
		if candidate.kind == 6 && candidate.datatypeID == node.datatypeID && candidate.constructorCount == node.constructorCount && candidate.id == constructor {
			return true
		}
	}
	for _, tag := range problem.tags {
		if tag.recursive && tag.nary && tag.constructor == constructor && problem.nodes[tag.node].datatypeID == node.datatypeID && problem.nodes[tag.node].constructorCount == node.constructorCount {
			return true
		}
	}
	return false
}

func (problem *datatypeProblem) baseConstructor(datatypeID, constructorCount int) int {
	for constructor := 0; constructor < constructorCount; constructor++ {
		recursive := false
		for index := range problem.nodes {
			if problem.nodes[index].datatypeID == datatypeID && problem.nodes[index].constructorCount == constructorCount && problem.recursiveConstructor(index, constructor) {
				recursive = true
				break
			}
		}
		if !recursive {
			return constructor
		}
	}
	return -1
}

func (problem *datatypeProblem) constructorName(datatypeID, constructorCount, constructorID int) string {
	for _, node := range problem.nodes {
		if isDatatypeConstructorNode(node) && node.datatypeID == datatypeID && node.constructorCount == constructorCount && node.id == constructorID {
			return node.name
		}
	}
	for _, tag := range problem.tags {
		if tag.constructor == constructorID && tag.name != "" && problem.nodes[tag.node].datatypeID == datatypeID && problem.nodes[tag.node].constructorCount == constructorCount {
			return tag.name
		}
	}
	return ""
}
