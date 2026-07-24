package smtlib

import (
	"strconv"
	"strings"

	smt "goforge.dev/goplus/std/smt"
)

const (
	fpFastAcknowledge uint8 = iota + 1
	fpFastDeclare
	fpFastAssign
	fpFastRound
	fpFastCheck
)

type fpFastCommand struct {
	kind                          uint8
	symbolID, exponentBits        int
	significandBits, commandIndex int
	name                          string
	value                         smt.BitVectorValue
	mode                          smt.FloatingPointRoundingMode
	negated                       bool
}

type fpFastSymbol struct {
	name             string
	id, exponentBits int
	significandBits  int
}

type fpFastToken struct {
	kind       uint8
	start, end int
}

const (
	fpFastAtom uint8 = iota + 1
	fpFastLeft
	fpFastRight
)

type fpFastScanner struct {
	source string
	at     int
}

type fpFastOperand struct {
	kind                   uint8
	symbolID, exponentBits int
	significandBits        int
	value                  smt.BitVectorValue
	mode                   smt.FloatingPointRoundingMode
}

const (
	fpFastSymbolBits uint8 = iota + 1
	fpFastRoundedBits
	fpFastLiteralBits
)

func executeFloatingPointFast(source string) (ExecutionResult, bool) {
	if !strings.Contains(source, "FloatingPoint") ||
		!strings.Contains(source, "QF_FP") {
		return nil, false
	}
	var commands [32]fpFastCommand
	var symbols [16]fpFastSymbol
	commandCount, symbolCount := 0, 0
	scanner := fpFastScanner{source: source}
	for {
		token, ok := scanner.next()
		if !ok {
			break
		}
		if token.kind != fpFastLeft || commandCount == len(commands) {
			return nil, false
		}
		operator, ok := scanner.atom()
		if !ok {
			return nil, false
		}
		command := fpFastCommand{commandIndex: commandCount}
		switch scanner.text(operator) {
		case "set-logic":
			logic, logicOK := scanner.atom()
			if !logicOK || scanner.text(logic) != "QF_FP" ||
				!scanner.right() {
				return nil, false
			}
			command.kind = fpFastAcknowledge
		case "declare-const":
			name, nameOK := scanner.atom()
			if !nameOK || symbolCount == len(symbols) ||
				!scanner.left() {
				return nil, false
			}
			marker, markerOK := scanner.atom()
			sort, sortOK := scanner.atom()
			exponent, exponentOK := scanner.positiveInt()
			significand, significandOK := scanner.positiveInt()
			if !markerOK || !sortOK || !exponentOK || !significandOK ||
				scanner.text(marker) != "_" ||
				scanner.text(sort) != "FloatingPoint" ||
				exponent < 2 || significand < 2 ||
				!scanner.right() || !scanner.right() {
				return nil, false
			}
			nameText := scanner.text(name)
			for index := 0; index < symbolCount; index++ {
				if symbols[index].name == nameText {
					return nil, false
				}
			}
			symbolCount++
			symbols[symbolCount-1] = fpFastSymbol{
				name: nameText, id: symbolCount,
				exponentBits: exponent, significandBits: significand,
			}
			command.kind = fpFastDeclare
			command.symbolID, command.name = symbolCount, nameText
			command.exponentBits, command.significandBits = exponent, significand
		case "assert":
			relation, relationOK := scanner.formula(symbols[:symbolCount])
			if !relationOK || !scanner.right() {
				return nil, false
			}
			command = relation
			command.commandIndex = commandCount
		case "check-sat":
			if !scanner.right() {
				return nil, false
			}
			command.kind = fpFastCheck
		default:
			return nil, false
		}
		commands[commandCount] = command
		commandCount++
	}
	if commandCount == 0 {
		return nil, false
	}
	responses := make([]Response, 0, commandCount)
	solver := smt.New()
	nextAssertion := 1
	for index := 0; index < commandCount; index++ {
		command := commands[index]
		switch command.kind {
		case fpFastAcknowledge, fpFastDeclare:
			responses = append(responses, Acknowledged{CommandIndex: command.commandIndex})
		case fpFastAssign:
			relation := smt.BitVectorRelation{
				Width:    command.exponentBits + command.significandBits,
				SymbolID: command.symbolID, Value: command.value,
				Negated: command.negated,
			}
			solver = smt.Assert(nextAssertion, solver, relation)
			nextAssertion++
			responses = append(responses, Acknowledged{CommandIndex: command.commandIndex})
		case fpFastRound:
			relation := smt.NewFloatingPointRoundToIntegralRelation(
				command.exponentBits, command.significandBits,
				command.symbolID, command.mode, command.value,
			)
			relation.Negated = command.negated
			solver = smt.AssertFloatingPointRoundToIntegralRelation(
				nextAssertion, solver, relation,
			)
			nextAssertion++
			responses = append(responses, Acknowledged{CommandIndex: command.commandIndex})
		case fpFastCheck:
			switch result := smt.Check(solver).(type) {
			case smt.Satisfiable:
				responses = append(responses, Satisfiable{Model: result.Value})
			case smt.Unsatisfiable:
				responses = append(responses, Unsatisfiable{Proof: result.Value})
			case smt.Unknown:
				responses = append(responses, Unknown{
					Proof: result.Context, Reason: result.Reason,
				})
			}
		}
	}
	return Executed{Responses: responses}, true
}

