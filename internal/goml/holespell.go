package goml

// Re-spelling core type text into goml. The shared core computes a hole's
// goal in the .gp/Go spelling it works in — `Vec[a, n+1]`, `[]string` —
// but a goml author writes `Vec a (n + 1)` and `Slice String`, and a goal
// that answers in the other surface's notation is a goal you have to
// translate before you can use it. This maps the answer back.
//
// Anything that does not parse as a Go type expression is returned
// unchanged: an approximate goal in the wrong spelling still beats a
// mangled one.

import (
	"go/ast"
	goparser "go/parser"
	"strings"

	"goforge.dev/goplus/internal/diag"
)

// gomlNameFor inverts gomlBuiltins, so the two directions cannot drift.
// Where several goml names share a Go spelling the first wins, which is
// why the map is built once rather than written out.
var gomlNameFor = func() map[string]string {
	out := make(map[string]string, len(gomlBuiltins))
	for goml, goName := range gomlBuiltins {
		if prev, seen := out[goName]; !seen || goml < prev {
			out[goName] = goml
		}
	}
	return out
}()

// respellHole rewrites a goal's types into goml spelling.
func respellHole(h diag.HoleInfo) diag.HoleInfo {
	h.Type = gomlSpelling(h.Type)
	h.DepType = gomlSpelling(h.DepType)
	bindings := make([]diag.HoleBinding, len(h.Bindings))
	for i, b := range h.Bindings {
		b.Type = gomlSpelling(b.Type)
		b.DepType = gomlSpelling(b.DepType)
		bindings[i] = b
	}
	h.Bindings = bindings
	return h
}

// gomlSpelling renders one Go/.gp type text in goml notation.
func gomlSpelling(text string) string {
	out, ok := spellText(text, 0)
	if !ok {
		return text
	}
	return out
}

// spellText renders one type text, handling the index terms that make a
// dependent instantiation invalid Go. `Vec[a, n+1]` does not parse —
// Go's index lists take types, and `n+1` is a term — so an instantiation
// is split textually and its arguments spelled one at a time.
func spellText(text string, prec int) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return text, false
	}
	if base, args, ok := splitInstantiation(text); ok {
		parts := []string{base}
		for _, a := range args {
			s, spelled := spellText(a, 2)
			if !spelled {
				return "", false
			}
			parts = append(parts, s)
		}
		return parens(prec >= 2, strings.Join(parts, " ")), true
	}
	expr, err := goparser.ParseExpr(text)
	if err != nil {
		return "", false
	}
	return spellType(expr, prec)
}

// splitInstantiation splits `Base[a, n+1]` into its base name and
// argument texts. It reports false for anything that is not a bracketed
// instantiation of a plain (possibly qualified) name.
func splitInstantiation(text string) (string, []string, bool) {
	if !strings.HasSuffix(text, "]") {
		return "", nil, false
	}
	open := -1
	depth := 0
	for i := len(text) - 1; i >= 0; i-- {
		switch text[i] {
		case ']':
			depth++
		case '[':
			depth--
			if depth == 0 {
				open = i
			}
		}
		if open >= 0 {
			break
		}
	}
	if open <= 0 {
		return "", nil, false
	}
	base := strings.TrimSpace(text[:open])
	if base == "" || !isQualifiedName(base) {
		return "", nil, false
	}
	inner := text[open+1 : len(text)-1]
	args := splitTopLevel(inner)
	if len(args) == 0 {
		return "", nil, false
	}
	if name, found := gomlNameFor[base]; found {
		base = name
	}
	return base, args, true
}

// isQualifiedName reports whether text is `Name` or `pkg.Name`.
func isQualifiedName(text string) bool {
	for i, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case i > 0 && (r >= '0' && r <= '9' || r == '.'):
		default:
			return false
		}
	}
	return text != ""
}

