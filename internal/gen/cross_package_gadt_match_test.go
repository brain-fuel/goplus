package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func crossPackageGADTMatchModule(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/gadtmatch\n\ngo 1.26.0\n")
	if err := os.MkdirAll(filepath.Join(dir, "term"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRefinementTestFile(t, dir, "term/term.gp", `package term

type Flag struct{}
type Other struct{}

type Term[S any] enum {
	Fixed(Value int) Term[Flag]
	Different(Value string) Term[Other]
}
`)
	writeRefinementTestFile(t, dir, "main.gp", body)
	return dir
}

func TestCrossPackageGADTMatchAcceptsQualifiedFixedResult(t *testing.T) {
	dir := crossPackageGADTMatchModule(t, `package main

import terms "example.com/gadtmatch/term"

func value(term terms.Term[terms.Flag]) int {
	match term {
	case terms.Fixed(value): return value
	case _: panic("impossible")
	}
}

func main() { _ = value(terms.Fixed(1)) }
`)
	result, err := Run(Options{Dir: dir, Patterns: []string{"./..."}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("generation diagnostics: %+v", result.Diags)
	}
}

func TestCrossPackageGADTMatchStillRejectsImpossibleQualifiedResult(t *testing.T) {
	dir := crossPackageGADTMatchModule(t, `package main

import terms "example.com/gadtmatch/term"

func value(term terms.Term[terms.Flag]) int {
	match term {
	case terms.Different(_): return 0
	case terms.Fixed(value): return value
	}
}

func main() {}
`)
	result, err := Run(Options{Dir: dir, Patterns: []string{"./..."}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok() {
		t.Fatal("impossible cross-package GADT arm unexpectedly compiled")
	}
	for _, diagnostic := range result.Diags {
		if strings.Contains(diagnostic.Msg, "can never match") {
			return
		}
	}
	t.Fatalf("diagnostics do not explain impossible arm: %+v", result.Diags)
}
