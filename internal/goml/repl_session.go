package goml

import (
	"fmt"
	"sort"
	"strings"
)

// The REPL keeps a session of declarations and replays it on every
// evaluation: goml has no interpreter, so each input is compiled by the
// real pipeline and run by the real go tool. That buys exact semantics
// and costs a compile per evaluation.

type declKind int

const (
	kLet declKind = iota
	kType
	kClass
	kInstance
	kNamespace
)

// sessionDecl is one retained declaration, kept as the source the user
// typed so `:list` and `:save` show their own words back to them.
type sessionDecl struct {
	name      string
	kind      declKind
	src       string
	effectful bool
}

// sessionImport is one import line.
type sessionImport struct {
	path  string // quoted, as written
	alias string
}

// session is the accumulated state. It is a value: the REPL clones it
// before each evaluation and commits only on success, so a failing
// input leaves no trace.
type session struct {
	imports []sessionImport
	decls   []sessionDecl
	lastExp string // source of the last value expression, bound to `it`
}

func (s session) clone() session {
	out := session{lastExp: s.lastExp}
	out.imports = append([]sessionImport(nil), s.imports...)
	out.decls = append([]sessionDecl(nil), s.decls...)
	return out
}

// addImport records an import, replacing any earlier one for the path.
func (s *session) addImport(imp sessionImport) {
	for i, existing := range s.imports {
		if existing.path == imp.path {
			s.imports[i] = imp
			return
		}
	}
	s.imports = append(s.imports, imp)
}

// addDecl appends a declaration, replacing any earlier one of the same
// name. The replacement moves to the end so it can reference everything
// defined before it.
func (s *session) addDecl(d sessionDecl) {
	s.dropDecl(d.name)
	s.decls = append(s.decls, d)
}

func (s *session) dropDecl(name string) bool {
	for i, d := range s.decls {
		if d.name == name {
			s.decls = append(s.decls[:i], s.decls[i+1:]...)
			return true
		}
	}
	return false
}

func (s session) lookup(name string) (sessionDecl, bool) {
	for _, d := range s.decls {
		if d.name == name {
			return d, true
		}
	}
	return sessionDecl{}, false
}

// render materialises the session as one .goml source. trailing is the
// current input, appended last so its position is stable for the
// incompleteness check and so it can use everything before it.
//
// Instances are rendered with @[laws off]: law generation would emit a
// test importing pgregory.net/rapid, which the session module does not
// require, and the directive is read per instance rather than per file.
func (s session) render(module string, trailing string) string {
	var b strings.Builder
	b.WriteString("module " + module + "\n")

	used := s.usedImports(trailing)
	for _, imp := range s.imports {
		if !used[imp.path] {
			continue
		}
		if imp.alias != "" {
			fmt.Fprintf(&b, "\nimport %s as %s", imp.path, imp.alias)
		} else {
			fmt.Fprintf(&b, "\nimport %s", imp.path)
		}
	}
	if len(s.imports) > 0 {
		b.WriteString("\n")
	}
	for _, d := range s.decls {
		b.WriteString("\n")
		if d.kind == kInstance && !strings.Contains(d.src, "@[laws") {
			b.WriteString("@[laws off]\n")
		}
		b.WriteString(strings.TrimRight(d.src, "\n") + "\n")
	}
	if trailing != "" {
		b.WriteString("\n" + strings.TrimRight(trailing, "\n") + "\n")
	}
	return b.String()
}

// usedImports reports which import paths are actually referenced, by
// qualifier, across the session and the current input. Go rejects an
// unused aliased import outright, and a saved session should carry no
// dead imports. The scan uses the real lexer, so comments and string
// literals cannot produce false positives.
func (s session) usedImports(trailing string) map[string]bool {
	var body strings.Builder
	for _, d := range s.decls {
		body.WriteString(d.src + "\n")
	}
	body.WriteString(trailing)
	quals := qualifiers([]byte(body.String()))

	used := map[string]bool{}
	for _, imp := range s.imports {
		if quals[importQualifier(imp)] {
			used[imp.path] = true
		}
	}
	return used
}

