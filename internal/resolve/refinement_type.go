package resolve

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"goforge.dev/goplus/internal/registry"
)

// refinementCallCandidate verifies introduction through R(value). The call is
// already the desired erased Go conversion, so successful proofs need no edit.
func (r *fileResolver) refinementCallCandidate(call *ast.CallExpr) {
	ref := r.refinementForExpr(call.Fun)
	if ref == nil {
		return
	}
	if len(call.Args) != 1 || call.Ellipsis.IsValid() {
		if r.report {
			r.errorf(call.Pos(), "%s construction requires exactly one value", ref.Name)
		}
		return
	}
	if r.provesRefinement(ref, call.Args[0], call.Pos()) {
		return
	}
	if r.report {
		r.errorf(call.Pos(), "cannot prove %s for %s(%s); the value does not establish refinement %s",
			ref.Predicate, ref.Name, r.text(call.Args[0].Pos(), call.Args[0].End()), ref.Name)
	}
}

// refinedFunctionCallCandidate enforces erased refinement contracts at Go+
// call sites. Plain-Go callers are handled by the generated entry guard.
func (r *fileResolver) refinedFunctionCallCandidate(call *ast.CallExpr) {
	if !r.report {
		return
	}
	fn := r.refinedFuncForCall(call)
	if fn == nil {
		return
	}
	for i, arg := range call.Args {
		ref := fn.Params[i]
		if ref == nil || r.expressionCarriesRefinement(arg, ref) || r.provesRefinement(ref, arg, arg.Pos()) {
			continue
		}
		r.errorf(arg.Pos(), "cannot prove %s for argument %d to %s; parameter requires refinement %s",
			ref.Predicate, i+1, fn.Name, ref.Name)
	}
}

func (r *fileResolver) refinedFuncForCall(call *ast.CallExpr) *registry.RefinedFunc {
	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj = r.pkg.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		obj = r.pkg.TypesInfo.Uses[fun.Sel]
	}
	f, ok := obj.(*types.Func)
	if !ok || f.Pkg() == nil {
		return nil
	}
	sig, _ := f.Type().(*types.Signature)
	if sig == nil || sig.Recv() != nil {
		return nil
	}
	fn, _ := r.reg.LookupRefinedFunc(f.Pkg().Path(), f.Name())
	return fn
}

func (r *fileResolver) expressionCarriesRefinement(e ast.Expr, want *registry.Refinement) bool {
	if r.isCheckedRefinementConstruction(e, want) {
		return true
	}
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	fn := r.refinedFuncForCall(call)
	if fn == nil || len(fn.Results) != 1 {
		return false
	}
	got := fn.Results[0]
	return got != nil && got.PkgPath == want.PkgPath && got.Name == want.Name
}

func (r *fileResolver) refinementForExpr(fun ast.Expr) *registry.Refinement {
	var obj types.Object
	switch x := fun.(type) {
	case *ast.Ident:
		obj = r.pkg.TypesInfo.Uses[x]
	case *ast.SelectorExpr:
		obj = r.pkg.TypesInfo.Uses[x.Sel]
	}
	tn, ok := obj.(*types.TypeName)
	if !ok || tn.Pkg() == nil {
		return nil
	}
	ref, _ := r.reg.LookupRefinement(tn.Pkg().Path(), tn.Name())
	return ref
}

func (r *fileResolver) refinementForTypeExpr(expr ast.Expr) *registry.Refinement {
	var obj types.Object
	switch x := expr.(type) {
	case *ast.Ident:
		obj = r.pkg.TypesInfo.Uses[x]
	case *ast.SelectorExpr:
		obj = r.pkg.TypesInfo.Uses[x.Sel]
	}
	tn, ok := obj.(*types.TypeName)
	if !ok || tn.Pkg() == nil {
		return nil
	}
	ref, _ := r.reg.LookupRefinement(tn.Pkg().Path(), tn.Name())
	return ref
}

func (r *fileResolver) refinementForGoType(t types.Type) *registry.Refinement {
	if t == nil {
		return nil
	}
	var obj *types.TypeName
	switch x := t.(type) {
	case *types.Alias:
		obj = x.Obj()
	case *types.Named:
		obj = x.Obj()
	}
	if obj == nil || obj.Pkg() == nil {
		return nil
	}
	ref, _ := r.reg.LookupRefinement(obj.Pkg().Path(), obj.Name())
	return ref
}

