package registry

import (
	"fmt"
	"go/parser"
	"go/token"
	"strings"

	"goforge.dev/goplus/internal/directive"
)

// Refinement is a predicate-bearing subtype which erases to Base.
type Refinement struct {
	PkgPath   string
	Name      string
	Binder    string
	Base      string
	Predicate string
}

// RefinedFunc records refinement contracts on an erased top-level function
// signature. Parameter/result maps are indexed after expanding grouped fields.
type RefinedFunc struct {
	PkgPath string
	Name    string
	Params  map[int]*Refinement
	Results map[int]*Refinement
}

func (r *Refinement) Origin() string { return fmt.Sprintf("refinement %s", r.Name) }

// RefinementsFromMarkers reconstructs refinement contracts from generated Go.
func RefinementsFromMarkers(pkgPath, filename string, src []byte) ([]*Refinement, error) {
	if !strings.Contains(string(src), "//goplus:refinement") {
		return nil, nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for refinement markers: %w", filename, err)
	}
	var out []*Refinement
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			m, ok := directive.ParseRefinementMarker(c.Text)
			if !ok {
				continue
			}
			out = append(out, &Refinement{PkgPath: pkgPath, Name: m.Name, Binder: m.Binder, Base: m.Base, Predicate: m.Predicate})
		}
	}
	return out, nil
}
