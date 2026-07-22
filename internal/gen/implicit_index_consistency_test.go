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

func main() {
	left := NewLeft(1)
	right := NewRight(2)
	Pair(left, right)
}
`)
	res, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok() {
		t.Fatal("inconsistent inferred index unexpectedly accepted")
	}
	found := false
	for _, diagnostic := range res.Diags {
		found = found || strings.Contains(diagnostic.Msg, "dependent index mismatch")
	}
	if !found {
		t.Fatalf("diagnostics do not explain mismatch: %+v", res.Diags)
	}
}
