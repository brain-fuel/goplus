// Package sourcemap maps positions in emitted Go back to .gpp source.
//
// Because emission is text-edit-based (intra-line splices plus inserted
// header/marker lines) and then gofmt-formatted, a line-level diff between
// source and output recovers an accurate mapping: unchanged lines map
// exactly, spliced lines map with line fidelity, inserted lines attribute
// to the nearest preceding mapped line.
package sourcemap

import "go/token"

// Map maps one emitted file's positions to its .gpp source.
type Map struct {
	GppPath string
	// gppLine[i] is the 1-based .gpp line for emitted line i+1; 0 for
	// inserted lines with no counterpart.
	gppLine []int
	// exact[i]: the line text is identical, so columns carry over.
	exact []bool
}

// Build diffs gpp source against emitted output.
func Build(gppPath string, gppSrc, emitted []byte) *Map {
	a := splitLines(gppSrc)
	b := splitLines(emitted)
	m := &Map{GppPath: gppPath, gppLine: make([]int, len(b)), exact: make([]bool, len(b))}

	// LCS over lines; DP is fine at source-file scale.
	const cap = 5000
	if len(a) > cap || len(b) > cap {
		for i := range b {
			if i < len(a) {
				m.gppLine[i] = i + 1
			}
		}
		return m
	}
	lcs := make([][]int32, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			m.gppLine[j] = i + 1
			m.exact[j] = true
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			i++ // line deleted from source
		default:
			j++ // line inserted in output
		}
	}
	// Pair modified runs: an inserted output line directly after the last
	// matched pair inherits the corresponding replaced source line when a
	// deletion run aligns; approximate by attributing to the previous
	// mapped line's successor region.
	prev := 0
	for j := range m.gppLine {
		if m.gppLine[j] != 0 {
			prev = m.gppLine[j]
			continue
		}
		if prev > 0 && prev < len(a) {
			// best-effort: the line after the previous match
			m.gppLine[j] = prev + 1
			if m.gppLine[j] > len(a) {
				m.gppLine[j] = len(a)
			}
		}
	}
	return m
}

// Map translates an emitted-file position to a .gpp position. ok is false
// when the line has no plausible source counterpart (e.g. the header).
func (m *Map) Map(pos token.Position) (token.Position, bool) {
	if pos.Line < 1 || pos.Line > len(m.gppLine) || m.gppLine[pos.Line-1] == 0 {
		return token.Position{}, false
	}
	out := token.Position{
		Filename: m.GppPath,
		Line:     m.gppLine[pos.Line-1],
		Column:   pos.Column,
	}
	if !m.exact[pos.Line-1] && out.Column > 1 {
		// Spliced line: the column may not correspond; keep it as a hint.
	}
	return out, true
}

func splitLines(b []byte) []string {
	var lines []string
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			lines = append(lines, string(b[start:i]))
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, string(b[start:]))
	}
	return lines
}
