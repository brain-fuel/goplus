package goml

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

const replHelp = `Commands:
  :help                 this list
  :quit, :q             leave (Ctrl-D also works)
  :type <expr>, :t      a named binding's declared signature; otherwise the Go type
  :holes                goals of the typed holes in the last input
  :gp [name]            the lowered .gp for the session, or one declaration
  :go [name]            the generated Go
  :list [name], :l      retained declarations in order ("!" marks effectful ones)
  :undo [n]             drop the last n declarations (default 1)
  :drop <name>...       drop declarations by name
  :load <file.goml>     adopt a file's imports and declarations
  :save <file.goml>     write the session as a standalone module
  :import "path" [as a] add an import
  :reset                empty the session
  :dir                  the session directory
  :{ ... :}             force a multi-line block

Bindings re-execute on every evaluation, because each evaluation compiles
and runs the whole session — put effectful code in a file, not a binding.
Expression results are never retained, so a bare call runs exactly once.
A leading "let" always declares; for let-in, parenthesize: (let x := 1; x + 1)
`

// command runs a :command, returning true to exit the REPL.
func (r *repl) command(input string) bool {
	name, rest, _ := strings.Cut(strings.TrimSpace(input), " ")
	rest = strings.TrimSpace(rest)
	switch name {
	case ":quit", ":q":
		return true
	case ":help", ":h", ":?":
		fmt.Fprint(r.out, replHelp)
	case ":dir":
		fmt.Fprintln(r.out, r.dir)
	case ":reset":
		r.sess = session{}
		r.lastHoles = nil
		fmt.Fprintln(r.out, "session cleared")
	case ":list", ":l":
		r.list(rest)
	case ":type", ":t":
		if rest == "" {
			fmt.Fprintln(r.errOut, "usage: :type <expr>")
			return false
		}
		r.typeOf(rest)
	case ":holes":
		r.showHoles()
	case ":gp":
		r.showLowering(rest)
	case ":go":
		r.showGenerated(rest)
	case ":undo":
		r.undo(rest)
	case ":drop":
		r.drop(rest)
	case ":import":
		r.addImportInput("import " + rest)
	case ":save":
		r.save(rest)
	case ":load":
		r.load(rest)
	default:
		fmt.Fprintf(r.errOut, "unknown command %q — :help for the list\n", name)
	}
	return false
}

func (r *repl) list(which string) {
	if len(r.sess.decls) == 0 && len(r.sess.imports) == 0 {
		fmt.Fprintln(r.out, "(empty session)")
		return
	}
	if which != "" {
		d, ok := r.sess.lookup(which)
		if !ok {
			fmt.Fprintf(r.errOut, "%s is not defined\n", which)
			return
		}
		fmt.Fprintln(r.out, d.src)
		return
	}
	for _, imp := range r.sess.imports {
		if imp.alias != "" {
			fmt.Fprintf(r.out, "   import %s as %s\n", imp.path, imp.alias)
		} else {
			fmt.Fprintf(r.out, "   import %s\n", imp.path)
		}
	}
	for i, d := range r.sess.decls {
		mark := " "
		if d.effectful {
			mark = "!"
		}
		first := strings.SplitN(strings.TrimSpace(d.src), "\n", 2)[0]
		if len(strings.Split(d.src, "\n")) > 1 {
			first += " ..."
		}
		fmt.Fprintf(r.out, "%s%2d %s\n", mark, i+1, first)
	}
}

func (r *repl) undo(arg string) {
	n := 1
	if arg != "" {
		parsed, err := strconv.Atoi(arg)
		if err != nil || parsed < 1 {
			fmt.Fprintln(r.errOut, "usage: :undo [n]")
			return
		}
		n = parsed
	}
	if n > len(r.sess.decls) {
		n = len(r.sess.decls)
	}
	if n == 0 {
		fmt.Fprintln(r.out, "(nothing to undo)")
		return
	}
	for i := 0; i < n; i++ {
		last := r.sess.decls[len(r.sess.decls)-1]
		r.sess.decls = r.sess.decls[:len(r.sess.decls)-1]
		fmt.Fprintf(r.out, "dropped %s\n", last.name)
	}
}

