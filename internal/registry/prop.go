package registry

import (
	"fmt"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Named propositions (v0.17.0). `type InRange[i nat, n nat] prop { … }`
// names a proposition over index terms. A use unfolds by substituting the
// arguments into the body, so nothing downstream needs to know the name:
// the decider sees the same relations it always did.
//
// The declaration is check-time only and erases completely; a
// //goplus:prop marker carries it to consumers, who receive generated Go
// and never the .gp.

// PropPrefix is the marker directive.
const PropPrefix = "//goplus:prop"

// PropDef is one named proposition.
type PropDef struct {
	PkgPath string
	Name    string
	Params  []string // index parameter names, in order
	Body    string   // the proposition, in terms of Params
}

// Marker renders the declaration as its directive line.
func (p *PropDef) Marker() string {
	return fmt.Sprintf("%s %s[%s] %s", PropPrefix, p.Name, strings.Join(p.Params, ", "), p.Body)
}

func (p *PropDef) key() string { return p.PkgPath + "." + p.Name }

// ParsePropMarker reads one marker body into a definition.
func ParsePropMarker(pkgPath, body string) (*PropDef, error) {
	body = strings.TrimSpace(body)
	open := strings.IndexByte(body, '[')
	close := strings.IndexByte(body, ']')
	if open <= 0 || close <= open {
		return nil, fmt.Errorf("malformed %s marker: %q", PropPrefix, body)
	}
	name := strings.TrimSpace(body[:open])
	var params []string
	for _, p := range strings.Split(body[open+1:close], ",") {
		if p = strings.TrimSpace(p); p != "" {
			params = append(params, p)
		}
	}
	prop := strings.TrimSpace(body[close+1:])
	if name == "" || len(params) == 0 || prop == "" {
		return nil, fmt.Errorf("malformed %s marker: %q", PropPrefix, body)
	}
	return &PropDef{PkgPath: pkgPath, Name: name, Params: params, Body: prop}, nil
}

// AddPropDef registers one named proposition.
func (r *Registry) AddPropDef(p *PropDef) error {
	if r.propIdx == nil {
		r.propIdx = map[string]*PropDef{}
	}
	if prev, ok := r.propIdx[p.key()]; ok && prev.Body != p.Body {
		return fmt.Errorf("conflicting prop markers for %s", p.key())
	}
	r.propIdx[p.key()] = p
	return nil
}

// LookupPropDef finds a named proposition.
func (r *Registry) LookupPropDef(pkgPath, name string) (*PropDef, bool) {
	p, ok := r.propIdx[pkgPath+"."+name]
	return p, ok
}

// PropDefs renders every registered proposition as the unfolding table
// the decider consumes, keyed by name. Names are package-qualified only
// where they must be: a body is written in its own package's vocabulary.
func (r *Registry) PropDefs() map[string][2]string {
	out := make(map[string][2]string, len(r.propIdx))
	for _, p := range r.propIdx {
		out[p.Name] = [2]string{strings.Join(p.Params, ","), p.Body}
	}
	return out
}

// AllPropDefs lists registered propositions in a deterministic order.
func (r *Registry) AllPropDefs() []*PropDef {
	out := make([]*PropDef, 0, len(r.propIdx))
	for _, p := range r.propIdx {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key() < out[j].key() })
	return out
}

// PropDefsFromMarkers reconstructs named propositions from one generated
// file. They sit on the erased type declaration they came from.
func PropDefsFromMarkers(pkgPath, filename string, src []byte) ([]*PropDef, error) {
	if !strings.Contains(string(src), PropPrefix) {
		return nil, nil
	}
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for goplus markers: %w", filename, err)
	}
	// The declaration erases completely — a named proposition names facts,
	// never values — so its marker is a free-floating comment rather than
	// the doc of anything. Scan every comment.
	var out []*PropDef
	for _, group := range astFile.Comments {
		for _, c := range group.List {
			rest, ok := strings.CutPrefix(c.Text, PropPrefix+" ")
			if !ok {
				continue
			}
			p, err := ParsePropMarker(pkgPath, rest)
			if err != nil {
				return nil, fmt.Errorf("%s: %v", filename, err)
			}
			out = append(out, p)
		}
	}
	return out, nil
}
