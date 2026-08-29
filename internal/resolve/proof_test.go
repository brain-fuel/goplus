package resolve

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"goforge.dev/goplus/internal/registry"
)

// depFn builds a registered dependent signature from its marker text.
func depFn(t *testing.T, reg *registry.Registry, pkgPath, sig string) {
	t.Helper()
	d, err := registry.ParseDepSig(pkgPath, sig)
	if err != nil {
		t.Fatalf("ParseDepSig(%q): %v", sig, err)
	}
	if err := reg.AddDepFn(d); err != nil {
		t.Fatal(err)
	}
}

func scan(t *testing.T, reg *registry.Registry, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var msgs []string
	for _, d := range proofObligations("p", fset, file, reg) {
		msgs = append(msgs, d.Msg)
	}
	return msgs
}

// A proposition argument may not be omitted: unlike an index, it is not
// something the compiler can recover.
func TestOmittedProofIsReported(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m]")

	got := scan(t, reg, "package p\nfunc f(v Vec[int]) Vec[int] { return Cast(v) }\n")
	if len(got) != 1 {
		t.Fatalf("want one diagnostic, got %d: %v", len(got), got)
	}
	for _, want := range []string{"proof argument for p of Cast cannot be omitted", "Eq[n, m]", "refl", "assume"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("diagnostic missing %q: %s", want, got[0])
		}
	}
	// A call that supplies every argument is checked by the ordinary
	// call-site path, not here.
	if got := scan(t, reg, "package p\nfunc f(v Vec[int]) Vec[int] { return Cast(1+1, 2, refl, v) }\n"); len(got) != 0 {
		t.Errorf("a complete call must not be reported: %v", got)
	}
}

// The guard against regressing the ordinary erased-index sites: an
// omitted plain nat index is inference, not a missing proof.
func TestOmittedIndexIsNotReported(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Length[T any](0 n nat, v Vec[T, n]) int")
	depFn(t, reg, "p", "Concat[T any](0 n nat, 0 m nat, a Vec[T, n], b Vec[T, m]) Vec[T, n+m]")

	src := "package p\n" +
		"func f(v Vec[int], b Vec[int]) int { return Length(v) + Length(Concat(v, b)) }\n"
	if got := scan(t, reg, src); len(got) != 0 {
		t.Fatalf("omitted erased indices must stay inferable, got: %v", got)
	}
}

// A proposition with no runtime parameters degenerates to a zero-argument
// call, which must still be caught.
func TestOmittedProofWithNoRuntimeArgs(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Swap(0 n nat, 0 m nat, 0 p Eq[n, m]) string")
	if got := scan(t, reg, "package p\nfunc f() string { return Swap() }\n"); len(got) != 1 {
		t.Fatalf("want one diagnostic, got %d: %v", len(got), got)
	}
}

// Ordering propositions report their own witness, since refl is
// reflexivity and means nothing for a strict inequality.
func TestOmittedOrderingProofNamesDecide(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Bounded(0 i nat, 0 n nat, 0 p Lt[i, n]) int")
	got := scan(t, reg, "package p\nfunc f() int { return Bounded() }\n")
	if len(got) != 1 || !strings.Contains(got[0], "pass decide") {
		t.Fatalf("want a diagnostic naming decide, got: %v", got)
	}
}

// An unknown callee is not this check's business.
func TestUnknownCalleeIsIgnored(t *testing.T) {
	reg := registry.New()
	if got := scan(t, reg, "package p\nfunc f() { g(1, 2) }\n"); len(got) != 0 {
		t.Fatalf("unknown callee reported: %v", got)
	}
}