// refinementExpectedCandidate checks every ordinary flow into a refined
// expected type: assignments, declarations, arguments, returns, and composite
// elements. Explicit R(x) introductions are checked by refinementCallCandidate.
func (r *fileResolver) refinementExpectedCandidate(e ast.Expr) {
	if !r.report {
		return
	}
	want := r.refinementForGoType(r.expectedType(e))
	if want == nil {
		return
	}
	if gotTV, ok := r.pkg.TypesInfo.Types[e]; ok {
		if got := r.refinementForGoType(gotTV.Type); got != nil && got.PkgPath == want.PkgPath && got.Name == want.Name {
			return
		}
	}
	if r.expressionCarriesRefinement(e, want) || r.provesRefinement(want, e, e.Pos()) {
		return
	}
	context := "this context"
	if ret, ok := r.parents[e].(*ast.ReturnStmt); ok {
		if fd := r.enclosingFuncDecl(ret); fd != nil {
			context = "a result of " + fd.Name.Name
		}
	}
	r.errorf(e.Pos(), "cannot prove %s for value %s; %s requires refinement %s",
		want.Predicate, r.text(e.Pos(), e.End()), context, want.Name)
}

// refinementReturnCandidate consults the authored result spelling retained in
// the generated AST. A Go alias deliberately disappears from signature types,
// so go/types alone cannot identify this boundary.
func (r *fileResolver) refinementReturnCandidate(ret *ast.ReturnStmt) {
	if !r.report {
		return
	}
	fd := r.enclosingFuncDecl(ret)
	if fd == nil || fd.Type.Results == nil {
		return
	}
	if len(ret.Results) == 0 {
		for _, field := range fd.Type.Results.List {
			if len(field.Names) == 0 {
				continue
			}
			if ref := r.refinementForTypeExpr(field.Type); ref != nil {
				r.errorf(ret.Pos(), "naked return cannot establish named refined result %s; return an explicit value satisfying %s", ref.Name, ref.Predicate)
			}
		}
		return
	}
	var resultTypes []ast.Expr
	for _, field := range fd.Type.Results.List {
		n := len(field.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			resultTypes = append(resultTypes, field.Type)
		}
	}
	if len(ret.Results) != len(resultTypes) {
		return
	}
	for i, value := range ret.Results {
		ref := r.refinementForTypeExpr(resultTypes[i])
		if ref == nil || r.expressionCarriesRefinement(value, ref) || r.provesRefinement(ref, value, value.Pos()) {
			continue
		}
		r.errorf(value.Pos(), "cannot prove %s for value %s; result %d of %s requires refinement %s",
			ref.Predicate, r.text(value.Pos(), value.End()), i+1, fd.Name.Name, ref.Name)
	}
}

func (r *fileResolver) provesRefinement(ref *registry.Refinement, arg ast.Expr, at token.Pos) bool {
	argText := r.text(arg.Pos(), arg.End())
	if needsParen(arg) {
		argText = "(" + argText + ")"
	}
	pred := replaceRefinementBinder(ref.Predicate, ref.Binder, argText)
	if tv, err := types.Eval(r.pkg.Fset, r.pkg.Types, at, pred); err == nil && tv.Value != nil && tv.Value.Kind() == constant.Bool {
		return constant.BoolVal(tv.Value)
	}
	want, err := parser.ParseExpr(pred)
	if err != nil {
		return false
	}
	facts := r.refinementPathFacts(at)
	return refinementExprProved(want, facts)
}

func replaceRefinementBinder(predicate, binder, value string) string {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(binder) + `\b`)
	return re.ReplaceAllString(predicate, value)
}

func (r *fileResolver) refinementPathFacts(at token.Pos) map[string]bool {
	out := map[string]bool{}
	for obj, ref := range r.refinedFactVars {
		if obj.Pos() > at || (obj.Parent() != nil && !obj.Parent().Contains(at)) {
			continue
		}
		pred := replaceRefinementBinder(ref.Predicate, ref.Binder, obj.Name())
		if e, err := parser.ParseExpr(pred); err == nil {
			collectConjunctTexts(e, out)
		}
	}
	for n := ast.Node(r.parentsNodeAt(at)); n != nil; n = r.parents[n] {
		if fd, ok := n.(*ast.FuncDecl); ok {
			for _, field := range fd.Type.Params.List {
				ref := r.refinementForTypeExpr(field.Type)
				if ref == nil {
					continue
				}
				for _, name := range field.Names {
					pred := replaceRefinementBinder(ref.Predicate, ref.Binder, name.Name)
					if e, err := parser.ParseExpr(pred); err == nil {
						collectConjunctTexts(e, out)
					}
				}
			}
		}
		ifs, ok := n.(*ast.IfStmt)
		if !ok || ifs.Cond == nil || ifs.Body == nil || !(ifs.Body.Pos() <= at && at <= ifs.Body.End()) {
			continue
		}
		collectConjunctTexts(ifs.Cond, out)
	}
	return out
}