// importQualifier is the name an import is referenced by: its alias, or
// the last element of its path.
func importQualifier(imp sessionImport) string {
	if imp.alias != "" {
		return imp.alias
	}
	path := strings.Trim(imp.path, `"`)
	if i := strings.LastIndex(path, "/"); i >= 0 {
		path = path[i+1:]
	}
	return path
}

// qualifiers collects every identifier that appears immediately before a
// dot, which is exactly the set of package qualifiers in use.
func qualifiers(src []byte) map[string]bool {
	out := map[string]bool{}
	toks, err := newLexer("<session>", src).tokens()
	if err != nil {
		return out
	}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind == IDENT && toks[i+1].Kind == Dot {
			out[toks[i].Text] = true
		}
	}
	return out
}

// autoImports maps a qualifier to the standard-library path a REPL user
// almost certainly means, so `fmt.Println "hi"` works without ceremony.
var autoImports = map[string]string{
	"fmt": "fmt", "os": "os", "io": "io", "bufio": "bufio",
	"bytes": "bytes", "strings": "strings", "strconv": "strconv",
	"errors": "errors", "sort": "sort", "slices": "slices",
	"maps": "maps", "cmp": "cmp", "math": "math", "time": "time",
	"regexp": "regexp", "json": "encoding/json", "filepath": "path/filepath",
	"http": "net/http", "rand": "math/rand", "sync": "sync",
	"context": "context", "log": "log", "unicode": "unicode",
	"utf8": "unicode/utf8", "reflect": "reflect",
}

// missingImports lists auto-importable packages the input references but
// the session has not imported, in deterministic order.
func (s session) missingImports(input string) []sessionImport {
	have := map[string]bool{}
	for _, imp := range s.imports {
		have[importQualifier(imp)] = true
	}
	for _, d := range s.decls {
		_ = d
	}
	var out []sessionImport
	var names []string
	for q := range qualifiers([]byte(input)) {
		if have[q] {
			continue
		}
		if _, ok := autoImports[q]; ok {
			names = append(names, q)
		}
	}
	sort.Strings(names)
	for _, q := range names {
		out = append(out, sessionImport{path: `"` + autoImports[q] + `"`})
	}
	return out
}

// effectQualifiers are packages whose use marks a binding as effectful,
// so the REPL can warn that it will re-run on every evaluation.
var effectQualifiers = map[string]bool{
	"fmt": true, "os": true, "io": true, "bufio": true, "log": true,
	"net": true, "http": true, "exec": true, "syscall": true,
	"rand": true, "time": true, "sync": true, "context": true,
}

// looksEffectful reports whether a declaration body performs effects, so
// the REPL can flag that it re-executes on every later evaluation. It is
// a heuristic, and `:help` says so.
func looksEffectful(e Expr) bool {
	found := false
	var walk func(Expr)
	walk = func(e Expr) {
		if found || e == nil {
			return
		}
		switch e := e.(type) {
		case *DoBlock, *SelectExpr, *Try:
			found = true
		case *Selector:
			if id, ok := e.X.(*Ident); ok && effectQualifiers[id.Name] {
				found = true
				return
			}
			walk(e.X)
		case *IndexExpr:
			walk(e.X)
			walk(e.Index)
		case *App:
			walk(e.Fn)
			for _, a := range e.Args {
				walk(a)
			}
		case *Binop:
			walk(e.L)
			walk(e.R)
		case *Unary:
			walk(e.X)
		case *If:
			walk(e.Cond)
			walk(e.Then)
			walk(e.Else)
		case *Match:
			walk(e.Subject)
			for _, cl := range e.Clauses {
				walk(cl.Body)
			}
		case *LetIn:
			walk(e.Val)
			walk(e.Body)
		case *LetStar:
			walk(e.Val)
			walk(e.Body)
		case *Fun:
			walk(e.Body)
		case *RecordLit:
			for _, f := range e.Fields {
				walk(f.Val)
			}
		case *RecordUpdate:
			walk(e.Base)
			for _, f := range e.Fields {
				walk(f.Val)
			}
		}
	}
	walk(e)
	return found
}
