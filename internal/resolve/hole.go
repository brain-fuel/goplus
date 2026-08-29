package resolve

// Typed holes (v0.10.0). A `?name` in the source reaches resolution as the
// carrier call `__gp_hole<N>_<name>()`, which nothing defines: a hole is a
// question, not a lowering. This file answers it, reporting what the hole
// must produce and what is in scope to produce it with, and never rewrites
// anything — the diagnostic stops generation before any output is written.

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"sort"
	"strings"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/lower"
	"goforge.dev/goplus/internal/registry"
)

var holeCarrier = regexp.MustCompile(`^` + lower.HoleCarrierPrefix + `(\d+)_([A-Za-z_][A-Za-z0-9_]*)$`)

// holeCarrierName returns the source name encoded in a carrier identifier.
func holeCarrierName(ident string) (string, bool) {
	m := holeCarrier.FindStringSubmatch(ident)
	if m == nil {
		return "", false
	}
	return m[2], true
}

// holeName returns the hole's source name when call is a hole carrier.
func holeName(call *ast.CallExpr) (string, bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok || len(call.Args) != 0 {
		return "", false
	}
	return holeCarrierName(id.Name)
}

// holeCandidate reports the goal at one typed hole. Holes surface only on
// the audit pass: during the fixpoint the surrounding code is still being
// rewritten, so the expected type is not yet trustworthy.
func (r *fileResolver) holeCandidate(call *ast.CallExpr) {
	name, ok := holeName(call)
	if !ok || !r.report {
		return
	}
	info := diag.HoleInfo{
		Name:     name,
		Pos:      posOf(r.pkg.Fset, call.Pos()),
		Type:     r.holeErasedGoal(call),
		DepType:  r.holeDepGoal(call),
		Bindings: r.holeBindings(call),
	}
	r.holes = append(r.holes, info)
	r.diags = append(r.diags, diag.HoleAt(info.Pos, info.String()))
}

// holeErasedGoal renders the Go type the context expects, or "" when the
// context does not determine one.
func (r *fileResolver) holeErasedGoal(call *ast.CallExpr) string {
	if parent, isCall := r.parents[call].(*ast.CallExpr); isCall && parent.Fun == call {
		return r.holeCalledGoal(parent)
	}
	t := r.expectedType(call)
	if t == nil {
		return ""
	}
	return r.holeTypeText(t)
}

// holeCalledGoal describes a hole in function position — `?f(x)` — as the
// signature its call site demands. Unknown components print as ?.
func (r *fileResolver) holeCalledGoal(outer *ast.CallExpr) string {
	params := make([]string, 0, len(outer.Args))
	for _, arg := range outer.Args {
		if t := r.pkg.TypesInfo.TypeOf(arg); t != nil && t != types.Typ[types.Invalid] {
			params = append(params, r.holeTypeText(types.Default(t)))
			continue
		}
		params = append(params, "?")
	}
	result := "?"
	if t := r.expectedType(outer); t != nil {
		result = r.holeTypeText(t)
	}
	return fmt.Sprintf("func(%s) %s", strings.Join(params, ", "), result)
}

// holeTypeText renders a type for display. Unlike typeText it never fails:
// a goal naming a type this file cannot yet import is still worth reading.
func (r *fileResolver) holeTypeText(t types.Type) string {
	return types.TypeString(t, types.RelativeTo(r.pkg.Types))
}

// holeDepGoal recovers the un-erased spelling of the hole's expected type:
// the callee's declared parameter type at an argument position, or the
// enclosing function's declared result at a return position. It returns ""
// when the hole sits outside any dependent context.
func (r *fileResolver) holeDepGoal(call *ast.CallExpr) string {
	if text := r.holeDepArgGoal(call); text != "" {
		return text
	}
	return r.holeDepReturnGoal(call)
}

// holeDepArgGoal handles `Concat(xs, ?rest)`: the callee's parameter type,
// with the sibling arguments substituted for its index variables.
func (r *fileResolver) holeDepArgGoal(call *ast.CallExpr) string {
	parent, ok := r.parents[call].(*ast.CallExpr)
	if !ok || parent.Fun == call {
		return ""
	}
	fnIdent, _, pkgPath := calleeIdent(r.file, r.pkg.PkgPath, parent.Fun)
	if fnIdent == nil {
		return ""
	}
	d, ok := r.reg.LookupDepFn(pkgPath, fnIdent.Name)
	if !ok {
		return ""
	}
	aligned, _, shapeOK := alignDependentCallArgs(parent.Args, d, r.propTest(d))
	if !shapeOK {
		return ""
	}
	outer := map[string]string{}
	position := -1
	for i, p := range d.Params {
		if i >= len(aligned) || aligned[i] == nil {
			continue
		}
		if aligned[i] == call {
			position = i
			continue // the hole itself substitutes nothing
		}
		value := r.text(aligned[i].Pos(), aligned[i].End())
		if p.Quantity == "0" {
			value = r.normalizeIndexText(value)
		}
		outer[p.Name] = value
	}
	if position < 0 {
		return ""
	}
	raw := d.Params[position].Type
	expected, err := substTypeTextLite(raw, outer)
	if err != nil {
		return raw // the declared spelling still tells the reader more than nothing
	}
	return expected
}

