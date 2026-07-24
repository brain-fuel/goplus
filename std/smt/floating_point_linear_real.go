package smt

type sharedFloatingPointRealCollector struct {
	bitCount      int
	bits          [8]Term[BoolSort]
	bitOverflow   []Term[BoolSort]
	realCount     int
	reals         [8]Term[BoolSort]
	realOverflow  []Term[BoolSort]
	mixedCount    int
	mixed         [4]FloatingPointToRealRelation
	mixedOverflow []FloatingPointToRealRelation
}

func (collector *sharedFloatingPointRealCollector) appendBit(
	value Term[BoolSort],
) {
	if collector.bitCount < len(collector.bits) &&
		collector.bitOverflow == nil {
		collector.bits[collector.bitCount] = value
		collector.bitCount++
		return
	}
	if collector.bitOverflow == nil {
		collector.bitOverflow = make(
			[]Term[BoolSort], collector.bitCount, collector.bitCount*2,
		)
		copy(collector.bitOverflow, collector.bits[:collector.bitCount])
	}
	collector.bitOverflow = append(collector.bitOverflow, value)
	collector.bitCount++
}

func (collector *sharedFloatingPointRealCollector) appendReal(
	value Term[BoolSort],
) {
	if collector.realCount < len(collector.reals) &&
		collector.realOverflow == nil {
		collector.reals[collector.realCount] = value
		collector.realCount++
		return
	}
	if collector.realOverflow == nil {
		collector.realOverflow = make(
			[]Term[BoolSort], collector.realCount, collector.realCount*2,
		)
		copy(collector.realOverflow, collector.reals[:collector.realCount])
	}
	collector.realOverflow = append(collector.realOverflow, value)
	collector.realCount++
}

func (collector *sharedFloatingPointRealCollector) appendMixed(
	value FloatingPointToRealRelation,
) {
	if collector.mixedCount < len(collector.mixed) &&
		collector.mixedOverflow == nil {
		collector.mixed[collector.mixedCount] = value
		collector.mixedCount++
		return
	}
	if collector.mixedOverflow == nil {
		collector.mixedOverflow = make(
			[]FloatingPointToRealRelation,
			collector.mixedCount, collector.mixedCount*2,
		)
		copy(collector.mixedOverflow, collector.mixed[:collector.mixedCount])
	}
	collector.mixedOverflow = append(collector.mixedOverflow, value)
	collector.mixedCount++
}

func (collector *sharedFloatingPointRealCollector) bitValues() []Term[BoolSort] {
	if collector.bitOverflow != nil {
		return collector.bitOverflow[:collector.bitCount]
	}
	return collector.bits[:collector.bitCount]
}

func (collector *sharedFloatingPointRealCollector) realValues() []Term[BoolSort] {
	if collector.realOverflow != nil {
		return collector.realOverflow[:collector.realCount]
	}
	return collector.reals[:collector.realCount]
}

func (collector *sharedFloatingPointRealCollector) mixedValues() []FloatingPointToRealRelation {
	if collector.mixedOverflow != nil {
		return collector.mixedOverflow[:collector.mixedCount]
	}
	return collector.mixed[:collector.mixedCount]
}

