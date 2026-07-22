package gen

import (
	"strings"
	"testing"
)

func TestImplicitIndexMustUnifyAcrossRuntimeArguments(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/implicitindex\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Left[n nat] enum { LeftAt(ID int) Left[n] }
type Right[n nat] enum { RightAt(ID int) Right[n] }
func NewLeft(id nat) Left[id] { return LeftAt(int(id)) }
func NewRight(id nat) Right[id] { return RightAt(int(id)) }
func Pair(0 n nat, left Left[n], right Right[n]) {}
func main() { left := NewLeft(1); right := NewRight(2); Pair(left, right) }
`)
	assertDependentMismatch(t, dir)
}

func TestImplicitIndexMustUnifyAcrossVariadicArguments(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/implicitvariadic\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Item[n nat] enum { ItemAt(ID int) Item[n] }
func NewItem(id nat) Item[id] { return ItemAt(int(id)) }
func All(0 n nat, values ...Item[n]) Item[n] { return values[0] }
func main() { a := NewItem(1); b := NewItem(2); _ = All(a, b) }
`)
	assertDependentMismatch(t, dir)
}

func TestImplicitIndexInfersVariadicResult(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/implicitvariadicok\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Item[n nat] enum { ItemAt(ID int) Item[n] }
func NewItem(id nat) Item[id] { return ItemAt(int(id)) }
func All(0 n nat, values ...Item[n]) Item[n] { return values[0] }
func Need(0 n nat, value Item[n]) {}
func main() { a := NewItem(1); b := NewItem(1); Need(All(a, b)) }
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok() {
		t.Fatalf("valid variadic index program rejected: %+v", res.Diags)
	}
}

func TestDependentResultMustMatchExplicitBindingType(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/dependentbinding\n\ngo 1.26.0\n")
	writeRefinementTestFile(t, dir, "main.gp", `package main

type Vec[n nat] enum { Vector() Vec[n] }
func Make(size nat) Vec[size] { return Vector() }
func main() { var value Vec[16] = Make(8); _ = value }
`)
	assertDependentMismatch(t, dir)
}

func assertDependentMismatch(t *testing.T, dir string) {
	t.Helper()
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok() {
		t.Fatal("inconsistent inferred index unexpectedly accepted")
	}
	for _, diagnostic := range res.Diags {
		if strings.Contains(diagnostic.Msg, "dependent index mismatch") {
			return
		}
	}
	t.Fatalf("diagnostics do not explain mismatch: %+v", res.Diags)
}
