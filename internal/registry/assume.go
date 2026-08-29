package registry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Assumptions (v0.150.0). An `assume` accepts a proposition on the
// author's authority rather than proving it, and erases like any other
// proof argument — so nothing of it would survive into the generated Go
// a consumer actually receives. A //goplus:assume marker carries it, so
// the audit does not stop at the module boundary: you can ask what your
// dependencies assumed, not only what you did.
//
// The marker deliberately carries no source position. Generated files are
// committed and `gen -check`ed, and a position would churn the artifact
// whenever an unrelated line moved.

// AssumePrefix is the marker directive.
const AssumePrefix = "//goplus:assume"

// Assumption is one proposition accepted rather than proved.
type Assumption struct {
	PkgPath     string
	Fn          string // the function containing the call
	Callee      string // the function whose proof parameter was assumed
	Param       string // that parameter's name
	Proposition string // the proposition, with call arguments substituted
}

// Marker renders the assumption as its directive line.
func (a *Assumption) Marker() string {
	return fmt.Sprintf("%s %s %s %s %s", AssumePrefix, a.Fn, a.Callee, a.Param, a.Proposition)
}

func (a *Assumption) key() string {
	return a.PkgPath + "\x00" + a.Fn + "\x00" + a.Callee + "\x00" + a.Param + "\x00" + a.Proposition
}

// ParseAssumeMarker reads one marker body — everything after the
// directive — into an assumption. The first three fields are the
// containing function, the callee, and the parameter; the rest is the
// proposition, which may contain spaces.
func ParseAssumeMarker(pkgPath, body string) (*Assumption, error) {
	fields := strings.SplitN(strings.TrimSpace(body), " ", 4)
	if len(fields) < 4 {
		return nil, fmt.Errorf("malformed %s marker: %q", AssumePrefix, body)
	}
	for _, f := range fields[:3] {
		if f == "" {
			return nil, fmt.Errorf("malformed %s marker: %q", AssumePrefix, body)
		}
	}
	prop := strings.TrimSpace(fields[3])
	if prop == "" {
		return nil, fmt.Errorf("malformed %s marker: %q", AssumePrefix, body)
	}
	return &Assumption{
		PkgPath:     pkgPath,
		Fn:          fields[0],
		Callee:      fields[1],
		Param:       fields[2],
		Proposition: prop,
	}, nil
}

// AddAssumption registers one assumption, ignoring exact repeats.
func (r *Registry) AddAssumption(a *Assumption) {
	if r.assumeIdx == nil {
		r.assumeIdx = map[string]*Assumption{}
	}
	r.assumeIdx[a.key()] = a
}

// Assumptions lists every registered assumption, ordered by package then
// by the declaration that contains it.
func (r *Registry) Assumptions() []*Assumption {
	out := make([]*Assumption, 0, len(r.assumeIdx))
	for _, a := range r.assumeIdx {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key() < out[j].key() })
	return out
}

// AssumptionsFromMarkers reconstructs assumptions from one generated
// file. Markers sit on the declaration that contains the call, so a
// reader can see which function rests on the claim.
func AssumptionsFromMarkers(pkgPath, filename string, src []byte) ([]*Assumption, error) {
	if !strings.Contains(string(src), AssumePrefix) {
		return nil, nil
	}
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for goplus markers: %w", filename, err)
	}
	var out []*Assumption
	for _, decl := range astFile.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Doc == nil {
			continue
		}
		for _, c := range fd.Doc.List {
			rest, ok := strings.CutPrefix(c.Text, AssumePrefix+" ")
			if !ok {
				continue
			}
			a, err := ParseAssumeMarker(pkgPath, rest)
			if err != nil {
				return nil, fmt.Errorf("%s: %v", filename, err)
			}
			out = append(out, a)
		}
	}
	return out, nil
}
