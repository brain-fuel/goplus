package smt

// DatatypeValue is the exact model value of a supported algebraic datatype.
// IDs are declaration-local ordinals; ConstructorName is retained when the
// corresponding constructor appeared in the authored formula.
type DatatypeValue struct {
	DatatypeID       int
	ConstructorCount int
	ConstructorID    int
	ConstructorName  string
	Child            *DatatypeValue
}

type datatypeModelEntry struct {
	datatypeID int
	symbolID   int
	value      DatatypeValue
}

type datatypeModel struct {
	count      int
	inline     [8]datatypeModelEntry
	overflow   map[[2]int]DatatypeValue
	childCount int
	children   *[8]DatatypeValue
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

func (model datatypeModel) lookup(datatypeID, symbolID int) (DatatypeValue, bool) {
	for index := 0; index < model.count; index++ {
		entry := model.inline[index]
		if entry.datatypeID == datatypeID && entry.symbolID == symbolID {
			return entry.value, true
		}
	}
	value, ok := model.overflow[[2]int{datatypeID, symbolID}]
	return value, ok
}

func evaluateDatatype(term Term[DatatypeSort], model datatypeModel) (DatatypeValue, bool) {
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
	default:
		return DatatypeValue{}, false
	}
}

func evaluateBoolWithDatatypes(term Term[BoolSort], booleans booleanModel, integers integerModel, reals rationalModel, datatypes datatypeModel) (bool, bool) {
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
	if left.DatatypeID != right.DatatypeID || left.ConstructorCount != right.ConstructorCount || left.ConstructorID != right.ConstructorID || (left.Child == nil) != (right.Child == nil) {
		return false
	}
	return left.Child == nil || equalDatatypeValue(*left.Child, *right.Child)
}

type datatypeNode struct {
	datatypeID       int
	constructorCount int
	kind             uint8
	id               int
	name             string
	child            int
}

type datatypePair struct{ left, right int }
type datatypeTagConstraint struct {
	node        int
	constructor int
	negated     bool
	recursive   bool
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
}