func (r *repl) drop(arg string) {
	if arg == "" {
		fmt.Fprintln(r.errOut, "usage: :drop <name>...")
		return
	}
	for _, name := range strings.Fields(arg) {
		if r.sess.dropDecl(name) {
			fmt.Fprintf(r.out, "dropped %s\n", name)
			continue
		}
		fmt.Fprintf(r.errOut, "%s is not defined\n", name)
	}
}

// showLowering prints the .gp the session transpiles to.
func (r *repl) showLowering(which string) {
	src := r.sess.render(replModule, "")
	gp, err := Convert("<session>.goml", []byte(src))
	if err != nil {
		fmt.Fprintf(r.errOut, "%v\n", err)
		return
	}
	fmt.Fprint(r.out, filterDecl(string(gp), which))
}

// showGenerated prints the Go the pipeline emitted for the session.
func (r *repl) showGenerated(which string) {
	if !r.generate(r.sess, "") {
		return
	}
	out, err := os.ReadFile(filepath.Join(r.dir, "session_gml.go"))
	if err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return
	}
	fmt.Fprint(r.out, filterDecl(string(out), which))
}

// filterDecl narrows generated text to the declarations mentioning a
// name, so :gp Foo and :go Foo stay readable in a large session.
func filterDecl(text, which string) string {
	if which == "" {
		return text
	}
	var keep []string
	for _, block := range strings.Split(text, "\n\n") {
		if strings.Contains(block, which) {
			keep = append(keep, block)
		}
	}
	if len(keep) == 0 {
		return "(nothing mentions " + which + ")\n"
	}
	return strings.Join(keep, "\n\n") + "\n"
}

// save writes the session as a standalone .goml module.
func (r *repl) save(path string) {
	if path == "" || !strings.HasSuffix(path, ".goml") {
		fmt.Fprintln(r.errOut, "usage: :save <file.goml>")
		return
	}
	module := strings.TrimSuffix(filepath.Base(path), ".goml")
	src := r.sess.render(module, "")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return
	}
	fmt.Fprintf(r.out, "wrote %s\n", path)
}

// load adopts a .goml file's imports and declarations.
func (r *repl) load(path string) {
	if path == "" || !strings.HasSuffix(path, ".goml") {
		fmt.Fprintln(r.errOut, "usage: :load <file.goml>")
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return
	}
	file, perr := Parse(path, src)
	if perr != nil {
		fmt.Fprintf(r.errOut, "%v\n", perr)
		return
	}
	next := r.sess.clone()
	for _, imp := range file.Imports {
		next.addImport(sessionImport{path: imp.Path, alias: imp.Alias})
	}
	lines := strings.Split(string(src), "\n")
	for i, d := range file.Decls {
		start := d.declPos().Line - 1
		end := len(lines)
		if i+1 < len(file.Decls) {
			end = file.Decls[i+1].declPos().Line - 1
		}
		if start < 0 || start >= len(lines) || end > len(lines) {
			continue
		}
		text := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		sd := sessionDecl{src: text}
		switch d := d.(type) {
		case *LetDecl:
			sd.name, sd.kind, sd.effectful = d.Name, kLet, looksEffectful(d.Body)
		case *TypeDecl:
			sd.name, sd.kind = d.Name, kType
		case *ClassDecl:
			sd.name, sd.kind = d.Name, kClass
		case *InstanceDecl:
			sd.name, sd.kind = d.Name, kInstance
		case *NamespaceDecl:
			sd.name, sd.kind = d.Name, kNamespace
		}
		next.addDecl(sd)
	}
	if !r.generate(next, "") {
		return
	}
	r.sess = next
	fmt.Fprintf(r.out, "loaded %d declaration(s) from %s\n", len(file.Decls), path)
}

// lookupType type-checks the generated session and reports one binding's
// Go type.
func lookupType(dir, name string) (string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax,
		Dir: dir,
		Env: append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod"),
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return "", err
	}
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		obj := pkg.Types.Scope().Lookup(name)
		if obj == nil {
			continue
		}
		t := obj.Type()
		if sig, ok := t.(*types.Signature); ok && sig.Params().Len() == 0 && sig.Results().Len() == 1 {
			t = sig.Results().At(0).Type()
		}
		return types.TypeString(t, types.RelativeTo(pkg.Types)), nil
	}
	return "", fmt.Errorf("could not determine a type for that expression")
}
