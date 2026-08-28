package goml

// Un-erased `:type`. A declaration's signature is already written in goml
// spelling in the session's own source, so reporting it is a matter of
// printing the parsed declaration back — no pipeline run, and no erasure.
// Only when there is no such declaration to read does :type fall back to
// the generated Go type, which is where the erasure caveat belongs.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goforge.dev/goplus/internal/registry"
)

// generatedSessionFile is the Go the pipeline emits for the session.
const generatedSessionFile = "session_gml.go"

func readGeneratedSession(dir string) (string, error) {
	out, err := os.ReadFile(filepath.Join(dir, generatedSessionFile))
	return string(out), err
}

// isPlainName reports whether input is a single identifier — the only
// shape whose declaration can be looked up directly.
func isPlainName(input string) bool {
	if input == "" {
		return false
	}
	for i, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// gomlTypeString prints a type in goml spelling. prec is the surrounding
// precedence: 0 anywhere, 1 to the left of an arrow, 2 as an application
// argument.
func gomlTypeString(t Type, prec int) string {
	switch t := t.(type) {
	case *TypeName:
		if t.Pkg != "" {
			return t.Pkg + "." + t.Name
		}
		return t.Name
	case *TypeNat:
		return t.Lit
	case *TypeApp:
		parts := make([]string, 0, len(t.Args)+1)
		parts = append(parts, gomlTypeString(t.Head, 2))
		for _, a := range t.Args {
			parts = append(parts, gomlTypeString(a, 2))
		}
		return parenIf(prec >= 2, strings.Join(parts, " "))
	case *TypeArrow:
		s := gomlTypeString(t.From, 1) + " -> " + gomlTypeString(t.To, 0)
		return parenIf(prec >= 1, s)
	case *TypeIndexOp:
		s := gomlTypeString(t.L, 2) + " " + t.Op + " " + gomlTypeString(t.R, 2)
		return parenIf(prec >= 2, s)
	case *TypeEq:
		s := gomlTypeString(t.L, 2) + " = " + gomlTypeString(t.R, 2)
		return parenIf(prec >= 1, s)
	case nil:
		return ""
	}
	return ""
}

func parenIf(cond bool, s string) string {
	if cond {
		return "(" + s + ")"
	}
	return s
}

// gomlBinderString prints one binder as it was written.
func gomlBinderString(b *Binder) string {
	if b.Unit {
		return "()"
	}
	inner := ""
	if b.Quantity != "" {
		inner = b.Quantity + " "
	}
	inner += strings.Join(b.Names, " ")
	if b.Type != nil {
		if inner != "" {
			inner += " : "
		}
		inner += gomlTypeString(b.Type, 0)
	}
	switch {
	case b.Instance:
		return "[" + inner + "]"
	case b.Implicit:
		return "{" + inner + "}"
	default:
		return "(" + inner + ")"
	}
}

// gomlSigString renders a declaration's signature. It reports false when
// the declaration carries no type information to show.
func gomlSigString(d *LetDecl) (string, bool) {
	if d.Sig != nil {
		return gomlTypeString(d.Sig, 0), true
	}
	if d.Result == nil {
		// Without a result annotation the binders alone would read as a
		// signature whose last parameter is the result.
		return "", false
	}
	var parts []string
	for _, b := range d.Binders {
		if b.Unit {
			continue // a nullary function's unit binder is not a parameter type
		}
		parts = append(parts, gomlBinderString(b))
	}
	parts = append(parts, gomlTypeString(d.Result, 0))
	return strings.Join(parts, " -> "), true
}

// declaredSignature finds a named declaration in the session and renders
// its signature in goml spelling.
func (r *repl) declaredSignature(name string) (string, bool) {
	d, ok := r.sess.lookup(name)
	if !ok || d.src == "" {
		return "", false
	}
	file, err := Parse("<session>.goml", []byte("module "+replModule+"\n\n"+d.src+"\n"))
	if err != nil {
		return "", false
	}
	for _, decl := range file.Decls {
		let, isLet := decl.(*LetDecl)
		if !isLet || let.Name != name {
			continue
		}
		return gomlSigString(let)
	}
	return "", false
}

// elaboratedSignature reads the un-erased signature the pipeline recorded
// for name, which includes binders the user left implicit. It is only
// consulted when the generated file matches the current session.
func (r *repl) elaboratedSignature(name string) (string, bool) {
	if r.lastGood == "" || r.lastGood != r.sess.render(replModule, "") {
		return "", false
	}
	generated, err := readGeneratedSession(r.dir)
	if err != nil {
		return "", false
	}
	prefix := registry.DepPrefix + " " + name
	for _, line := range strings.Split(generated, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rest := strings.TrimPrefix(line, registry.DepPrefix+" ")
		// The marker's own name is followed immediately by its
		// parameters, so only an exact name match counts.
		if !strings.HasPrefix(rest, name) {
			continue
		}
		if tail := rest[len(name):]; tail == "" || strings.HasPrefix(tail, "[") || strings.HasPrefix(tail, "(") {
			return rest, true
		}
	}
	return "", false
}

// showHoles recalls the goals of the last input that had holes. A
// declaration with a hole is not retained, so this is how its goals stay
// available while the user works out the implementation.
func (r *repl) showHoles() {
	if len(r.lastHoles) == 0 {
		fmt.Fprintln(r.out, "(no open holes)")
		return
	}
	for _, h := range r.lastHoles {
		goal := h.DepType
		if goal == "" {
			goal = h.Type
		}
		if goal == "" {
			goal = "cannot infer a type from this context"
		}
		fmt.Fprintf(r.out, "?%s : %s\n", h.Name, goal)
		for _, b := range h.Bindings {
			text := b.DepType
			if text == "" {
				text = b.Type
			}
			fmt.Fprintf(r.out, "    %s : %s", b.Name, text)
			if b.Erased {
				fmt.Fprint(r.out, " (erased, quantity 0)")
			}
			fmt.Fprintln(r.out)
		}
	}
}

// reportDeclaredType prints a named binding's declared signature, adding
// the elaborated form when it says more than the source did.
func (r *repl) reportDeclaredType(name, sig string) {
	fmt.Fprintf(r.out, "%s : %s\n", name, sig)
	if elaborated, ok := r.elaboratedSignature(name); ok {
		fmt.Fprintf(r.out, "  elaborated: %s\n", elaborated)
	}
}
