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
	"strings"

	"goforge.dev/goplus/internal/core"
	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/lower"
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
// a call that omits its proof argument, and — since a proof can only be
// written at a call — every use of a proof-carrying function that is not
// one: composed, piped, partially applied, or stored in a variable.
func proofObligations(pkgPath string, fset *token.FileSet, file *ast.File, reg *registry.Registry) []diag.Diagnostic {
	if reg == nil {
		return nil
	}
	// An identifier naming a proof-carrying function is fine as a callee
	// and nowhere else. A name that is merely being DECLARED — a field, a
	// parameter, a binding — is not a use of the function at all, even
	// when it happens to share the name. (A local that shadows one and is
	// then used would be misreported; the check has no type information,
	// and every name-keyed lookup here carries that same hazard.) Gather the positions that are not value uses at
	// all — the declaration's own name, and selector members, which name
	// a field rather than the function.
	exempt := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if id, _, _ := calleeIdent(file, pkgPath, x.Fun); id != nil {
				exempt[id] = true
			}
		case *ast.FuncDecl:
			exempt[x.Name] = true
		case *ast.SelectorExpr:
			exempt[x.Sel] = true
		case *ast.KeyValueExpr:
			if k, isIdent := x.Key.(*ast.Ident); isIdent {
				exempt[k] = true
			}
		case *ast.Field: // struct fields, parameters, results
			for _, n := range x.Names {
				exempt[n] = true
			}
		case *ast.ValueSpec:
			for _, n := range x.Names {
				exempt[n] = true
			}
		case *ast.TypeSpec:
			exempt[x.Name] = true
		case *ast.LabeledStmt:
			exempt[x.Label] = true
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for _, lhs := range x.Lhs {
					if n, isIdent := lhs.(*ast.Ident); isIdent {
						exempt[n] = true
					}
				}
			}
		}
		return true
	})

	var diags []diag.Diagnostic
	at := func(pos token.Pos) token.Position { return posOf(fset, pos) }

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			diags = append(diags, callProofDiags(pkgPath, file, reg, x, at)...)
		case *ast.Ident:
			if exempt[x] {
				return true
			}
			d, props := lookupProps(file, pkgPath, reg, x)
			if d == nil {
				return true
			}
			diags = append(diags, diag.At(at(x.Pos()),
				"%s carries a proof obligation (%s %s) and can only be used in a direct call, where its proof can be written",
				d.Name, props[0].name, props[0].text))
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
	// A pipeline segment is lowered to a carrier before this runs; the
	// value it pipes becomes the FIRST argument, which for a dependent
	// callee is an erased index, so the proof could never land correctly.
	piped := strings.HasPrefix(name, lower.BareCarrierPrefix)
	if piped {
		name = strings.TrimPrefix(name, lower.BareCarrierPrefix)
	}
	d, ok := reg.LookupDepFn(callPkg, name)
	if !ok {
		return nil
	}
	props := propParams(d)
	if len(props) == 0 {
		return nil
	}
	if piped {
		return []diag.Diagnostic{diag.At(at(call.Pos()),
			"%s carries a proof obligation (%s %s) and cannot be a pipeline stage: the piped value becomes its first argument, which is an erased parameter — call it directly",
			d.Name, props[0].name, props[0].text)}
	}
	// A placeholder defers the call into a closure built later, where no
	// proof argument can be supplied.
	for _, a := range call.Args {
		if ph, isIdent := a.(*ast.Ident); isIdent && ph.Name == "_" {
			return []diag.Diagnostic{diag.At(at(call.Pos()),
				"%s carries a proof obligation (%s %s) and cannot be partially applied, because the proof must be written at the call",
				d.Name, props[0].name, props[0].text)}
		}
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

// lookupProps resolves an identifier to a proof-carrying dependent
// function, when it names one.
func lookupProps(file *ast.File, pkgPath string, reg *registry.Registry, id *ast.Ident) (*registry.DepFn, []propParam) {
	d, ok := reg.LookupDepFn(pkgPath, id.Name)
	if !ok {
		return nil, nil
	}
	props := propParams(d)
	if len(props) == 0 {
		return nil, nil
	}
	return d, props
}
