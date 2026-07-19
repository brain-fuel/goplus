package resolve

import (
	"go/ast"
	"strings"

	"goforge.dev/gpp/internal/lower"
	"goforge.dev/gpp/internal/registry"
)

// Dependent call sites (v0.7.0). The surface passes every argument —
// erased ones included (`Head(2, v)`); the signature dropped its
// 0-quantity parameters in pass 1, so the call drops the matching
// arguments here. Idempotent by arity: a call already at the erased
// arity is left alone. Erased arguments must be index expressions
// (pure); anything effectful is an error — its evaluation would vanish.

// depCallCandidate drops erased arguments from one call.
func (r *fileResolver) depCallCandidate(call *ast.CallExpr) {
	fnIdent, _, pkgPath := calleeIdent(r, call.Fun)
	if fnIdent == nil {
		return
	}
	d, ok := r.reg.LookupDepFn(pkgPath, fnIdent.Name)
	if !ok || len(d.Dropped) == 0 {
		return
	}
	if len(call.Args) != len(d.Params) {
		return // already erased (or an arity error for the backstop)
	}
	dropped := map[int]bool{}
	for _, i := range d.Dropped {
		dropped[i] = true
	}
	for i, a := range call.Args {
		if !dropped[i] {
			continue
		}
		if !pureIndexArg(a) {
			if r.report {
				r.errorf(a.Pos(), "the argument for erased parameter %s of %s must be an index expression (it is erased at runtime)",
					d.Params[i].Name, d.Name)
			}
			return
		}
	}
	for i, a := range call.Args {
		if !dropped[i] {
			continue
		}
		if i+1 < len(call.Args) {
			r.edits = append(r.edits, lower.Edit{Start: r.off(a.Pos()), End: r.off(call.Args[i+1].Pos()), New: ""})
		} else if i > 0 {
			r.edits = append(r.edits, lower.Edit{Start: r.off(call.Args[i-1].End()), End: r.off(a.End()), New: ""})
		} else {
			r.edits = append(r.edits, lower.Edit{Start: r.off(a.Pos()), End: r.off(a.End()), New: ""})
		}
	}
}

// pureIndexArg reports whether an expression is a pure index term:
// identifiers, literals, arithmetic, parens, and calls to totals
// (validated as total elsewhere; effectful calls are not total).
func pureIndexArg(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return pureIndexArg(x.X)
	case *ast.BinaryExpr:
		return pureIndexArg(x.X) && pureIndexArg(x.Y)
	case *ast.SelectorExpr:
		return true
	case *ast.CallExpr:
		for _, a := range x.Args {
			if !pureIndexArg(a) {
				return false
			}
		}
		return pureIndexArg(x.Fun)
	}
	return false
}

// scrutineeIndexTerms recovers a match scrutinee's index terms when the
// scrutinee is a parameter of the enclosing dependent function — its
// //gpp:dep marker preserves the unerased type. Unknown otherwise
// (conservative: every variant stays possible).
func (r *fileResolver) scrutineeIndexTerms(e *registry.Enum, subj ast.Expr) []string {
	if len(e.Indices) == 0 {
		return nil
	}
	id, ok := subj.(*ast.Ident)
	if !ok {
		return nil
	}
	var encl *ast.FuncDecl
	for _, decl := range r.file.Decls {
		if fd, isFn := decl.(*ast.FuncDecl); isFn && fd.Pos() <= subj.Pos() && subj.Pos() < fd.End() {
			encl = fd
		}
	}
	if encl == nil {
		return nil
	}
	if enum, terms, found := r.reg.LookupParamIndex(r.pkg.PkgPath, encl.Name.Name, id.Name); found {
		if enum == e.Name && len(terms) == len(e.Indices) {
			return terms
		}
	}
	d, ok := r.reg.LookupDepFn(r.pkg.PkgPath, encl.Name.Name)
	if !ok {
		return nil
	}
	for _, p := range d.Params {
		if p.Name != id.Name {
			continue
		}
		base, args := instantiationBase(p.Type)
		if base != e.Name || len(args) != len(e.TParams)+len(e.Indices) {
			return nil
		}
		idxPos := map[int]bool{}
		for _, ib := range e.Indices {
			idxPos[ib.Pos] = true
		}
		var terms []string
		for i, a := range args {
			if idxPos[i] {
				terms = append(terms, a)
			}
		}
		return terms
	}
	return nil
}

// instantiationBase splits "Vec[T, n+1]" into base name and args.
func instantiationBase(text string) (string, []string) {
	open := strings.IndexByte(text, '[')
	if open <= 0 || !strings.HasSuffix(text, "]") {
		return "", nil
	}
	base := strings.TrimSpace(text[:open])
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		base = base[i+1:]
	}
	var args []string
	for _, part := range splitArgsTopLevel(text[open+1 : len(text)-1]) {
		args = append(args, strings.TrimSpace(part))
	}
	return base, args
}

// splitArgsTopLevel splits comma-separated args at bracket depth zero.
func splitArgsTopLevel(s string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}