// holeDepReturnGoal handles a hole in return position: the enclosing
// function's own declared result, whose index variables are the quantity-0
// parameters already in scope by name.
func (r *fileResolver) holeDepReturnGoal(call *ast.CallExpr) string {
	if !r.holeInReturnPosition(call) {
		return ""
	}
	fn := r.enclosingFuncDecl(call)
	if fn == nil {
		return ""
	}
	d, ok := r.reg.LookupDepFn(r.pkg.PkgPath, fn.Name.Name)
	if !ok {
		return ""
	}
	return d.Result
}

func (r *fileResolver) holeInReturnPosition(call *ast.CallExpr) bool {
	var node ast.Node = call
	for {
		parent, ok := r.parents[node]
		if !ok || parent == nil {
			return false
		}
		switch p := parent.(type) {
		case *ast.ReturnStmt:
			for _, res := range p.Results {
				if res == node {
					return true
				}
			}
			return false
		case *ast.ParenExpr:
			node = parent
		default:
			return false
		}
	}
}

// holeBindings lists what is in scope at the hole: the go/types bindings
// visible there, upgraded to their dependent or refined spelling where one
// is known, plus the enclosing function's quantity-0 parameters, which the
// checker can see but the erased signature no longer mentions.
func (r *fileResolver) holeBindings(call *ast.CallExpr) []diag.HoleBinding {
	var out []diag.HoleBinding
	seen := map[string]bool{}

	// The enclosing dependent signature, when there is one, is the source
	// of both un-erased parameter spellings and the erased indices that
	// go/types cannot see.
	var enclosing *registry.DepFn
	if fn := r.enclosingFuncDecl(call); fn != nil {
		if d, ok := r.reg.LookupDepFn(r.pkg.PkgPath, fn.Name.Name); ok {
			enclosing = d
		}
	}

	scope := r.pkg.Types.Scope().Innermost(call.Pos())
	for s := scope; s != nil && s != r.pkg.Types.Scope(); s = s.Parent() {
		var local []diag.HoleBinding
		for _, name := range s.Names() {
			if seen[name] || strings.HasPrefix(name, "__gp_") || name == "_" {
				continue
			}
			v, isVar := s.Lookup(name).(*types.Var)
			if !isVar || v.Pos() >= call.Pos() {
				continue
			}
			if t := v.Type(); t == nil || t == types.Typ[types.Invalid] {
				continue // the hole's own binding, mid-inference
			}
			seen[name] = true
			local = append(local, diag.HoleBinding{
				Name:    name,
				Type:    r.holeTypeText(v.Type()),
				DepType: r.bindingDepText(v, name, enclosing),
			})
		}
		out = append(out, local...)
	}

	if enclosing != nil {
		for _, p := range enclosing.Params {
			if p.Quantity != "0" || seen[p.Name] || p.Name == "_" {
				continue
			}
			seen[p.Name] = true
			out = append(out, diag.HoleBinding{
				Name:    p.Name,
				DepType: p.Type,
				Erased:  true,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// bindingDepText is a binding's un-erased spelling: the dependent type
// recovered for it, its declared parameter type in the enclosing dependent
// signature, or its refinement.
func (r *fileResolver) bindingDepText(v *types.Var, name string, enclosing *registry.DepFn) string {
	if known, ok := r.dependentVars[v]; ok {
		return known.typeText
	}
	if enclosing != nil {
		for _, p := range enclosing.Params {
			if p.Name == name {
				return p.Type
			}
		}
	}
	if ref, ok := r.refinedVars[v]; ok {
		return refinementText(ref)
	}
	return ""
}

func refinementText(ref *registry.Refinement) string {
	if ref == nil {
		return ""
	}
	return fmt.Sprintf("%s (refine(%s %s) { %s })", ref.Name, ref.Binder, ref.Base, ref.Predicate)
}
