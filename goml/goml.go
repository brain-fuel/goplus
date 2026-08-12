// Package goml is the public facade of the goml front end: an ML-family
// surface (SML/OCaml/Idris2/Lean4 flavored) for the Go+ core, specified
// in spec/goml-design.md. A .goml source transpiles to .gp text and
// generates through the ordinary goplus pipeline, emitting <file>_gml.go
// beside <file>.goml — committed, ordinary Go that consumers use without
// either toolchain.
package goml

import (
	"goforge.dev/goplus/internal/goml"
	"goforge.dev/goplus/internal/version"
)

// Options configures a generation run.
type Options struct {
	Dir      string   // working directory; resolved against pattern paths
	Patterns []string // go-style package patterns; default ["./..."]
	Check    bool     // verify only: report stale outputs, write nothing
	Stage    bool     // after writing, git-add changed/deleted outputs
}

// Diagnostic is one user-facing error, positioned in source when
// attributable. Positions in transpiled packages reference the .goml
// file's lowered .gp text (a v0 limitation).
type Diagnostic struct {
	Filename     string
	Line, Column int
	Message      string
}

// Result reports what a run did (paths relative to Options.Dir when
// under it).
type Result struct {
	Converted   []string // .goml sources transpiled this run
	Written     []string // files written (or deleted orphans)
	Stale       []string // check mode: outputs missing or out of date
	Orphans     []string // generated files whose source is gone
	Diagnostics []Diagnostic
}

// Ok reports whether generation completed without diagnostics.
func (r *Result) Ok() bool { return len(r.Diagnostics) == 0 }

// Run transpiles every matched .goml source and generates its package.
// Packages mixing .gp and .goml regenerate both surfaces (generation is
// package-wide).
func Run(opts Options) (*Result, error) {
	inner, err := goml.Run(goml.RunOptions(opts))
	if err != nil {
		return nil, err
	}
	res := &Result{Converted: inner.Converted}
	if g := inner.Gen; g != nil {
		res.Written = g.Written
		res.Stale = g.Stale
		res.Orphans = g.Orphans
		for _, d := range g.Diags {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Filename: d.Pos.Filename,
				Line:     d.Pos.Line,
				Column:   d.Pos.Column,
				Message:  d.Msg,
			})
		}
	}
	return res, nil
}

// Convert transpiles one .goml source to .gp text without generating.
func Convert(path string, src []byte) ([]byte, error) {
	return goml.Convert(path, src)
}

// Version reports the goplus toolchain version goml ships with.
func Version() string { return version.Version }