func (scanner *fpFastScanner) formula(
	symbols []fpFastSymbol,
) (fpFastCommand, bool) {
	if !scanner.left() {
		return fpFastCommand{}, false
	}
	operator, ok := scanner.atom()
	if !ok {
		return fpFastCommand{}, false
	}
	if scanner.text(operator) == "not" {
		relation, relationOK := scanner.formula(symbols)
		if !relationOK || !scanner.right() {
			return fpFastCommand{}, false
		}
		relation.negated = !relation.negated
		return relation, true
	}
	if scanner.text(operator) != "=" {
		return fpFastCommand{}, false
	}
	left, leftOK := scanner.operand(symbols)
	right, rightOK := scanner.operand(symbols)
	if !leftOK || !rightOK || !scanner.right() {
		return fpFastCommand{}, false
	}
	derived, literal := left, right
	if derived.kind == fpFastLiteralBits {
		derived, literal = right, left
	}
	if literal.kind != fpFastLiteralBits ||
		derived.exponentBits+derived.significandBits != literal.value.Width() {
		return fpFastCommand{}, false
	}
	command := fpFastCommand{
		symbolID:        derived.symbolID,
		exponentBits:    derived.exponentBits,
		significandBits: derived.significandBits,
		value:           literal.value, mode: derived.mode,
	}
	switch derived.kind {
	case fpFastSymbolBits:
		command.kind = fpFastAssign
	case fpFastRoundedBits:
		command.kind = fpFastRound
	default:
		return fpFastCommand{}, false
	}
	return command, true
}

func (scanner *fpFastScanner) operand(
	symbols []fpFastSymbol,
) (fpFastOperand, bool) {
	snapshot := scanner.at
	if token, ok := scanner.next(); ok && token.kind == fpFastAtom {
		text := scanner.text(token)
		if strings.HasPrefix(text, "#x") && len(text) > 2 {
			width := 4 * (len(text) - 2)
			if width > 64 {
				return fpFastOperand{}, false
			}
			value, err := strconv.ParseUint(text[2:], 16, 64)
			if err != nil {
				return fpFastOperand{}, false
			}
			return fpFastOperand{
				kind:  fpFastLiteralBits,
				value: smt.NewBitVectorUint64(width, value),
			}, true
		}
		if strings.HasPrefix(text, "#b") && len(text) > 2 {
			width := len(text) - 2
			if width > 64 {
				return fpFastOperand{}, false
			}
			value, err := strconv.ParseUint(text[2:], 2, 64)
			if err != nil {
				return fpFastOperand{}, false
			}
			return fpFastOperand{
				kind:  fpFastLiteralBits,
				value: smt.NewBitVectorUint64(width, value),
			}, true
		}
	}
	scanner.at = snapshot
	if !scanner.left() {
		return fpFastOperand{}, false
	}
	operator, operatorOK := scanner.atom()
	if !operatorOK || scanner.text(operator) != "fp.to_ieee_bv" {
		return fpFastOperand{}, false
	}
	symbolSnapshot := scanner.at
	if symbol, ok := scanner.atom(); ok {
		found, foundOK := fpFastFindSymbol(scanner.text(symbol), symbols)
		if !foundOK || !scanner.right() {
			return fpFastOperand{}, false
		}
		return fpFastOperand{
			kind: fpFastSymbolBits, symbolID: found.id,
			exponentBits:    found.exponentBits,
			significandBits: found.significandBits,
		}, true
	}
	scanner.at = symbolSnapshot
	if !scanner.left() {
		return fpFastOperand{}, false
	}
	round, roundOK := scanner.atom()
	mode, modeOK := scanner.atom()
	symbol, symbolOK := scanner.atom()
	if !roundOK || !modeOK || !symbolOK ||
		scanner.text(round) != "fp.roundToIntegral" {
		return fpFastOperand{}, false
	}
	roundingMode, roundingModeOK := fpFastRoundingMode(scanner.text(mode))
	found, foundOK := fpFastFindSymbol(scanner.text(symbol), symbols)
	if !roundingModeOK || !foundOK || !scanner.right() || !scanner.right() {
		return fpFastOperand{}, false
	}
	return fpFastOperand{
		kind: fpFastRoundedBits, symbolID: found.id,
		exponentBits:    found.exponentBits,
		significandBits: found.significandBits,
		mode:            roundingMode,
	}, true
}

