package lower

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestLowerTailCalls(t *testing.T) {
	src := []byte(`package p
//goplus:tail
func fact(n, acc int) int {
	if n == 0 { return acc }
	recur(n-1, acc*n)
}
`)
	out, errs := LowerTailCalls("fact.gp", src)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	text := string(out)
	for _, want := range []string{
		"__goplus_recur:\nfor {",
		"n, acc = n-1, acc*n",
		"continue __goplus_recur",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output does not contain %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "recur(") {
		t.Fatalf("recur survived lowering:\n%s", text)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fact.go", out, 0); err != nil {
		t.Fatalf("lowered output is not Go: %v\n%s", err, out)
	}
}

func TestLowerTailCallsInTailBranches(t *testing.T) {
	src := []byte(`package p
//goplus:tail
func step(n int) int {
	switch {
	case n < 0:
		return 0
	case n == 0:
		recur(1)
	default:
		if n > 10 { recur(n-1) } else { recur(n+1) }
	}
}
`)
	out, errs := LowerTailCalls("branches.gp", src)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	if got := strings.Count(string(out), "continue __goplus_recur"); got != 3 {
		t.Fatalf("continue count = %d, want 3:\n%s", got, out)
	}
}

func TestLowerTailCallsMethodKeepsReceiver(t *testing.T) {
	src := []byte(`package p
type counter int
//goplus:tail
func (c counter) add(n int) int { if n == 0 { return int(c) }; recur(n-1) }
`)
	out, errs := LowerTailCalls("scopes.gp", src)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	if got := strings.Count(string(out), "n = n-1"); got != 1 {
		t.Fatalf("parameter assignment count = %d, want 1:\n%s", got, out)
	}
	if strings.Contains(string(out), "c =") {
		t.Fatalf("method receiver was rebound:\n%s", out)
	}
}

func TestLowerTailCallsKeepsReceiverAfterMethodLowering(t *testing.T) {
	src := []byte(`package p
//goplus:tail receiver=c
func add(c counter, n int) int { if n == 0 { return int(c) }; recur(n-1) }
`)
	out, errs := LowerTailCalls("lowered_method.gp", src)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	if !strings.Contains(string(out), "n = n-1") || strings.Contains(string(out), "c, n =") {
		t.Fatalf("lowered receiver entered recurrence state:\n%s", out)
	}
}

func TestLowerTailCallsRejectsInvalidForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"non-tail", "package p;\n//goplus:tail\nfunc f(n int) int { recur(n-1); return n }", "final statement"},
		{"return-expression", "package p;\n//goplus:tail\nfunc f(n int) int { return recur(n-1) }", "final statement"},
		{"arity", "package p;\n//goplus:tail\nfunc f(n int) int { recur() }", "want 1"},
		{"unnamed", "package p;\n//goplus:tail\nfunc f(int) int { recur(1) }", "unnamed parameter"},
		{"blank", "package p;\n//goplus:tail\nfunc f(_ int) int { recur(1) }", "blank parameter"},
		{"ellipsis", "package p;\n//goplus:tail\nfunc f(xs ...int) { recur(xs...) }", "... is not permitted"},
		{"fallthrough", "package p;\n//goplus:tail\nfunc f(n int) int { if n > 0 { recur(n-1) } }", "fall through"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, errs := LowerTailCalls(test.name+".gp", []byte(test.src))
			if len(errs) == 0 || !strings.Contains(errs[0].Msg, test.want) {
				t.Fatalf("errors = %+v, want %q", errs, test.want)
			}
			if string(out) != test.src {
				t.Fatalf("invalid source was partially lowered:\n%s", out)
			}
		})
	}
}

func TestLowerTailCallsUsesFreshLabel(t *testing.T) {
	src := []byte("package p;\n//goplus:tail\nfunc f(__goplus_recur, n int) int { if n == 0 { return n }; recur(__goplus_recur, n-1) }")
	out, errs := LowerTailCalls("label.gp", src)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	if !strings.Contains(string(out), "__goplus_recur1:") {
		t.Fatalf("label was not made fresh:\n%s", out)
	}
}

func TestLowerTailCallsInSelectClause(t *testing.T) {
	src := []byte(`package p
//goplus:tail
func wait(ch chan int, n int) {
	select {
	case <-ch:
		recur(ch, n-1)
	default:
		return
	}
}
`)
	out, errs := LowerTailCalls("select.gp", src)
	if len(errs) != 0 || !strings.Contains(string(out), "continue __goplus_recur") {
		t.Fatalf("select lowering: errors=%+v\n%s", errs, out)
	}
}

func TestLowerTailCallsDoesNotTreatShadowedPanicAsTerminating(t *testing.T) {
	src := []byte(`package p
//goplus:tail
func f(n int) int {
	panic := func(string) {}
	if n > 0 { recur(n-1) }
	panic("not the builtin")
}
`)
	out, errs := LowerTailCalls("shadow.gp", src)
	if len(errs) != 1 || !strings.Contains(errs[0].Msg, "fall through") || string(out) != string(src) {
		t.Fatalf("shadowed panic proof: errors=%+v\n%s", errs, out)
	}
}

func TestLowerTailCallsPreservesOrdinaryGoRecur(t *testing.T) {
	src := []byte(`package p
func recur(int) int { return 1 }
func f(n int) int { recur(n); return recur(n-1) }
var g = func(n int) int { return recur(n) }
`)
	out, errs := LowerTailCalls("ordinary.go", src)
	if len(errs) != 0 || string(out) != string(src) {
		t.Fatalf("ordinary Go recur changed: errors=%+v\n%s", errs, out)
	}
}
