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

// A proof can only be written at a call, so a proof-carrying function
// used as a value could never discharge its obligation. Every route to
// that — composition, piping, partial application, plain assignment —
// is the same bypass wearing a different hat.
func TestProofCarryingFunctionCannotBeAValue(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m]")

	for _, src := range []string{
		"package p\nfunc f() { g := Cast; _ = g }\n",
		"package p\nfunc f() { _ = __gp_comp(h, Cast) }\n",
		"package p\nfunc f() { _ = []any{Cast} }\n",
	} {
		got := scan(t, reg, src)
		if len(got) != 1 || !strings.Contains(got[0], "can only be used in a direct call") {
			t.Errorf("%q: want one value-use diagnostic, got: %v", src, got)
		}
	}
	// The declaration's own name and a member selector are not value uses.
	decl := "package p\nfunc Cast(v int) int { return v }\nfunc f(x struct{ Cast int }) int { return x.Cast }\n"
	if got := scan(t, reg, decl); len(got) != 0 {
		t.Errorf("a declaration name or selector must not be reported: %v", got)
	}
}

// A placeholder defers the call into a closure built later, where no
// proof argument can be supplied.
func TestProofCarryingFunctionCannotBePartiallyApplied(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m]")
	got := scan(t, reg, "package p\nfunc f() { _ = Cast(1+1, 2, refl, _) }\n")
	if len(got) != 1 || !strings.Contains(got[0], "cannot be partially applied") {
		t.Fatalf("want a partial-application diagnostic, got: %v", got)
	}
}

// A pipeline reaches this check as its lowered carrier, and the piped
// value would land in an erased index slot rather than the value one.
func TestProofCarryingFunctionCannotBePiped(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Cast[T any](0 n nat, 0 m nat, 0 p Eq[n, m], v Vec[T, n]) Vec[T, m]")
	got := scan(t, reg, "package p\nfunc f(v Vec[int]) { _ = __gp_bare_Cast(v, 1+1, 2, refl) }\n")
	if len(got) != 1 || !strings.Contains(got[0], "cannot be a pipeline stage") {
		t.Fatalf("want a pipeline diagnostic, got: %v", got)
	}
}

// A function with no proposition parameter is unaffected by any of it.
func TestPlainDependentFunctionMayBeAValue(t *testing.T) {
	reg := registry.New()
	depFn(t, reg, "p", "Length[T any](0 n nat, v Vec[T, n]) int")
	if got := scan(t, reg, "package p\nfunc f() { g := Length; _ = g }\n"); len(got) != 0 {
		t.Fatalf("a proof-free dependent function may be a value: %v", got)
	}
}
