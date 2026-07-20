package parser_test

import (
	"go/token"
	"testing"

	forkparser "goforge.dev/goplus/internal/syntax/parser"
)

func TestRefinementDeclaration(t *testing.T) {
	src := []byte("package p\n\ntype Positive refine(value int) {\n\tvalue > 0\n}\n")
	_, ext, err := forkparser.ParseFileExt(token.NewFileSet(), "p.gp", src, forkparser.ParseComments|forkparser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	if len(ext.Refinements) != 1 {
		t.Fatalf("got %d refinement declarations, want 1", len(ext.Refinements))
	}
	d := ext.Refinements[0]
	if d.Spec.Name.Name != "Positive" || d.Param == nil || d.Param.Names[0].Name != "value" {
		t.Fatalf("unexpected refinement declaration: %#v", d)
	}
}

func TestRefinementContextualClaim(t *testing.T) {
	for _, src := range []string{
		"package p\ntype refine int\n",
		"package p\ntype T refine\n",
		"package p\nfunc refine(value int) bool { return value > 0 }\n",
	} {
		_, ext, _ := forkparser.ParseFileExt(token.NewFileSet(), "p.go", []byte(src), forkparser.SkipObjectResolution)
		if ext != nil && len(ext.Refinements) != 0 {
			t.Fatalf("valid Go spelling was claimed as a refinement: %q", src)
		}
	}
}
