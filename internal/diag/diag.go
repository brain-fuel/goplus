// Package diag defines goplus's user-facing diagnostics.
package diag

import (
	"fmt"
	"go/token"
	"io"
	"sort"
	"strings"
)

// Kind classifies how a diagnostic reads. The zero value is KindError, so
// every ordinary construction stays an error without naming the field.
type Kind int

const (
	KindError Kind = iota // a mistake in the source
	KindHole              // a typed hole's goal: informational, but still blocks generation
)

// Diagnostic is one user-facing error, positioned in .gp source when
// attributable.
type Diagnostic struct {
	Pos  token.Position // zero value when the diagnostic has no position
	Msg  string
	Kind Kind
}

func (d Diagnostic) String() string {
	if d.Pos.Filename == "" && d.Pos.Line == 0 {
		return d.Msg
	}
	return fmt.Sprintf("%s: %s", d.Pos, d.Msg)
}

// Errorf builds an unpositioned diagnostic.
func Errorf(format string, args ...any) Diagnostic {
	return Diagnostic{Msg: fmt.Sprintf(format, args...)}
}

// At builds a positioned diagnostic.
func At(pos token.Position, format string, args ...any) Diagnostic {
	return Diagnostic{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

// HoleAt builds the goal diagnostic for one typed hole.
func HoleAt(pos token.Position, msg string) Diagnostic {
	return Diagnostic{Pos: pos, Msg: msg, Kind: KindHole}
}

// HoleBinding is one binding visible at a typed hole. Type is the erased Go
// spelling; DepType carries the un-erased dependent or refined spelling when
// one exists. Erased marks a quantity-0 binding, which is in scope for the
// checker but absent from the generated Go.
type HoleBinding struct {
	Name    string
	Type    string
	DepType string
	Erased  bool
}

// HoleInfo is the goal at one `?name`: what the hole must produce, and what
// is available to produce it with. Type is the erased expected type, empty
// when the context does not determine one; DepType is the dependent spelling
// when the hole sits in a dependent position.
type HoleInfo struct {
	Name     string
	Pos      token.Position
	Type     string
	DepType  string
	Bindings []HoleBinding
}

// String renders the goal: what the hole must produce, and what is in
// scope to produce it with. The head names the un-erased spelling when
// there is one; the erased line appears only when it differs.
func (h HoleInfo) String() string {
	var b strings.Builder
	switch {
	case h.DepType != "":
		fmt.Fprintf(&b, "hole ?%s : %s", h.Name, h.DepType)
		if h.Type != "" && h.Type != h.DepType {
			fmt.Fprintf(&b, "\n  erased: %s", h.Type)
		}
	case h.Type != "":
		fmt.Fprintf(&b, "hole ?%s : %s", h.Name, h.Type)
	default:
		fmt.Fprintf(&b, "hole ?%s : cannot infer a type from this context", h.Name)
		b.WriteString("\n  hint: annotate the binding this hole initializes; an inferred binding and other untyped positions carry no expectation")
	}
	if len(h.Bindings) > 0 {
		b.WriteString("\n  in scope:")
		for _, bind := range h.Bindings {
			text := bind.DepType
			if text == "" {
				text = bind.Type
			}
			if text == "" {
				text = "?"
			}
			fmt.Fprintf(&b, "\n    %s : %s", bind.Name, text)
			if bind.Erased {
				b.WriteString(" (erased, quantity 0)")
			}
		}
	}
	return b.String()
}

// Assumption is one `assume` — a proposition accepted on the author's
// authority rather than discharged by the decider. It is not an error,
// but it is the one place a dependent guarantee rests on a claim instead
// of a proof, so every use is recorded for review.
type Assumption struct {
	Pos         token.Position
	Fn          string // the declaration containing the call
	Callee      string // the function whose proof parameter was assumed
	Param       string // that parameter's name
	Proposition string // the proposition assumed, with call arguments substituted
}

func (a Assumption) String() string {
	return fmt.Sprintf("%s: assumed %s for %s of %s", a.Pos, a.Proposition, a.Param, a.Callee)
}

// SortAssumptions orders assumptions by position and removes duplicates.
func SortAssumptions(as []Assumption) []Assumption {
	sort.Slice(as, func(i, j int) bool {
		a, b := as[i], as[j]
		if a.Pos.Filename != b.Pos.Filename {
			return a.Pos.Filename < b.Pos.Filename
		}
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Column != b.Pos.Column {
			return a.Pos.Column < b.Pos.Column
		}
		return a.Proposition < b.Proposition
	})
	out := as[:0]
	for i, a := range as {
		if i == 0 || a != as[i-1] {
			out = append(out, a)
		}
	}
	return out
}

// Sort orders diagnostics by file, line, column, then message, and removes
// exact duplicates.
func Sort(ds []Diagnostic) []Diagnostic {
	sort.Slice(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Pos.Filename != b.Pos.Filename {
			return a.Pos.Filename < b.Pos.Filename
		}
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Column != b.Pos.Column {
			return a.Pos.Column < b.Pos.Column
		}
		return a.Msg < b.Msg
	})
	out := ds[:0]
	for i, d := range ds {
		if i == 0 || d != ds[i-1] {
			out = append(out, d)
		}
	}
	return out
}

// Render writes diagnostics one per line.
func Render(w io.Writer, ds []Diagnostic) {
	for _, d := range ds {
		fmt.Fprintln(w, d.String())
	}
}
