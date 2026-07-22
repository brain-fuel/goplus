package smtlib

import "testing"

func TestParseCoreScript(t *testing.T) {
	source := `; difference logic
(set-logic QF_IDL)
(set-option :produce-models true)
(declare-const |x value| Int)
(declare-fun p (Int) Bool)
(assert (<= (- |x value| 1) 3))
(push 1)
(check-sat-assuming ((p |x value|)))
(pop 1)
(get-value (|x value|))
(get-model)
(exit)`
	result, ok := Parse(source).(Parsed)
	if !ok {
		t.Fatalf("result=%T", Parse(source))
	}
	if len(result.Commands) != 11 {
		t.Fatalf("commands=%d", len(result.Commands))
	}
	if command, ok := result.Commands[0].(SetLogic); !ok || command.Name != "QF_IDL" {
		t.Fatalf("command[0]=%#v", result.Commands[0])
	}
	if declaration, ok := result.Commands[2].(DeclareConst); !ok || declaration.Name != "x value" {
		t.Fatalf("command[2]=%#v", result.Commands[2])
	}
	if _, ok := result.Commands[7].(Pop); !ok {
		t.Fatalf("command[7]=%T", result.Commands[7])
	}
}

func TestParsePreservesUnknownCommand(t *testing.T) {
	result := Parse(`(future-command :mode "a""b" 123456789012345678901234567890 12.340)`).(Parsed)
	command, ok := result.Commands[0].(RawCommand)
	if !ok || command.Name != "future-command" || len(command.Arguments) != 4 {
		t.Fatalf("command=%#v", result.Commands[0])
	}
	value := command.Arguments[1].(Atom)
	if value.Text != `a"b` {
		t.Fatalf("text=%q", value.Text)
	}
	if _, ok := command.Arguments[2].(Atom).Kind.(NumeralAtom); !ok {
		t.Fatalf("large numeral kind=%T", command.Arguments[2].(Atom).Kind)
	}
	if _, ok := command.Arguments[3].(Atom).Kind.(DecimalAtom); !ok {
		t.Fatalf("decimal kind=%T", command.Arguments[3].(Atom).Kind)
	}
}

func TestParseReportsStructuralErrors(t *testing.T) {
	for _, source := range []string{`(assert true`, `)`, `"unterminated`} {
		if result := Parse(source); func() bool { _, ok := result.(Rejected); return ok }() == false {
			t.Fatalf("source %q result=%T", source, result)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	source := `(set-logic QF_IDL)
(declare-sort U 0)
(declare-const |x value| Int)
(assert (<= |x value| 123456789012345678901234567890))
(future-command :label "a""b")
(check-sat)`
	first := Parse(source).(Parsed)
	formatted := Format(first.Commands)
	second, ok := Parse(formatted).(Parsed)
	if !ok || len(second.Commands) != len(first.Commands) {
		t.Fatalf("formatted=%q result=%#v", formatted, second)
	}
	declaration := second.Commands[2].(DeclareConst)
	if declaration.Name != "x value" {
		t.Fatalf("name=%q", declaration.Name)
	}
	unknown := second.Commands[4].(RawCommand)
	if unknown.Arguments[1].(Atom).Text != `a"b` {
		t.Fatalf("string=%q", unknown.Arguments[1].(Atom).Text)
	}
}

var benchmarkParseResult ParseResult

func BenchmarkParseSMTLib(b *testing.B) {
	const source = `(set-logic QF_IDL)
(set-option :produce-models true)
(declare-const x Int)
(declare-const y Int)
(assert (<= (- x y) 3))
(assert (<= y 2))
(assert (<= 4 x))
(check-sat)
(get-value (x y))`
	b.ReportAllocs()
	for b.Loop() {
		benchmarkParseResult = Parse(source)
	}
}
