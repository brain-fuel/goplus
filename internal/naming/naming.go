// Package naming implements the lowered-function naming scheme:
// concatenation with a visibility rule, plus collision detection.
package naming

import (
	"fmt"
	"go/ast"
	"unicode"
	"unicode/utf8"
)

// FuncName computes the generated function name for a method on a receiver
// type. override (from //gpp:name) wins when non-empty.
//
// Rule: concat(recvType, Capitalize(method)); the result is exported iff
// BOTH recvType and method are exported, enforced by casing the first rune.
//
//	(Stack).Map -> StackMap    (stack).Map -> stackMap
//	(Stack).map -> stackMap    (stack).map -> stackMap
func FuncName(recvType, method, override string) string {
	if override != "" {
		return override
	}
	exported := ast.IsExported(recvType) && ast.IsExported(method)
	return setFirstCase(recvType, exported) + capitalize(method)
}

func capitalize(s string) string { return setFirstCase(s, true) }

func setFirstCase(s string, upper bool) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	mapped := unicode.ToLower(r)
	if upper {
		mapped = unicode.ToUpper(r)
	}
	if mapped == r {
		return s
	}
	return string(mapped) + s[size:]
}

// Table detects name collisions between generated functions and any other
// package-scope declaration (authored or generated).
type Table struct {
	entries map[string]entry
}

type entry struct {
	origin    string
	generated bool
}

func NewTable() *Table { return &Table{entries: map[string]entry{}} }

// AddAuthored records an existing package-scope identifier, e.g.
// AddAuthored("StackMap", "util.go:14:1"). Authored identifiers never
// collide with each other (go/types owns that); they only reserve names.
func (t *Table) AddAuthored(name, origin string) {
	if _, exists := t.entries[name]; !exists {
		t.entries[name] = entry{origin: origin}
	}
}

// AddGenerated records a generated function name, returning an error if the
// name is already taken by an authored declaration or another generated
// function. origin describes the method, e.g. `method (Stack[T]) Map[U] at
// stack.gpp:5:1`.
func (t *Table) AddGenerated(name, origin string) error {
	if prev, exists := t.entries[name]; exists {
		kind := "declaration"
		if prev.generated {
			kind = "generated function"
		}
		return fmt.Errorf("generated name %s for %s collides with %s at %s; use //gpp:name to choose a different name",
			name, origin, kind, prev.origin)
	}
	t.entries[name] = entry{origin: origin, generated: true}
	return nil
}
