package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossPackageImplicitIndexStillWrapsLinearArgument(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/implicitlinear\n\ngo 1.26.0\n")
	if err := os.MkdirAll(filepath.Join(dir, "cap"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRefinementTestFile(t, dir, "cap/cap.go", `package cap
import "sync/atomic"

//goplus:enum Token[m nat, r nat]
type Token interface{ token() }
type tokenValue struct{}
func (tokenValue) token() {}

//goplus:dep New() Token[7, 1]
func New() Token { return tokenValue{} }

//goplus:dep Move(0 m nat, 1 value Token[m, 1]) Token[m, 0]
func Move(value Lin[Token]) Token { return value.Use() }

type Lin[T any] struct { value T; used *atomic.Bool }
func LinOf[T any](value T) Lin[T] { return Lin[T]{value: value, used: new(atomic.Bool)} }
func (value Lin[T]) Use() T {
    if !value.used.CompareAndSwap(false, true) { panic("used twice") }
    return value.value
}
`)
	writeRefinementTestFile(t, dir, "main.gp", `package main
import "example.com/implicitlinear/cap"
func main() {
    value := cap.New()
    _ = cap.Move(value)
}
`)
	result, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("generation diagnostics: %+v", result.Diags)
	}
	generated, err := os.ReadFile(filepath.Join(dir, "main_gp.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "cap.Move(cap.LinOf(value))") {
		t.Fatalf("implicit-index linear call was not wrapped:\n%s", generated)
	}
}
