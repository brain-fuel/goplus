package resolve

import (
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/sourcemap"
)

// Backstop type-checks the final emitted texts strictly and returns every
// error, remapped into .gp positions where the error lies in a generated
// file. This is the full go/types safety net behind the targeted
// resolution pass: anything the lowering got wrong, and any ordinary type
// error the user wrote, surfaces here exactly once — before anything is
// written to disk.
func Backstop(in *Input, maps map[string]*sourcemap.Map) ([]diag.Diagnostic, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:        in.Dir,
		Overlay:    in.Texts,
		BuildFlags: in.BuildFlags,
	}
	pkgs, err := packages.Load(cfg, in.Patterns...)
	if err != nil {
		return nil, err
	}
	var diags []diag.Diagnostic
	seen := map[string]bool{}
	for _, pkg := range pkgs {
		for _, perr := range pkg.Errors {
			pos := parsePos(perr.Pos)
			msg := perr.Msg
			if m, ok := maps[pos.Filename]; ok {
				if mapped, ok := m.Map(pos); ok {
					pos = mapped
				} else {
					msg = "goplus internal lowering error (please report): " + msg
				}
			}
			msg = explainErasedUse(msg, in)
			d := diag.At(pos, "%s", msg)
			if !seen[d.String()] {
				seen[d.String()] = true
				diags = append(diags, d)
			}
		}
	}
	return diags, nil
}

// parsePos parses a go/packages error position ("file:line:col",
// "file:line", or "-"), tolerating Windows drive letters.
func parsePos(s string) token.Position {
	if s == "" || s == "-" {
		return token.Position{}
	}
	rest := s
	var col, line int
	if i := strings.LastIndex(rest, ":"); i > 1 {
		if n, err := strconv.Atoi(rest[i+1:]); err == nil {
			col = n
			rest = rest[:i]
		}
	}
	if i := strings.LastIndex(rest, ":"); i > 1 {
		if n, err := strconv.Atoi(rest[i+1:]); err == nil {
			line = n
			rest = rest[:i]
		}
	}
	if line == 0 {
		line, col = col, 0
	}
	return token.Position{Filename: rest, Line: line, Column: col}
}

// explainErasedUse recovers the reason behind an "undefined" error whose
// name is a quantity-0 parameter. Those parameters are absent from the
// generated signature by design, so using one at runtime reaches
// go/types as a plain undefined name; saying why is more use than saying
// what.
func explainErasedUse(msg string, in *Input) string {
	name, ok := strings.CutPrefix(msg, "undefined: ")
	if !ok || strings.ContainsAny(name, ". ") {
		return msg
	}
	for _, deps := range in.DepsByDir {
		for _, d := range deps {
			for _, p := range d.Params {
				if p.Quantity == "0" && p.Name == name {
					return msg + " — a quantity-0 parameter exists only at check time, so it is erased from the generated code; use it in types and index terms, not as a value"
				}
			}
		}
	}
	return msg
}
