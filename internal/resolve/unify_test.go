package resolve

import (
	"go/parser"
	"testing"
)

func TestExprTextPreservesCompositeTypeExpressions(t *testing.T) {
	for _, want := range []string{
		"[]int",
		"map[string][]int",
		"pkg.Pair[int, []string]",
		"func(int) (string, error)",
	} {
		expr, err := parser.ParseExpr(want)
		if err != nil {
			t.Fatal(err)
		}
		if got := exprText(expr); got != want {
			t.Fatalf("exprText(%q) = %q", want, got)
		}
	}
}

func TestUnifyTextBindsCompositeSubexpression(t *testing.T) {
	bind := map[string]string{}
	if !unifyText("[]T", "[][]int", map[string]bool{"T": true}, bind) {
		t.Fatal("unification failed")
	}
	if got := bind["T"]; got != "[]int" {
		t.Fatalf("T = %q", got)
	}
}
