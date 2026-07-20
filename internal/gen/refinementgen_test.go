package gen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefinementGeneratedGoAndCallContract(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/refinementtest\n\ngo 1.24\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Positive refine(value int) { value > 0 }

func Echo(value Positive) Positive { return value }
func main() { _ = Echo(Positive(1)) }
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok() {
		t.Fatalf("generation diagnostics: %+v", res.Diags)
	}
	out, err := os.ReadFile(filepath.Join(dir, "main_gp.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`//goplus:refinement "Positive" "value" "int" "value > 0"`,
		"func __goplus_refinement_Positive(value int) bool",
		"if !__goplus_refinement_Positive(value)",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("generated output does not contain %q:\n%s", want, out)
		}
	}
}

func TestRefinementRejectsUnprovedFunctionArgument(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/refinementtest\n\ngo 1.24\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Positive refine(value int) { value > 0 }

func Need(value Positive) int { return value }
func unchecked(value int) int { return Need(value) }
func main() {}
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok() {
		t.Fatal("generation unexpectedly accepted an unproved refined argument")
	}
	if got := res.Diags[0].Msg; !strings.Contains(got, "cannot prove value > 0 for argument 1 to Need") {
		t.Fatalf("diagnostic = %q", got)
	}
}

func TestRefinementContractCrossesPackageBoundary(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/refinementtest\n\ngo 1.24\n")
	if err := os.Mkdir(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRefinementTestFile(t, dir, "lib/ref.gp", `package lib

type Positive refine(value int) { value > 0 }
func Need(value Positive) int { return value }
`)
	writeRefinementTestFile(t, dir, "main.gp", `package main

import "example.com/refinementtest/lib"
func unchecked(value int) int { return lib.Need(value) }
func main() {}
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"./..."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok() {
		t.Fatal("generation unexpectedly lost a cross-package refinement contract")
	}
	found := false
	for _, d := range res.Diags {
		found = found || strings.Contains(d.Msg, "cannot prove value > 0 for argument 1 to Need")
	}
	if !found {
		t.Fatalf("diagnostics: %+v", res.Diags)
	}
}

func TestCompositeGADTEmitsPortableErasedView(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/gadttest\n\ngo 1.24\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Expr[T any] enum {
	Plain(value T)
	Wrap(inner Expr[T]) Expr[[]T]
}
func main() { _ = Wrap[int](Plain(1)) }
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok() {
		for path, text := range res.Outputs {
			t.Logf("%s:\n%s", path, text)
		}
		t.Fatalf("generation diagnostics: %+v", res.Diags)
	}
	res, err = Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil || !res.Ok() {
		t.Fatalf("write generation: %v, %+v", err, res.Diags)
	}
	out, err := os.ReadFile(filepath.Join(dir, "main_gp.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"func (v Wrap[T]) __goplus_view_Expr() (int, []any)",
		"func GoplusExprView[T any](value Expr[T]) (int, []any)",
		"[]any{v.Inner}",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("generated output does not contain %q:\n%s", want, out)
		}
	}
}

func TestCompositeGADTEliminatesThroughGenericScrutinee(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/gadttest\n\ngo 1.24\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

import "fmt"

type Expr[T any] enum {
	Plain(value T)
	Wrap(inner Expr[T]) Expr[[]T]
}

func describe[U any](e Expr[U]) string {
	match e {
	case Plain(value):
		return fmt.Sprint(value)
	case Wrap(inner):
		_ = inner
		return "wrap"
	}
}

func main() {
	fmt.Println(describe(Plain(1)))
	fmt.Println(describe[[]int](Wrap[int](Plain(2))))
}
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok() {
		t.Fatalf("generation diagnostics: %+v", res.Diags)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated package does not run: %v\n%s", err, out)
	}
	if string(out) != "1\nwrap\n" {
		t.Fatalf("output = %q", out)
	}
}

func TestCompositeGADTRecursiveErasedElimination(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/gadttest\n\ngo 1.24\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

import "fmt"

type Expr[T any] enum {
	Plain(value T)
	Wrap(inner Expr[T]) Expr[[]T]
}

func depth[U any](e Expr[U]) int {
	match e {
	case Plain(_):
		return 0
	case Wrap(inner):
		return 1 + depth(inner)
	}
}

func main() {
	v := Wrap[[]int](Wrap[int](Plain(2)))
	fmt.Println(depth(v))
}
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok() {
		t.Fatalf("generation diagnostics: %+v", res.Diags)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated package does not run: %v\n%s", err, out)
	}
	if string(out) != "2\n" {
		t.Fatalf("output = %q", out)
	}
}

func writeRefinementTestFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
