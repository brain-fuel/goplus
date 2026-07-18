package naming

import (
	"go/ast"
	"go/token"
)

// Decl is a package-scope identifier and where it was declared.
type Decl struct {
	Name     string
	Position string
}

// TopLevelDecls lists every package-scope identifier declared in a file
// (functions, types, vars, consts — not methods), for reserving authored
// names in a Table.
func TopLevelDecls(fset *token.FileSet, f *ast.File) []Decl {
	var out []Decl
	add := func(id *ast.Ident) {
		if id != nil && id.Name != "_" {
			out = append(out, Decl{Name: id.Name, Position: fset.Position(id.Pos()).String()})
		}
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				add(d.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					add(s.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						add(n)
					}
				}
			}
		}
	}
	return out
}