func fpFastFindSymbol(name string, symbols []fpFastSymbol) (fpFastSymbol, bool) {
	for _, symbol := range symbols {
		if symbol.name == name {
			return symbol, true
		}
	}
	return fpFastSymbol{}, false
}

func fpFastRoundingMode(
	name string,
) (smt.FloatingPointRoundingMode, bool) {
	switch name {
	case "RNE", "roundNearestTiesToEven":
		return smt.RoundNearestTiesToEven(), true
	case "RNA", "roundNearestTiesToAway":
		return smt.RoundNearestTiesToAway(), true
	case "RTP", "roundTowardPositive":
		return smt.RoundTowardPositive(), true
	case "RTN", "roundTowardNegative":
		return smt.RoundTowardNegative(), true
	case "RTZ", "roundTowardZero":
		return smt.RoundTowardZero(), true
	default:
		return nil, false
	}
}

func (scanner *fpFastScanner) next() (fpFastToken, bool) {
	for scanner.at < len(scanner.source) {
		switch scanner.source[scanner.at] {
		case ' ', '\t', '\n', '\r':
			scanner.at++
			continue
		case ';':
			for scanner.at < len(scanner.source) &&
				scanner.source[scanner.at] != '\n' {
				scanner.at++
			}
			continue
		}
		break
	}
	if scanner.at == len(scanner.source) {
		return fpFastToken{}, false
	}
	start := scanner.at
	switch scanner.source[scanner.at] {
	case '(':
		scanner.at++
		return fpFastToken{kind: fpFastLeft, start: start, end: scanner.at}, true
	case ')':
		scanner.at++
		return fpFastToken{kind: fpFastRight, start: start, end: scanner.at}, true
	}
	for scanner.at < len(scanner.source) {
		switch scanner.source[scanner.at] {
		case ' ', '\t', '\n', '\r', '(', ')', ';':
			return fpFastToken{kind: fpFastAtom, start: start, end: scanner.at}, true
		default:
			scanner.at++
		}
	}
	return fpFastToken{kind: fpFastAtom, start: start, end: scanner.at}, true
}

func (scanner *fpFastScanner) atom() (fpFastToken, bool) {
	token, ok := scanner.next()
	return token, ok && token.kind == fpFastAtom
}

func (scanner *fpFastScanner) left() bool {
	token, ok := scanner.next()
	return ok && token.kind == fpFastLeft
}

func (scanner *fpFastScanner) right() bool {
	token, ok := scanner.next()
	return ok && token.kind == fpFastRight
}

func (scanner *fpFastScanner) positiveInt() (int, bool) {
	token, ok := scanner.atom()
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(scanner.text(token))
	return value, err == nil && value > 0
}

func (scanner *fpFastScanner) text(token fpFastToken) string {
	return scanner.source[token.start:token.end]
}
