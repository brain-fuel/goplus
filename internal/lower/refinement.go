package lower

import (
	"fmt"
	"go/ast"

	"goforge.dev/goplus/internal/directive"
	"goforge.dev/goplus/internal/naming"
	"goforge.dev/goplus/internal/syntax"
)

// RefinementEdits erases a refinement declaration to a Go alias while
// retaining its proof contract in a machine-readable marker.
func RefinementEdits(f *syntax.File, d *syntax.RefinementDecl) []Edit {
	if d.Gen == nil || d.Param == nil || len(d.Param.Names) != 1 {
		return nil
	}
	text := func(n ast.Node) string { return string(f.Src[f.Offset(n.Pos()):f.Offset(n.End())]) }
	m := directive.RefinementMarker{
		Name:      d.Spec.Name.Name,
		Binder:    d.Param.Names[0].Name,
		Base:      text(d.Param.Type),
		Predicate: text(d.Predicate),
	}
	markerAt := f.Offset(d.Gen.Pos())
	if d.Gen.Doc != nil {
		markerAt = f.Offset(d.Gen.Doc.Pos())
	}
	for markerAt > 0 && f.Src[markerAt-1] != '\n' {
		markerAt--
	}
	return []Edit{
		{Start: markerAt, End: markerAt, New: m.String() + "\n"},
		{Start: f.Offset(d.Gen.Pos()), End: f.Offset(d.Gen.End()), New: fmt.Sprintf("type %s = %s\n\nfunc %s(%s %s) bool { return %s }",
			m.Name, m.Base, naming.RefinementPredicateName(m.Name), m.Binder, m.Base, m.Predicate)},
	}
}
