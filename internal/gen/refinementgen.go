package gen

import (
	"fmt"
	"go/ast"
	"strings"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/lower"
	"goforge.dev/goplus/internal/naming"
	"goforge.dev/goplus/internal/registry"
	"goforge.dev/goplus/internal/syntax"
)

// planRefinements captures local refinement contracts before declarations
// erase. Predicate proof checking runs in resolve after package types exist.
func planRefinements(idx *pkgIndex, pkgPath string, tbl *naming.Table) ([]*registry.Refinement, []diag.Diagnostic) {
	var out []*registry.Refinement
	var diags []diag.Diagnostic
	seen := map[string]bool{}
	for _, f := range idx.files {
		if f.gp == nil {
			continue
		}
		text := func(n ast.Node) string { return string(f.src[f.gp.Offset(n.Pos()):f.gp.Offset(n.End())]) }
		for _, d := range f.gp.Refinements {
			if d.Gen == nil || d.Param == nil || len(d.Param.Names) != 1 {
				continue
			}
			name := d.Spec.Name.Name
			if seen[name] {
				diags = append(diags, diag.At(idx.fset.Position(d.Spec.Name.Pos()), "duplicate refinement %s", name))
				continue
			}
			seen[name] = true
			if err := tbl.AddGenerated(naming.RefinementPredicateName(name), "predicate for refinement "+name+" at "+idx.fset.Position(d.Spec.Name.Pos()).String()); err != nil {
				diags = append(diags, diag.At(idx.fset.Position(d.Spec.Name.Pos()), "%v", err))
				continue
			}
			out = append(out, &registry.Refinement{
				PkgPath: pkgPath, Name: name, Binder: d.Param.Names[0].Name,
				Base: text(d.Param.Type), Predicate: text(d.Predicate),
			})
		}
	}
	return out, diags
}

// refinementGuardEdits protects exported erased boundaries from ordinary Go
// callers, which cannot participate in Go+'s static refinement proof.
func refinementGuardEdits(f *syntax.File, refs []*registry.Refinement) []lower.Edit {
	byName := map[string]*registry.Refinement{}
	for _, ref := range refs {
		byName[ref.Name] = ref
	}
	var edits []lower.Edit
	for _, decl := range f.AST.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil || !ast.IsExported(fd.Name.Name) {
			continue
		}
		var guards []string
		for _, field := range fd.Type.Params.List {
			id, ok := field.Type.(*ast.Ident)
			if !ok || byName[id.Name] == nil {
				continue
			}
			ref := byName[id.Name]
			for _, param := range field.Names {
				guards = append(guards, fmt.Sprintf("if !%s(%s) { panic(%q) }",
					naming.RefinementPredicateName(ref.Name), param.Name,
					"goplus: "+fd.Name.Name+": "+param.Name+" violates refinement "+ref.Name))
			}
		}
		if len(guards) > 0 {
			at := f.Offset(fd.Body.Lbrace) + 1
			edits = append(edits, lower.Edit{Start: at, End: at, New: "\n" + strings.Join(guards, "\n") + "\n"})
		}
	}
	return edits
}