func containsDatatypeTheory(term Term[BoolSort]) bool {
	switch value := term.(type) {
	case Equal:
		return isDatatypeTerm(value.Left) || isDatatypeTerm(value.Right)
	case datatypeRecognizer, datatypeRecursiveRecognizer:
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
	case datatypeSymbol[DatatypeSort], datatypeConstructor[DatatypeSort], datatypeRecursiveApplication[DatatypeSort], datatypeRecursiveSelector[DatatypeSort]:
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
		if candidateNode.kind == 1 || candidateNode.kind == 2 {
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
		if candidateNode.kind == 1 || candidateNode.kind == 2 {
			matches := candidateNode.kind == 2 && candidateNode.id == value.constructorID
			problem.unsat = problem.unsat || matches == negated
			return true
		}
		problem.tags = append(problem.tags, datatypeTagConstraint{node: candidate, constructor: value.constructorID, negated: negated, recursive: true, name: value.name})
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
	default:
		return 0, false
	}
}

func (problem *datatypeProblem) ensure(node datatypeNode) int {
	for index, existing := range problem.nodes {
		if existing.datatypeID == node.datatypeID && existing.constructorCount == node.constructorCount && existing.kind == node.kind && existing.id == node.id && (node.kind != 2 && node.kind != 3 || existing.child == node.child) {
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
	leftConstructor, rightConstructor := leftNode.kind == 1 || leftNode.kind == 2, rightNode.kind == 1 || rightNode.kind == 2
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
	if problem.ranks[left] < problem.ranks[right] {
		left, right = right, left
	}
	problem.parents[right] = left
	if problem.ranks[left] == problem.ranks[right] {
		problem.ranks[left]++
	}
}

func (problem *datatypeProblem) solve() (checkOutcome, bool) {
	if problem.unsat {
		return checkOutcome{status: checkUnsat}, true
	}
	for {
		changed := false
		for selector, selectorNode := range problem.nodes {
			if selectorNode.kind != 3 {
				continue
			}
			targetRoot := problem.find(selectorNode.child)
			for application, applicationNode := range problem.nodes {
				if applicationNode.kind == 2 && applicationNode.id == selectorNode.id && problem.find(application) == targetRoot && problem.find(selector) != problem.find(applicationNode.child) {
					problem.union(selector, applicationNode.child)
					changed = true
					break
				}
			}
		}
		for left := 0; left < len(problem.nodes); left++ {
			if problem.nodes[left].kind != 2 {
				continue
			}
			for right := left + 1; right < len(problem.nodes); right++ {
				if problem.nodes[right].kind == 2 && problem.nodes[left].datatypeID == problem.nodes[right].datatypeID && problem.nodes[left].constructorCount == problem.nodes[right].constructorCount && problem.nodes[left].id == problem.nodes[right].id && problem.find(problem.nodes[left].child) == problem.find(problem.nodes[right].child) && problem.find(left) != problem.find(right) {
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
	assignment := make([]int, len(problem.nodes))
	for index := range assignment {
		assignment[index] = -1
	}
	for index, node := range problem.nodes {
		if node.kind == 1 || node.kind == 2 {
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
	roots := make([]int, 0, len(problem.nodes))
	for index := range problem.nodes {
		root := problem.find(index)
		if root == index && (problem.nodes[index].kind == 0 || problem.nodes[index].kind == 3) {
			roots = append(roots, root)
		}
	}
	if !problem.color(roots, 0, assignment) {
		return checkOutcome{status: checkUnsat}, true
	}
	var model datatypeModel
	for index, node := range problem.nodes {
		if node.kind != 0 {
			continue
		}
		value, ok := problem.valueForRoot(problem.find(index), assignment, map[int]bool{}, &model)
		if !ok {
			return checkOutcome{status: checkUnknown, reason: UnsupportedTheory{Name: "recursive datatype model construction"}}, true
		}
		model.set(node.datatypeID, node.id, value)
	}
	return checkOutcome{status: checkSat, datatypes: model}, true
}

func (problem *datatypeProblem) valueForRoot(root int, assignment []int, visiting map[int]bool, model *datatypeModel) (DatatypeValue, bool) {
	root = problem.find(root)
	if visiting[root] {
		return DatatypeValue{}, false
	}
	visiting[root] = true
	defer delete(visiting, root)
	for index, node := range problem.nodes {
		if node.kind != 2 || problem.find(index) != root {
			continue
		}
		child, ok := problem.valueForRoot(problem.find(node.child), assignment, visiting, model)
		if !ok {
			return DatatypeValue{}, false
		}
		return DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: node.id, ConstructorName: node.name, Child: model.retainChild(child)}, true
	}
	node := problem.nodes[root]
	constructorID := assignment[root]
	if constructorID < 0 {
		return DatatypeValue{}, false
	}
	value := DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: constructorID, ConstructorName: problem.constructorName(node.datatypeID, node.constructorCount, constructorID)}
	if problem.recursiveConstructor(root, constructorID) {
		base := problem.baseConstructor(node.datatypeID, node.constructorCount)
		if base < 0 {
			return DatatypeValue{}, false
		}
		child := DatatypeValue{DatatypeID: node.datatypeID, ConstructorCount: node.constructorCount, ConstructorID: base, ConstructorName: problem.constructorName(node.datatypeID, node.constructorCount, base)}
		value.Child = model.retainChild(child)
	}
	return value, true
}

func (problem *datatypeProblem) hasRecursiveCycle() bool {
	for index, node := range problem.nodes {
		if node.kind != 2 {
			continue
		}
		origin := problem.find(index)
		current := problem.find(node.child)
		seen := map[int]bool{}
		for !seen[current] {
			if current == origin {
				return true
			}
			seen[current] = true
			next := -1
			for candidate, candidateNode := range problem.nodes {
				if candidateNode.kind == 2 && problem.find(candidate) == current {
					next = problem.find(candidateNode.child)
					break
				}
			}
			if next < 0 {
				break
			}
			current = next
		}
	}
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
	node := problem.nodes[root]
	for _, candidate := range problem.nodes {
		if candidate.kind == 2 && candidate.datatypeID == node.datatypeID && candidate.constructorCount == node.constructorCount && candidate.id == constructor {
			return true
		}
	}
	for _, tag := range problem.tags {
		if tag.recursive && tag.constructor == constructor && problem.nodes[tag.node].datatypeID == node.datatypeID && problem.nodes[tag.node].constructorCount == node.constructorCount {
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
		if (node.kind == 1 || node.kind == 2) && node.datatypeID == datatypeID && node.constructorCount == constructorCount && node.id == constructorID {
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