// splitTopLevel splits on commas outside brackets and parentheses.
func splitTopLevel(s string) []string {
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
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	last := strings.TrimSpace(s[start:])
	if last != "" {
		out = append(out, last)
	}
	for _, a := range out {
		if a == "" {
			return nil
		}
	}
	return out
}

// spellType renders one type expression. prec is the surrounding
// precedence: 0 anywhere, 1 left of an arrow, 2 as an application
// argument. ok is false when the shape has no goml spelling, in which
// case the caller keeps the original text.
func spellType(e ast.Expr, prec int) (string, bool) {
	switch t := e.(type) {
	case *ast.Ident:
		if name, found := gomlNameFor[t.Name]; found {
			return name, true
		}
		return t.Name, true
	case *ast.SelectorExpr:
		pkg, ok := spellType(t.X, 2)
		if !ok {
			return "", false
		}
		return pkg + "." + t.Sel.Name, true
	case *ast.BasicLit:
		return t.Value, true
	case *ast.ParenExpr:
		return spellType(t.X, prec)
	case *ast.StarExpr:
		return spellApp(prec, "Ptr", t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return spellApp(prec, "Slice", t.Elt)
		}
		return spellApp(prec, "Array", t.Len, t.Elt)
	case *ast.MapType:
		return spellApp(prec, "Map", t.Key, t.Value)
	case *ast.ChanType:
		return spellApp(prec, "Chan", t.Value)
	case *ast.Ellipsis:
		return spellApp(prec, "Slice", t.Elt)
	case *ast.IndexExpr:
		return spellApp(prec, spellHead(t.X), t.Index)
	case *ast.IndexListExpr:
		return spellApp(prec, spellHead(t.X), t.Indices...)
	case *ast.BinaryExpr:
		// Index arithmetic: n+1 reads as n + 1.
		l, lok := spellType(t.X, 2)
		r, rok := spellType(t.Y, 2)
		if !lok || !rok {
			return "", false
		}
		return parens(prec >= 2, l+" "+t.Op.String()+" "+r), true
	case *ast.FuncType:
		return spellFunc(t, prec)
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "Type", true // `any` after erasure
		}
		return "", false
	}
	return "", false
}

// spellHead renders an instantiation's head name.
func spellHead(e ast.Expr) string {
	s, ok := spellType(e, 2)
	if !ok {
		return ""
	}
	return s
}

// spellApp renders a head applied to arguments by juxtaposition.
func spellApp(prec int, head string, args ...ast.Expr) (string, bool) {
	out := []string{head}
	for _, a := range args {
		s, ok := spellType(a, 2)
		if !ok {
			return "", false
		}
		out = append(out, s)
	}
	return parens(prec >= 2, strings.Join(out, " ")), true
}

// spellFunc renders func(A, B) C as A -> B -> C, the curried spelling
// goml writes. A multi-result Go function has no arrow spelling.
func spellFunc(t *ast.FuncType, prec int) (string, bool) {
	var parts []string
	if t.Params != nil {
		for _, field := range t.Params.List {
			n := len(field.Names)
			if n == 0 {
				n = 1
			}
			s, ok := spellType(field.Type, 1)
			if !ok {
				return "", false
			}
			for i := 0; i < n; i++ {
				parts = append(parts, s)
			}
		}
	}
	result := "Unit"
	if t.Results != nil && len(t.Results.List) > 0 {
		if len(t.Results.List) > 1 || len(t.Results.List[0].Names) > 1 {
			return "", false // multi-result: no arrow spelling
		}
		s, ok := spellType(t.Results.List[0].Type, 0)
		if !ok {
			return "", false
		}
		result = s
	}
	if len(parts) == 0 {
		parts = append(parts, "Unit")
	}
	parts = append(parts, result)
	return parens(prec >= 1, strings.Join(parts, " -> ")), true
}

func parens(cond bool, s string) string {
	if cond {
		return "(" + s + ")"
	}
	return s
}
