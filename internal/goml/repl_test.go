package goml

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// --- pure tests: no go tool, no compilation -------------------------------

func TestREPLClassify(t *testing.T) {
	cases := []struct {
		in   string
		want inputKind
	}{
		{"1 + 1", inputExpr},
		{"Double 21", inputExpr},
		{"let X := 42", inputDecl},
		{"let Twice (n : Int) : Int := n * 2", inputDecl},
		{"total let P (a : Nat) : Nat := a", inputDecl},
		{"type Color := | Red | Green", inputDecl},
		{"class Magma (t : Type) where", inputDecl},
		{"instance I : Magma Int where", inputDecl},
		{"@[tail]\nlet rec L () : Int := L ()", inputDecl},
		{`import "fmt"`, inputImport},
		{"module other", inputModule},
		{"-- a comment\n1 + 1", inputExpr},
	}
	for _, tc := range cases {
		if got := classify(tc.in); got != tc.want {
			t.Errorf("classify(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestREPLEofPos(t *testing.T) {
	cases := []struct {
		src  string
		want Pos
	}{
		{"", Pos{1, 1}},
		{"ab", Pos{1, 3}},
		{"ab\n", Pos{2, 1}},
		{"ab\ncd", Pos{2, 3}},
	}
	for _, tc := range cases {
		if got := eofPos([]byte(tc.src)); got != tc.want {
			t.Errorf("eofPos(%q) = %v, want %v", tc.src, got, tc.want)
		}
	}
}

func TestREPLIncomplete(t *testing.T) {
	r := &repl{}
	incomplete := []string{
		"type Color :=",
		"let Twice (n : Int) : Int :=",
		`let S := "unterminated`,
		"let X := (* open comment",
	}
	for _, in := range incomplete {
		if !r.incomplete(in) {
			t.Errorf("incomplete(%q) = false, want true", in)
		}
	}
	complete := []string{
		"1 + 1",
		"let X := 42",
		"type Color :=\n  | Red\n  | Green",
	}
	for _, in := range complete {
		if r.incomplete(in) {
			t.Errorf("incomplete(%q) = true, want false", in)
		}
	}
	// A genuine syntax error is not incompleteness.
	if r.incomplete("let X := match y with | 0 => 1") {
		t.Error("a literal-pattern error was treated as incomplete")
	}
}

func TestREPLSessionRenderAndImports(t *testing.T) {
	s := session{}
	s.addImport(sessionImport{path: `"fmt"`})
	s.addImport(sessionImport{path: `"strings"`})
	s.addDecl(sessionDecl{name: "Twice", kind: kLet, src: "let Twice (n : Int) : Int := n * 2"})
	s.addDecl(sessionDecl{name: "I", kind: kInstance, src: "instance I : Magma Int where\n  Combine a b := a + b"})

	got := s.render("main", "let _v := fmt.Sprint 1")
	if !strings.Contains(got, "module main") {
		t.Fatalf("missing module clause:\n%s", got)
	}
	if !strings.Contains(got, `import "fmt"`) {
		t.Fatalf("used import was pruned:\n%s", got)
	}
	if strings.Contains(got, `import "strings"`) {
		t.Fatalf("unused import was kept:\n%s", got)
	}
	// Law generation would import rapid, which the session module lacks.
	if !strings.Contains(got, "@[laws off]\ninstance I") {
		t.Fatalf("instance is missing @[laws off]:\n%s", got)
	}
	if !strings.HasSuffix(strings.TrimRight(got, "\n"), "let _v := fmt.Sprint 1") {
		t.Fatalf("the current input must render last:\n%s", got)
	}
}

func TestREPLSessionReplaceAndDrop(t *testing.T) {
	s := session{}
	s.addDecl(sessionDecl{name: "A", kind: kLet, src: "let A := 1"})
	s.addDecl(sessionDecl{name: "B", kind: kLet, src: "let B := 2"})
	s.addDecl(sessionDecl{name: "A", kind: kLet, src: "let A := 3"})
	if len(s.decls) != 2 {
		t.Fatalf("rebinding should replace, got %d decls", len(s.decls))
	}
	// The replacement moves last so it can use everything before it.
	if s.decls[len(s.decls)-1].name != "A" {
		t.Fatalf("replacement did not move to the end: %+v", s.decls)
	}
	if !s.dropDecl("A") || s.dropDecl("A") {
		t.Fatal("dropDecl should report whether it removed anything")
	}
}

func TestREPLEffectFlagging(t *testing.T) {
	effectful := []string{`fmt.Println "hi"`, `do { println "x" }`, `parse s ?`}
	for _, src := range effectful {
		file, err := Parse("t.goml", []byte("module m\nlet X := "+src+"\n"))
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		if !looksEffectful(file.Decls[0].(*LetDecl).Body) {
			t.Errorf("looksEffectful(%q) = false, want true", src)
		}
	}
	file, err := Parse("t.goml", []byte("module m\nlet X := 1 + 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if looksEffectful(file.Decls[0].(*LetDecl).Body) {
		t.Error("pure arithmetic was flagged as effectful")
	}
}

func TestREPLSubstituteIt(t *testing.T) {
	cases := []struct{ expr, want string }{
		{"it + 1", "(42) + 1"},
		{"Twice it", "Twice (42)"},
		{"it", "(42)"},
		{"x.it + it", "x.it + (42)"}, // a selector member named it is untouched
		{"1 + 2", "1 + 2"},
	}
	for _, tc := range cases {
		if got := substituteIt(tc.expr, "(42)"); got != tc.want {
			t.Errorf("substituteIt(%q) = %q, want %q", tc.expr, got, tc.want)
		}
	}
}

func TestREPLCommandsWithoutCompiling(t *testing.T) {
	out, errOut := runREPLScript(t, ":help\n:list\n:dir\n:nope\n:quit\n")
	if !strings.Contains(out, ":type <expr>") {
		t.Errorf(":help did not list commands:\n%s", out)
	}
	if !strings.Contains(out, "re-execute on every evaluation") {
		t.Errorf(":help must state the replay caveat:\n%s", out)
	}
	if !strings.Contains(out, "(empty session)") {
		t.Errorf(":list on an empty session:\n%s", out)
	}
	if !strings.Contains(errOut, `unknown command ":nope"`) {
		t.Errorf("unknown command not reported:\n%s", errOut)
	}
}

func TestREPLRejectsModuleInput(t *testing.T) {
	_, errOut := runREPLScript(t, "module other\n:quit\n")
	if !strings.Contains(errOut, "module is fixed") {
		t.Errorf("module input should be refused:\n%s", errOut)
	}
}

func TestREPLParseErrorIsPositioned(t *testing.T) {
	_, errOut := runREPLScript(t, "let F := match y with | 0 => 1\n:quit\n")
	if !strings.Contains(errOut, "<stdin>:1:") {
		t.Errorf("parse error not positioned at the user's line:\n%s", errOut)
	}
	if !strings.Contains(errOut, "^") {
		t.Errorf("parse error missing its caret:\n%s", errOut)
	}
}

// --- compile-and-run tests ------------------------------------------------

func TestREPLEvaluates(t *testing.T) {
	requireGo(t)
	out, errOut := runREPLScript(t, strings.Join([]string{
		"1 + 1",
		"let Double (n : Int) : Int := n * 2",
		"Double 21",
		"let Answer := 42",
		"Answer + 1",
		":quit",
	}, "\n")+"\n")
	for _, want := range []string{"2", "42", "43"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s\nstderr:\n%s", want, out, errOut)
		}
	}
}

func TestREPLValueBindingsCompose(t *testing.T) {
	requireGo(t)
	out, _ := runREPLScript(t, "let Port := 8080\nlet Next : Int := Port + 1\nNext\n:quit\n")
	if !strings.Contains(out, "8081") {
		t.Fatalf("value bindings did not compose:\n%s", out)
	}
}

func TestREPLEffectfulExpressionRunsOnce(t *testing.T) {
	requireGo(t)
	out, _ := runREPLScript(t, "fmt.Println \"once\"\n1 + 1\n:quit\n")
	if n := strings.Count(out, "once"); n != 1 {
		t.Fatalf("effectful expression ran %d times, want 1:\n%s", n, out)
	}
}

func TestREPLBrokenInputLeavesSessionIntact(t *testing.T) {
	requireGo(t)
	out, errOut := runREPLScript(t, "let A := 1\nlet Bad : Int := \"nope\"\nA + 1\n:quit\n")
	if !strings.Contains(errOut, "cannot use") {
		t.Fatalf("expected a type error:\n%s", errOut)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("session did not survive the failure:\n%s\nstderr:\n%s", out, errOut)
	}
}

func TestREPLReportsOlderBinding(t *testing.T) {
	requireGo(t)
	_, errOut := runREPLScript(t, "let A := 1\nlet Uses : Int := A + 1\nlet A := \"text\"\n:quit\n")
	if !strings.Contains(errOut, "in Uses (defined earlier)") {
		t.Fatalf("a break in an older binding was not attributed:\n%s", errOut)
	}
}

func TestREPLTypeReportsErasedGoType(t *testing.T) {
	requireGo(t)
	out, errOut := runREPLScript(t, "let Twice (n : Int) : Int := n * 2\n:type Twice\n:quit\n")
	if !strings.Contains(out, "func(n int) int") {
		t.Fatalf(":type output:\n%s\nstderr:\n%s", out, errOut)
	}
	if !strings.Contains(out, "erased") {
		t.Fatalf(":type must state the erasure caveat:\n%s", out)
	}
}

func TestREPLMultiLineDeclaration(t *testing.T) {
	requireGo(t)
	script := "type Color :=\n  | Red\n  | Green\n\nlet Name (c : Color) : String :=\n  match c with\n  | Red => \"red\"\n  | Green => \"green\"\n\nName Green\n:quit\n"
	out, errOut := runREPLScript(t, script)
	if !strings.Contains(out, "green") {
		t.Fatalf("multi-line declarations did not evaluate:\n%s\nstderr:\n%s", out, errOut)
	}
}

func TestREPLSaveProducesLoadableModule(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	path := dir + "/saved.goml"
	out, errOut := runREPLScript(t, "let Twice (n : Int) : Int := n * 2\n:save "+path+"\n:quit\n")
	if !strings.Contains(out, "wrote ") {
		t.Fatalf(":save did not report:\n%s\n%s", out, errOut)
	}
	res, err := Run(RunOptions{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Gen.Diags {
		t.Errorf("saved session does not generate: %s", d)
	}
}

// --- helpers --------------------------------------------------------------

func runREPLScript(t *testing.T, script string) (string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	REPL(strings.NewReader(script), &out, &errOut, REPLOptions{Dir: t.TempDir()})
	return out.String(), errOut.String()
}

func requireGo(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("compiling REPL test: -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("compiling REPL test: no go tool")
	}
}
