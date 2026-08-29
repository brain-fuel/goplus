package gen

import (
	"go/token"
	"strings"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/lower"
	"goforge.dev/goplus/internal/sourcemap"
)

// Pass 1 does not lower a Go+ construct to Go. It lowers it to a SKELETON
// plus a carrier that resolution is contracted to consume: a match becomes
// `case nil:` heads whose bodies are `//goplus:pattern` comments, a
// pipeline becomes a `__gp_bare_` call, an expression-position block
// becomes a `__gp_val` binding. Resolution needs type information, so it
// needs a module; without go.mod it never runs, and the skeleton was
// written out as if it were the finished artifact — invalid Go, from a
// command that exited 0.
//
// So the artifact is checked for the carriers themselves before it is
// written. This is the oracle the property suite already asserts over
// sampled programs (bddtest/steps_properties.go), promoted from a test to
// the write path, where it holds for every program rather than sampled
// ones. It is deliberately keyed off the reserved `__gp_` names and the
// pattern marker rather than the skeleton's shape: `case nil:` is a legal
// arm of a real Go type switch, and std/decimal writes one.
var resolveCarriers = []struct {
	token     string
	construct string
}{
	{lower.PatternCarrier, "a match"},
	{lower.BareCarrierPrefix, "a pipeline"},
	{lower.SegCarrierPrefix, "a pipeline"},
	{lower.DotCarrier, "a method-syntax call"},
	{lower.ComposeCarrier, "a composition"},
	{lower.KleisliCarrierPrefix, "a Kleisli composition"},
	{lower.ValCarrierPrefix, "an expression-position block"},
	{lower.HoleCarrierPrefix, "a hole"},
}

// residueDiags reports the unconsumed lowering carriers in one generated
// file, positioned on the .gp source they came from. Each construct is
// reported once — a single unresolved match is one problem, not one per
// binder it names — and at its first occurrence, which is the one the
// author can act on.
func residueDiags(gpPath string, gpSource, text []byte, haveModule bool) []diag.Diagnostic {
	var out []diag.Diagnostic
	var smap *sourcemap.Map
	seen := map[string]bool{}
	for number, line := range strings.Split(string(text), "\n") {
		for _, carrier := range resolveCarriers {
			if !strings.Contains(line, carrier.token) || seen[carrier.construct] {
				continue
			}
			seen[carrier.construct] = true
			if smap == nil {
				smap = sourcemap.Build(gpPath, gpSource, text)
			}
			pos := token.Position{Filename: gpPath, Line: number + 1, Column: 1}
			if mapped, ok := smap.Map(pos); ok {
				pos = firstCodeLineAt(gpSource, mapped)
			}
			if haveModule {
				out = append(out, diag.At(pos,
					"internal error: %s was lowered but never resolved, so the generated file would not be Go; please report this",
					carrier.construct))
				continue
			}
			out = append(out, diag.At(pos,
				"%s cannot be generated here: this package has no module context, so it cannot be resolved; run inside a module",
				carrier.construct))
		}
	}
	return out
}

// firstCodeLineAt moves a mapped position forward off a blank line. A
// carrier is generated-only text, so it has no exact counterpart in the
// .gp and lands wherever the map interpolates — sometimes in the gap
// between two declarations. The next line with code on it is the one the
// author would look at.
func firstCodeLineAt(gpSource []byte, pos token.Position) token.Position {
	lines := strings.Split(string(gpSource), "\n")
	for number := pos.Line; number >= 1 && number <= len(lines); number++ {
		if strings.TrimSpace(lines[number-1]) == "" {
			continue
		}
		pos.Line = number
		return pos
	}
	return pos
}
