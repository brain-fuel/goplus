package goml

import (
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/gen"
)

// RunOptions configures a goml generation run.
type RunOptions struct {
	Dir      string   // working directory; resolved against pattern paths
	Patterns []string // go-style package patterns; default ["./..."]
	Check    bool     // verify only: report stale outputs, write nothing
	Stage    bool     // after writing, git-add changed/deleted outputs
}

// RunResult reports what a goml run did.
type RunResult struct {
	Converted []string // .goml sources transpiled this run (relative)
	Gen       *gen.Result
}

// Ok reports whether the run completed without diagnostics.
func (r *RunResult) Ok() bool { return r.Gen == nil || r.Gen.Ok() }

// Run transpiles every matched .goml source to .gp text and generates
// the package through the ordinary goplus pipeline; foo.goml emits
// foo_gml.go beside it. Packages mixing .gp and .goml regenerate both
// surfaces (generation is package-wide).
func Run(opts RunOptions) (*RunResult, error) {
	res := &RunResult{Gen: &gen.Result{}}
	dirs, err := gen.ExpandPatterns(opts.Dir, opts.Patterns)
	if err != nil {
		return nil, err
	}
	overlay := map[string][]byte{}
	lineMaps := map[string]map[int]int{}
	holeMaps := map[string]map[string]Pos{}
	baseSeen := map[string]struct{}{}
	for _, dir := range dirs {
		entries, rerr := os.ReadDir(dir)
		if rerr != nil {
			res.Gen.Diags = append(res.Gen.Diags, diag.Errorf("%s: %v", dir, rerr))
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".goml") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				res.Gen.Diags = append(res.Gen.Diags, diag.Errorf("%s: %v", path, rerr))
				continue
			}
			gp, info, cerr := ConvertWithInfo(path, src)
			if cerr != nil {
				res.Gen.Diags = append(res.Gen.Diags, convertDiag(cerr))
				continue
			}
			abs, aerr := filepath.Abs(path)
			if aerr != nil {
				abs = path
			}
			overlay[abs] = gp
			lineMaps[path] = info.Lines
			lineMaps[abs] = info.Lines
			if len(info.Holes) > 0 {
				// The pipeline may name the file by any of these spellings
				// depending on how the run was invoked. The bare base name
				// is a last resort, so it is only trusted while it stays
				// unambiguous across this run.
				for _, key := range []string{path, abs, relTo(opts.Dir, path)} {
					holeMaps[key] = info.Holes
				}
				base := filepath.Base(path)
				if _, clash := baseSeen[base]; clash {
					delete(holeMaps, base)
				} else {
					holeMaps[base] = info.Holes
				}
				baseSeen[base] = struct{}{}
			}
			res.Converted = append(res.Converted, relTo(opts.Dir, path))
		}
	}
	sort.Strings(res.Converted)
	if len(res.Gen.Diags) > 0 {
		res.Gen.Diags = diag.Sort(res.Gen.Diags)
		return res, nil
	}
	// Run the pipeline even with an empty overlay: orphaned *_gml.go
	// files (their .goml deleted) are found and cleaned by generation.
	genRes, err := gen.Run(gen.Options{
		Dir:      opts.Dir,
		Patterns: opts.Patterns,
		Check:    opts.Check,
		Stage:    opts.Stage,
		Overlay:  overlay,
	})
	if err != nil {
		return nil, err
	}
	// A hole is named, so its position is exact: rewrite it and its
	// diagnostic to the `?` in the .goml source, and remember where the
	// pipeline had put it so the diagnostic can be recognized.
	holeFix := map[token.Position]token.Position{}
	for i, h := range genRes.Holes {
		exact, found := holeMaps[h.Pos.Filename][h.Name]
		if !found {
			exact, found = holeMaps[filepath.Base(h.Pos.Filename)][h.Name]
		}
		if !found {
			continue
		}
		fixed := h.Pos
		fixed.Line, fixed.Column = exact.Line, exact.Col
		holeFix[h.Pos] = fixed
		genRes.Holes[i].Pos = fixed
	}
	// Other diagnostics attributed to a .goml file carry positions in its
	// lowered .gp text; bring them back to the source line (the map is
	// decl/statement grained, so the column is dropped).
	for i, d := range genRes.Diags {
		if fixed, ok := holeFix[d.Pos]; ok && d.Kind == diag.KindHole {
			genRes.Diags[i].Pos = fixed
			continue
		}
		m, ok := lineMaps[d.Pos.Filename]
		if !ok {
			continue
		}
		if src := SourceLine(m, d.Pos.Line); src > 0 && src != d.Pos.Line {
			genRes.Diags[i].Pos.Line = src
			genRes.Diags[i].Pos.Column = 0
		}
	}
	res.Gen = genRes
	return res, nil
}

func convertDiag(err error) diag.Diagnostic {
	if e, ok := err.(*Error); ok {
		return diag.At(token.Position{Filename: e.Path, Line: e.Pos.Line, Column: e.Pos.Col}, "%s", e.Msg)
	}
	return diag.Errorf("%v", err)
}

func relTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}
