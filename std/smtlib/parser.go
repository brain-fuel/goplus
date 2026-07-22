package smtlib

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	tokenAtom = iota
	tokenString
	tokenLeft
	tokenRight
)

type token struct {
	kind       int
	text       string
	start, end int
}

type scriptParser struct {
	tokens []token
	at     int
	errors []ParseError
}

func parseSMTLib(source string) ([]Command, []ParseError) {
	tokens, lexicalErrors := lexSMTLib(source)
	parser := scriptParser{tokens: tokens, errors: lexicalErrors}
	var expressions []SExpr
	for parser.at < len(parser.tokens) {
		if expression, ok := parser.expression(); ok {
			expressions = append(expressions, expression)
		}
	}
	commands := make([]Command, 0, len(expressions))
	for _, expression := range expressions {
		command, ok := commandFromExpression(expression, &parser.errors)
		if ok {
			commands = append(commands, command)
		}
	}
	return commands, parser.errors
}

func lexSMTLib(source string) ([]token, []ParseError) {
	var tokens []token
	var errors []ParseError
	for offset := 0; offset < len(source); {
		r, size := utf8.DecodeRuneInString(source[offset:])
		if unicode.IsSpace(r) {
			offset += size
			continue
		}
		if r == ';' {
			for offset < len(source) && source[offset] != '\n' {
				offset++
			}
			continue
		}
		if r == '(' || r == ')' {
			kind := tokenLeft
			if r == ')' {
				kind = tokenRight
			}
			tokens = append(tokens, token{kind: kind, text: string(r), start: offset, end: offset + size})
			offset += size
			continue
		}
		if r == '"' {
			start := offset
			offset += size
			var value strings.Builder
			closed := false
			for offset < len(source) {
				next, nextSize := utf8.DecodeRuneInString(source[offset:])
				if next == '"' {
					if offset+nextSize < len(source) {
						after, afterSize := utf8.DecodeRuneInString(source[offset+nextSize:])
						if after == '"' {
							value.WriteRune('"')
							offset += nextSize + afterSize
							continue
						}
					}
					offset += nextSize
					closed = true
					break
				}
				value.WriteRune(next)
				offset += nextSize
			}
			if !closed {
				errors = append(errors, ParseError{Message: "unterminated string literal", At: Span{Start: start, End: len(source)}})
			}
			tokens = append(tokens, token{kind: tokenString, text: value.String(), start: start, end: offset})
			continue
		}
		if r == '|' {
			start := offset
			offset += size
			content := offset
			for offset < len(source) && source[offset] != '|' {
				_, nextSize := utf8.DecodeRuneInString(source[offset:])
				offset += nextSize
			}
			if offset == len(source) {
				errors = append(errors, ParseError{Message: "unterminated quoted symbol", At: Span{Start: start, End: offset}})
				tokens = append(tokens, token{kind: tokenAtom, text: source[content:offset], start: start, end: offset})
				continue
			}
			tokens = append(tokens, token{kind: tokenAtom, text: source[content:offset], start: start, end: offset + 1})
			offset++
			continue
		}
		start := offset
		for offset < len(source) {
			next, nextSize := utf8.DecodeRuneInString(source[offset:])
			if unicode.IsSpace(next) || next == '(' || next == ')' || next == ';' {
				break
			}
			offset += nextSize
		}
		tokens = append(tokens, token{kind: tokenAtom, text: source[start:offset], start: start, end: offset})
	}
	return tokens, errors
}

func (parser *scriptParser) expression() (SExpr, bool) {
	if parser.at >= len(parser.tokens) {
		return nil, false
	}
	current := parser.tokens[parser.at]
	parser.at++
	switch current.kind {
	case tokenRight:
		parser.errors = append(parser.errors, ParseError{Message: "unexpected ')'", At: spanOf(current)})
		return nil, false
	case tokenLeft:
		values := make([]SExpr, 0, 4)
		for parser.at < len(parser.tokens) && parser.tokens[parser.at].kind != tokenRight {
			value, ok := parser.expression()
			if ok {
				values = append(values, value)
			}
		}
		end := current.end
		if parser.at == len(parser.tokens) {
			parser.errors = append(parser.errors, ParseError{Message: "unterminated list", At: Span{Start: current.start, End: end}})
		} else {
			end = parser.tokens[parser.at].end
			parser.at++
		}
		return List{Values: values, At: Span{Start: current.start, End: end}}, true
	default:
		kind := classifyAtom(current)
		return Atom{Kind: kind, Text: current.text, At: spanOf(current)}, true
	}
}