func (r *fileResolver) indexRefinedVariables() {
	r.refinedVars = map[*types.Var]*registry.Refinement{}
	r.refinedFactVars = map[*types.Var]*registry.Refinement{}
	for _, decl := range r.file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		for _, field := range fd.Type.Params.List {
			ref := r.refinementForTypeExpr(field.Type)
			if ref == nil {
				continue
			}
			for _, id := range field.Names {
				if v, ok := r.pkg.TypesInfo.Defs[id].(*types.Var); ok {
					r.refinedVars[v], r.refinedFactVars[v] = ref, ref
				}
			}
		}
		if fd.Type.Results != nil {
			for _, field := range fd.Type.Results.List {
				ref := r.refinementForTypeExpr(field.Type)
				if ref == nil {
					continue
				}
				for _, id := range field.Names {
					if v, ok := r.pkg.TypesInfo.Defs[id].(*types.Var); ok {
						r.refinedVars[v] = ref
					}
				}
			}
		}
	}
	ast.Inspect(r.file, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || vs.Type == nil {
			return true
		}
		ref := r.refinementForTypeExpr(vs.Type)
		if ref == nil {
			return true
		}
		for i, id := range vs.Names {
			if v, ok := r.pkg.TypesInfo.Defs[id].(*types.Var); ok {
				r.refinedVars[v] = ref
				if i < len(vs.Values) {
					r.refinedFactVars[v] = ref
				}
			}
		}
		return true
	})
}

func (r *fileResolver) refinementValueSpecCandidate(vs *ast.ValueSpec) {
	if !r.report || vs.Type == nil {
		return
	}
	ref := r.refinementForTypeExpr(vs.Type)
	if ref == nil {
		return
	}
	for i, id := range vs.Names {
		if i >= len(vs.Values) {
			if !r.provesRefinementText(ref, "0", id.Pos()) {
				r.errorf(id.Pos(), "variable %s requires refinement %s but its zero value does not satisfy %s; initialize it with a proved value", id.Name, ref.Name, ref.Predicate)
			}
			continue
		}
		value := vs.Values[i]
		if r.expressionCarriesRefinement(value, ref) || r.provesRefinement(ref, value, value.Pos()) {
			continue
		}
		r.errorf(value.Pos(), "cannot prove %s for initializer %s of %s", ref.Predicate, r.text(value.Pos(), value.End()), id.Name)
	}
}

func (r *fileResolver) refinementAssignCandidate(as *ast.AssignStmt) {
	if !r.report || as.Tok != token.ASSIGN || len(as.Lhs) != len(as.Rhs) {
		return
	}
	for i, lhs := range as.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		v, _ := r.pkg.TypesInfo.Uses[id].(*types.Var)
		ref := r.refinedVars[v]
		if ref == nil {
			continue
		}
		value := as.Rhs[i]
		if r.expressionCarriesRefinement(value, ref) || r.provesRefinement(ref, value, value.Pos()) {
			continue
		}
		r.errorf(value.Pos(), "cannot prove %s for assignment to %s; it requires refinement %s", ref.Predicate, id.Name, ref.Name)
	}
}

func (r *fileResolver) provesRefinementText(ref *registry.Refinement, value string, at token.Pos) bool {
	pred := replaceRefinementBinder(ref.Predicate, ref.Binder, value)
	tv, err := types.Eval(r.pkg.Fset, r.pkg.Types, at, pred)
	return err == nil && tv.Value != nil && tv.Value.Kind() == constant.Bool && constant.BoolVal(tv.Value)
}

func (r *fileResolver) isCheckedRefinementConstruction(value ast.Expr, want *registry.Refinement) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok {
		return false
	}
	got := r.refinementForExpr(call.Fun)
	return got != nil && got.PkgPath == want.PkgPath && got.Name == want.Name
}

func (r *fileResolver) enclosingFuncDecl(n ast.Node) *ast.FuncDecl {
	for p := r.parents[n]; p != nil; p = r.parents[p] {
		if fd, ok := p.(*ast.FuncDecl); ok {
			return fd
		}
		if _, ok := p.(*ast.FuncLit); ok {
			return nil
		}
	}
	return nil
}

func (r *fileResolver) parentsNodeAt(at token.Pos) ast.Node {
	var best ast.Node
	ast.Inspect(r.file, func(n ast.Node) bool {
		if n == nil || at < n.Pos() || at > n.End() {
			return false
		}
		if best == nil || n.End()-n.Pos() < best.End()-best.Pos() {
			best = n
		}
		return true
	})
	return best
}

func refinementExprProved(e ast.Expr, facts map[string]bool) bool {
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.LAND {
		return refinementExprProved(b.X, facts) && refinementExprProved(b.Y, facts)
	}
	return facts[canonicalExpr(e)]
}

func collectConjunctTexts(e ast.Expr, out map[string]bool) {
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op == token.LAND {
		collectConjunctTexts(b.X, out)
		collectConjunctTexts(b.Y, out)
		return
	}
	out[canonicalExpr(e)] = true
}

func canonicalExpr(e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, token.NewFileSet(), e)
	return strings.TrimSpace(b.String())
}
