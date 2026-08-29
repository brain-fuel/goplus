package lower

import (
	"go/ast"
	"strings"

	"goforge.dev/goplus/internal/registry"
	"goforge.dev/goplus/internal/syntax"
)

// PropEdits lowers one `type InRange[i nat, n nat] prop { … }`
// declaration. A named proposition is check-time only: it names facts,
// never values, so the declaration ERASES COMPLETELY and only its marker
// survives — which is what a consumer of the generated Go needs to unfold
// a use in their own package.
func PropEdits(f *syntax.File, d *syntax.PropDecl) []Edit {
	if d.Gen == nil || d.Spec == nil || d.Body == nil {
		return nil
	}
	def := &registry.PropDef{
		Name:   d.Spec.Name.Name,
		Params: PropParamNames(d.Spec),
		Body:   PropBodyText(f, d),
	}
	markerAt := f.Offset(d.Gen.Pos())
	if d.Gen.Doc != nil {
		markerAt = f.Offset(d.Gen.Doc.Pos())
	}
	for markerAt > 0 && f.Src[markerAt-1] != '\n' {
		markerAt--
	}
	// The marker attaches to nothing, so it needs its own blank line to
	// stay a free-floating comment rather than documenting whatever
	// declaration happens to follow.
	end := f.Offset(d.Gen.End())
	return []Edit{
		{Start: markerAt, End: end, New: def.Marker() + "\n"},
	}
}

// PropBodyText renders a proposition's body on one line.
func PropBodyText(f *syntax.File, d *syntax.PropDecl) string {
	if d.Body == nil {
		return ""
	}
	raw := string(f.Src[f.Offset(d.Body.Pos()):f.Offset(d.Body.End())])
	return strings.Join(strings.Fields(raw), " ")
}

// PropParamNames lists a proposition's index parameters in order.
func PropParamNames(spec *ast.TypeSpec) []string {
	var out []string
	if spec.TypeParams == nil {
		return out
	}
	for _, field := range spec.TypeParams.List {
		for _, n := range field.Names {
			out = append(out, n.Name)
		}
	}
	return out
}
