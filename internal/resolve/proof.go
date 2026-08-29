package resolve

// Proof obligations at dependent call sites.
//
// A proof argument is DELETED by the same pass that discharges it, so a
// call whose author never wrote one is textually identical to a call
// whose argument has already been erased. That ambiguity is why an
// omitted proof used to be accepted in silence: the checker could not
// tell "never proved" from "proved and erased", so it assumed the
// latter.
//
// The ambiguity exists only after erasure has begun. On the first
// fixpoint iteration the text is exactly what pass 1 produced, and pass 1
// does not touch call arguments — a match lowers to a skeleton whose arm
// bodies are copied verbatim, and hoisting moves expressions without
// rewriting their calls. So every argument the author wrote is present,
// and an erased-arity call at that moment is an omission with certainty
// rather than a guess.
//
// This check therefore runs once, before any edit is applied, and needs
// no type information: callees resolve through the file's imports, and
// propositions are classified from marker text. That matters, because on
// iteration 0 a match arm's binders do not exist yet, so anything
// type-directed would give up exactly where the code lives.

import (
	"go/ast"
	"go/token"

	"goforge.dev/goplus/internal/core"
	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/registry"
)

// propParam is one erased parameter that carries a proposition.
type propParam struct {
	name string // the parameter's name
	text string // its declared type, e.g. "Eq[n, m]"
	op   core.PropOp
}

// propParams lists the proposition-carrying parameters of a dependent
// signature. Only erased parameters are considered, which is complete:
// generation already refuses a proposition parameter that is not erased.
func propParams(d *registry.DepFn) []propParam {
	var out []propParam
	for _, p := range d.Params {
		if p.Quantity != "0" {
			continue
		}
		base, terms := instantiationBase(p.Type)
		op, isProp := core.PropFor(base)
		if !isProp || len(terms) != 2 {
			continue
		}
		out = append(out, propParam{name: p.Name, text: p.Type, op: op})
	}
	return out
}

// proofObligations reports the obligations this file leaves unsettled:
// a call that omits its proof argument entirely.
func proofObligations(pkgPath string, fset *token.FileSet, file *ast.File, reg *registry.Registry) []diag.Diagnostic {
	if reg == nil {
		return nil
	}
	var diags []diag.Diagnostic
	at := func(pos token.Pos) token.Position { return posOf(fset, pos) }
	ast.Inspect(file, func(n ast.Node) bool {
		if call, isCall := n.(*ast.CallExpr); isCall {
			diags = append(diags, callProofDiags(pkgPath, file, reg, call, at)...)
		}
		return true
	})
	return diags
}

// callProofDiags checks one call: its arity, and whether a placeholder
// defers it into a closure that could never carry the proof.
func callProofDiags(pkgPath string, file *ast.File, reg *registry.Registry, call *ast.CallExpr, at func(token.Pos) token.Position) []diag.Diagnostic {
	id, _, callPkg := calleeIdent(file, pkgPath, call.Fun)
	if id == nil {
		return nil
	}
	name := id.Name
	d, ok := reg.LookupDepFn(callPkg, name)
	if !ok {
		return nil
	}
	props := propParams(d)
	if len(props) == 0 {
		return nil
	}
	if len(call.Args) == len(d.Params) {
		return nil // every argument is present; the call site is checked as usual
	}
	var out []diag.Diagnostic
	for _, p := range props {
		out = append(out, diag.At(at(call.Lparen),
			"the proof argument for %s of %s cannot be omitted: %s is a proposition, not an inferable index — pass %s (proved by the decider) or assume (asserted on your authority)",
			p.name, d.Name, p.text, p.op.Witness()))
	}
	return out
}
