package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnyExistentialErasesNestedGenericFields(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/existentialany\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Box[T any] enum { Boxed(Value T) }
type Term[S any] enum {
	Equal[X any](Left Box[X], Right Box[X]) Term[bool]
}

func main() {
	_ = Equal(Boxed(1), Boxed(2))
}
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
		`//goplus:variant (Term[S]) Equal[X any](Left Box[X], Right Box[X]) Term[bool]`,
		"type Equal struct {",
		"Left  any",
		"Right any",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("generated output does not contain %q:\n%s", want, out)
		}
	}
}

func TestAnyExistentialStillUnifiesAuthoredFields(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/existentialany\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Box[T any] enum { Boxed(Value T) }
type Term[S any] enum { Equal[X any](Left Box[X], Right Box[X]) Term[bool] }

func main() { _ = Equal(Boxed(1), Boxed("wrong")) }
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok() {
		t.Fatal("mismatched existential fields unexpectedly accepted")
	}
}