func classifyAtom(value token) AtomKind {
	if value.kind == tokenString {
		return StringAtom{}
	}
	if strings.HasPrefix(value.text, ":") {
		return KeywordAtom{}
	}
	if isDigits(value.text) {
		return NumeralAtom{}
	}
	if before, after, ok := strings.Cut(value.text, "."); ok && isDigits(before) && isDigits(after) {
		return DecimalAtom{}
	}
	return SymbolAtom{}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func commandFromExpression(expression SExpr, errors *[]ParseError) (Command, bool) {
	list, ok := expression.(List)
	if !ok || len(list.Values) == 0 {
		*errors = append(*errors, ParseError{Message: "top-level SMT-LIB form must be a nonempty list", At: expressionSpan(expression)})
		return nil, false
	}
	head, ok := list.Values[0].(Atom)
	if !ok {
		*errors = append(*errors, ParseError{Message: "command name must be a symbol", At: expressionSpan(list.Values[0])})
		return nil, false
	}
	arguments := list.Values[1:]
	require := func(count int) bool {
		if len(arguments) == count {
			return true
		}
		*errors = append(*errors, ParseError{Message: "command " + head.Text + " has invalid arity", At: list.At})
		return false
	}
	symbol := func(value SExpr) (string, bool) {
		atom, ok := value.(Atom)
		return atom.Text, ok
	}
	switch head.Text {
	case "set-logic":
		if require(1) {
			if name, ok := symbol(arguments[0]); ok {
				return SetLogic{Name: name, At: list.At}, true
			}
		}
	case "set-option":
		if require(2) {
			if name, ok := symbol(arguments[0]); ok {
				return SetOption{Name: name, Value: arguments[1], At: list.At}, true
			}
		}
	case "declare-const":
		if require(2) {
			if name, ok := symbol(arguments[0]); ok {
				return DeclareConst{Name: name, Sort: arguments[1], At: list.At}, true
			}
		}
	case "declare-sort":
		if require(2) {
			name, nameOK := symbol(arguments[0])
			arity, arityOK := numeral(arguments[1])
			if nameOK && arityOK {
				return DeclareSort{Name: name, Arity: arity, At: list.At}, true
			}
		}
	case "declare-fun":
		if require(3) {
			name, nameOK := symbol(arguments[0])
			domain, domainOK := arguments[1].(List)
			if nameOK && domainOK {
				return DeclareFun{Name: name, Domain: domain.Values, Range: arguments[2], At: list.At}, true
			}
		}
	case "assert":
		if require(1) {
			return Assert{Term: arguments[0], At: list.At}, true
		}
	case "check-sat":
		if require(0) {
			return CheckSat{At: list.At}, true
		}
	case "check-sat-assuming":
		if require(1) {
			if values, ok := arguments[0].(List); ok {
				return CheckSatAssuming{Assumptions: values.Values, At: list.At}, true
			}
		}
	case "push":
		if require(1) {
			if count, ok := numeral(arguments[0]); ok {
				return Push{Levels: count, At: list.At}, true
			}
		}
	case "pop":
		if require(1) {
			if count, ok := numeral(arguments[0]); ok {
				return Pop{Levels: count, At: list.At}, true
			}
		}
	case "get-model":
		if require(0) {
			return GetModel{At: list.At}, true
		}
	case "get-value":
		if require(1) {
			if values, ok := arguments[0].(List); ok {
				return GetValue{Terms: values.Values, At: list.At}, true
			}
		}
	case "exit":
		if require(0) {
			return Exit{At: list.At}, true
		}
	default:
		return RawCommand{Name: head.Text, Arguments: append([]SExpr(nil), arguments...), At: list.At}, true
	}
	*errors = append(*errors, ParseError{Message: "malformed command " + head.Text, At: list.At})
	return nil, false
}

func numeral(expression SExpr) (int, bool) {
	atom, ok := expression.(Atom)
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(atom.Text)
	return value, err == nil && value >= 0
}

func expressionSpan(expression SExpr) Span {
	switch value := expression.(type) {
	case Atom:
		return value.At
	case List:
		return value.At
	default:
		return Span{}
	}
}

func spanOf(value token) Span { return Span{Start: value.start, End: value.end} }

func formatCommands(commands []Command) string {
	var output strings.Builder
	for index, command := range commands {
		if index != 0 {
			output.WriteByte('\n')
		}
		formatCommand(&output, command)
	}
	return output.String()
}

func formatCommand(output *strings.Builder, command Command) {
	output.WriteByte('(')
	switch value := command.(type) {
	case SetLogic:
		output.WriteString("set-logic ")
		writeSymbol(output, value.Name)
	case SetOption:
		output.WriteString("set-option ")
		output.WriteString(value.Name)
		output.WriteByte(' ')
		writeExpression(output, value.Value)
	case DeclareSort:
		output.WriteString("declare-sort ")
		writeSymbol(output, value.Name)
		output.WriteByte(' ')
		output.WriteString(strconv.Itoa(value.Arity))
	case DeclareConst:
		output.WriteString("declare-const ")
		writeSymbol(output, value.Name)
		output.WriteByte(' ')
		writeExpression(output, value.Sort)
	case DeclareFun:
		output.WriteString("declare-fun ")
		writeSymbol(output, value.Name)
		output.WriteString(" (")
		writeExpressions(output, value.Domain)
		output.WriteString(") ")
		writeExpression(output, value.Range)
	case Assert:
		output.WriteString("assert ")
		writeExpression(output, value.Term)
	case CheckSat:
		output.WriteString("check-sat")
	case CheckSatAssuming:
		output.WriteString("check-sat-assuming (")
		writeExpressions(output, value.Assumptions)
		output.WriteByte(')')
	case Push:
		output.WriteString("push ")
		output.WriteString(strconv.Itoa(value.Levels))
	case Pop:
		output.WriteString("pop ")
		output.WriteString(strconv.Itoa(value.Levels))
	case GetModel:
		output.WriteString("get-model")
	case GetValue:
		output.WriteString("get-value (")
		writeExpressions(output, value.Terms)
		output.WriteByte(')')
	case Exit:
		output.WriteString("exit")
	case RawCommand:
		writeSymbol(output, value.Name)
		if len(value.Arguments) != 0 {
			output.WriteByte(' ')
			writeExpressions(output, value.Arguments)
		}
	default:
		panic("smtlib: impossible command variant")
	}
	output.WriteByte(')')
}

func formatExpression(expression SExpr) string {
	var output strings.Builder
	writeExpression(&output, expression)
	return output.String()
}

func writeExpression(output *strings.Builder, expression SExpr) {
	switch value := expression.(type) {
	case Atom:
		switch value.Kind.(type) {
		case StringAtom:
			output.WriteByte('"')
			output.WriteString(strings.ReplaceAll(value.Text, `"`, `""`))
			output.WriteByte('"')
		case SymbolAtom:
			writeSymbol(output, value.Text)
		default:
			output.WriteString(value.Text)
		}
	case List:
		output.WriteByte('(')
		writeExpressions(output, value.Values)
		output.WriteByte(')')
	default:
		panic("smtlib: impossible expression variant")
	}
}

func writeExpressions(output *strings.Builder, expressions []SExpr) {
	for index, expression := range expressions {
		if index != 0 {
			output.WriteByte(' ')
		}
		writeExpression(output, expression)
	}
}

func writeSymbol(output *strings.Builder, symbol string) {
	if isSimpleSymbol(symbol) {
		output.WriteString(symbol)
		return
	}
	output.WriteByte('|')
	output.WriteString(symbol)
	output.WriteByte('|')
}

func isSimpleSymbol(symbol string) bool {
	if symbol == "" || strings.HasPrefix(symbol, ":") || isDigits(symbol) {
		return false
	}
	if before, after, ok := strings.Cut(symbol, "."); ok && isDigits(before) && isDigits(after) {
		return false
	}
	for _, character := range symbol {
		if unicode.IsSpace(character) || strings.ContainsRune(`();"|`, character) {
			return false
		}
	}
	return true
}