func (collector *sharedFloatingPointRealCollector) add(
	term Term[BoolSort],
	negated bool,
) bool {
	switch value := term.(type) {
	case And:
		if negated {
			return false
		}
		for _, item := range value.Values {
			if !collector.add(item, false) {
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
			if !collector.add(item, polarities[index]) {
				return false
			}
		}
		return true
	case TheoryConjunction:
		if negated {
			return false
		}
		atoms, polarities := value.atomValues()
		for index, item := range atoms {
			if !collector.add(item, polarities[index]) {
				return false
			}
		}
		for _, constraint := range value.realValues() {
			collector.appendReal(constraint)
		}
		return true
	case Not:
		return collector.add(value.Value, !negated)
	case FloatingPointToRealRelation:
		if value.RealCount == 0 {
			if negated {
				value.Negated = !value.Negated
			}
			collector.appendBit(value)
			return true
		}
		value.Negated = value.Negated != negated
		if value.Negated && value.Comparison == 0 {
			return false
		}
		collector.appendMixed(value)
		return true
	default:
		if negated {
			return false
		}
		if containsBitVectorTheory(term) {
			collector.appendBit(term)
			return true
		}
		if containsRealTheory(term) {
			collector.appendReal(term)
			return true
		}
		return false
	}
}

// solveSharedFloatingPointReal solves conjunctions in which assigned
// floating-point symbols occur through fp.to_real in affine LRA constraints.
// Floating-point assignments are decided first, then substituted exactly into
// the Real system; the resulting models are merged.
func solveSharedFloatingPointReal(
	assertions []Term[BoolSort],
) (checkOutcome, bool) {
	var collector sharedFloatingPointRealCollector
	for _, assertion := range assertions {
		if !collector.add(assertion, false) {
			return checkOutcome{}, false
		}
	}
	if collector.mixedCount == 0 || collector.bitCount == 0 {
		return checkOutcome{}, false
	}
	bitTerms := collector.bitValues()
	realTerms := collector.realValues()
	mixed := collector.mixedValues()
	bitOutcome, recognized := solveCompactBitVectorAssertions(bitTerms)
	if !recognized || bitOutcome.status != checkSat {
		return bitOutcome, recognized
	}
	if direct, recognized := solveDirectFloatingPointRealBridge(
		mixed, realTerms, bitOutcome,
	); recognized {
		return direct, true
	}
	for _, relation := range mixed {
		constraint, ok := floatingPointRealLinearConstraint(
			relation, bitOutcome.bitVectors,
		)
		if !ok {
			return checkOutcome{}, false
		}
		if relation.Negated {
			negateLinearRealConstraint(&constraint)
			constraint.Strict = relation.Comparison == 1
			realTerms = append(realTerms, constraint)
			continue
		}
		constraint.Strict = relation.Comparison == 2
		realTerms = append(realTerms, constraint)
		if relation.Comparison == 0 {
			negateLinearRealConstraint(&constraint)
			realTerms = append(realTerms, constraint)
		}
	}
	realOutcome, recognized := solveLinearRealAssertions(realTerms)
	if !recognized || realOutcome.status != checkSat {
		return realOutcome, recognized
	}
	realOutcome.bitVectors = bitOutcome.bitVectors
	realOutcome.booleans = bitOutcome.booleans
	return realOutcome, true
}

func solveDirectFloatingPointRealBridge(
	relations []FloatingPointToRealRelation,
	realTerms []Term[BoolSort],
	bitOutcome checkOutcome,
) (checkOutcome, bool) {
	var reals rationalModel
	for _, relation := range relations {
		if relation.Negated || relation.Comparison != 0 ||
			relation.RealCount != 1 {
			return checkOutcome{}, false
		}
		constraint, ok := floatingPointRealLinearConstraint(
			relation, bitOutcome.bitVectors,
		)
		if !ok {
			return checkOutcome{}, false
		}
		real := relation.realValues()[0]
		value := DivideRational(
			NegateRational(constraint.Constant), real.Coefficient,
		)
		if previous, found := reals.lookup(real.SymbolID); found {
			if CompareRational(previous, value) != 0 {
				return checkOutcome{status: checkUnsat}, true
			}
			continue
		}
		reals.set(real.SymbolID, value)
	}
	for _, term := range realTerms {
		holds, ok := evaluateBool(
			term, booleanModel{}, integerModel{}, reals,
		)
		if !ok {
			return checkOutcome{}, false
		}
		if !holds {
			return checkOutcome{status: checkUnsat}, true
		}
	}
	return checkOutcome{
		status: checkSat, reals: reals,
		bitVectors: bitOutcome.bitVectors,
		booleans:   bitOutcome.booleans,
	}, true
}

func floatingPointRealLinearConstraint(
	relation FloatingPointToRealRelation,
	model bitVectorModel,
) (LinearRealConstraint, bool) {
	constant := relation.Constant
	for _, term := range relation.values() {
		bits, found := model.lookup(term.SymbolID)
		if !found ||
			bits.Width() != term.ExponentBits+term.SignificandBits {
			return LinearRealConstraint{}, false
		}
		value, valid := floatingPointToRational(FloatingPointFromBits(
			term.ExponentBits, term.SignificandBits, bits,
		))
		if !valid {
			value = Rational{}
		}
		constant = AddRational(
			constant, MultiplyRational(term.Coefficient, value),
		)
	}
	constraint := LinearRealConstraint{
		Count: relation.RealCount, Constant: constant,
	}
	if relation.RealCount > len(constraint.Symbols) {
		constraint.OverflowSymbols = make([]int, relation.RealCount)
		constraint.OverflowCoefficients = make([]Rational, relation.RealCount)
	}
	for index, term := range relation.realValues() {
		if constraint.OverflowSymbols != nil {
			constraint.OverflowSymbols[index] = term.SymbolID
			constraint.OverflowCoefficients[index] = term.Coefficient
		} else {
			constraint.Symbols[index] = term.SymbolID
			constraint.Coefficients[index] = term.Coefficient
		}
	}
	return constraint, true
}

func negateLinearRealConstraint(constraint *LinearRealConstraint) {
	constraint.Constant = NegateRational(constraint.Constant)
	if constraint.OverflowCoefficients != nil {
		for index := range constraint.OverflowCoefficients {
			constraint.OverflowCoefficients[index] =
				NegateRational(constraint.OverflowCoefficients[index])
		}
		return
	}
	for index := 0; index < constraint.Count; index++ {
		constraint.Coefficients[index] =
			NegateRational(constraint.Coefficients[index])
	}
}
