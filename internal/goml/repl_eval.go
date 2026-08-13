package goml

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goforge.dev/goplus/internal/gen"
	"goforge.dev/goplus/internal/toolchain"
)

const (
	sessionFile = "session.goml"
	mainFile    = "zz_repl_main.go"
	valueMarker = "\x00goml-value\x00"
	// The same marker spelled for embedding in generated Go source: a
	// literal NUL byte is not valid there.
	valueMarkerLit = `\x00goml-value\x00`
)

// generate materialises a candidate session and runs it through the
// ordinary pipeline, reporting diagnostics against what the user typed.
// It returns false when the input should be rejected.
func (r *repl) generate(s session, trailing string) bool {
	res, ok := r.tryGenerate(s, trailing)
	if ok {
		return true
	}
	if res != nil {
		r.reportDiags(s, r.lastRender, trailing, res)
	}
	return false
}

// tryGenerate runs the pipeline without reporting, so a caller can
// inspect the diagnostics and retry in another mode.
func (r *repl) tryGenerate(s session, trailing string) (*RunResult, bool) {
	src := s.render(replModule, trailing)
	path := filepath.Join(r.dir, sessionFile)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return nil, false
	}
	r.lastRender = src

	// A repeated render (e.g. :go after :list) needs no pipeline run.
	if src == r.lastGood {
		return &RunResult{Gen: &gen.Result{}}, true
	}
	res, err := Run(RunOptions{Dir: r.dir, Patterns: []string{"."}})
	if err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return nil, false
	}
	ok := len(res.Gen.Diags) == 0
	if ok {
		r.lastGood = src
	}
	return res, ok
}

// wantsStatement reports whether diagnostics say the expression has no
// single value — a call used for effect, or a multi-result Go call.
func wantsStatement(res *RunResult) bool {
	if res == nil {
		return false
	}
	for _, d := range res.Gen.Diags {
		msg := d.Msg
		if strings.Contains(msg, "used as value") ||
			strings.Contains(msg, "multiple-value") ||
			strings.Contains(msg, "(no value)") {
			return true
		}
	}
	return false
}

// reportDiags renders pipeline diagnostics against the user's input,
// naming the retained declaration when an older binding is what broke.
func (r *repl) reportDiags(s session, src, trailing string, res *RunResult) {
	spans := declSpans(s, src, trailing)
	for _, d := range res.Gen.Diags {
		where := ""
		rel := d.Pos.Line
		if d.Pos.Line > 0 {
			if name, start, ok := spans.find(d.Pos.Line); ok {
				// Only an OLDER declaration gets attributed; the input
				// being evaluated is what the user is looking at.
				if name != "" && !r.pending[name] {
					where = fmt.Sprintf("in %s (defined earlier): ", name)
				}
				rel = d.Pos.Line - start + 1
			}
		}
		if rel > 0 {
			fmt.Fprintf(r.errOut, "%s<stdin>:%d: %s\n", where, rel, d.Msg)
			continue
		}
		fmt.Fprintf(r.errOut, "%s%s\n", where, d.Msg)
	}
}

// span maps a line range of the rendered session to its origin.
type span struct {
	name       string // "" for the current input
	start, end int
}

type spanSet []span

func (ss spanSet) find(line int) (string, int, bool) {
	for _, s := range ss {
		if line >= s.start && line <= s.end {
			return s.name, s.start, true
		}
	}
	return "", 0, false
}

// declSpans recomputes where each declaration landed in the rendered
// session, so a diagnostic line can be attributed to its source.
func declSpans(s session, src, trailing string) spanSet {
	var out spanSet
	lines := strings.Split(src, "\n")
	cursor := 0
	find := func(text string) (int, int, bool) {
		want := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if len(want) == 0 {
			return 0, 0, false
		}
		for i := cursor; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == strings.TrimSpace(want[0]) {
				cursor = i + len(want)
				return i + 1, i + len(want), true
			}
		}
		return 0, 0, false
	}
	for _, d := range s.decls {
		if start, end, ok := find(d.src); ok {
			out = append(out, span{name: d.name, start: start, end: end})
		}
	}
	if trailing != "" {
		if start, end, ok := find(trailing); ok {
			out = append(out, span{start: start, end: end})
		}
	}
	return out
}

// runValue compiles and runs the session, printing the bound value. The
// generated main writes a marker before the value so output the program
// itself produced stays distinguishable and in order.
func (r *repl) runValue(next session, name, trailing string) {
	gp, err := os.ReadFile(filepath.Join(r.dir, "session_gml.go"))
	if err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return
	}
	// A value binding lowers to `var name`; a procedure to `func name()`.
	call := name
	if bytes.Contains(gp, []byte("func "+name+"(")) {
		call = name + "()"
	}

	var main string
	if strings.Contains(trailing, ": Unit :=") {
		main = fmt.Sprintf(`package main

func main() {
	%s
}
`, call)
	} else {
		main = fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Printf("%s%%+v\n", %s)
}
`, valueMarkerLit, call)
	}
	mainPath := filepath.Join(r.dir, mainFile)
	if err := os.WriteFile(mainPath, []byte(main), 0o644); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return
	}
	defer os.Remove(mainPath)

	ctx, cancel := context.WithCancel(context.Background())
	if r.opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), r.opts.Timeout)
	}
	defer cancel()
	r.mu.Lock()
	r.cancelRun = cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.cancelRun = nil
		r.mu.Unlock()
	}()
	var stdout, stderr bytes.Buffer
	code := toolchain.GoWith(ctx, r.dir, "run", []string{"."}, r.goEnv(), nil, &stdout, &stderr)
	switch ctx.Err() {
	case context.DeadlineExceeded:
		fmt.Fprintf(r.errOut, "evaluation timed out after %s\n", r.opts.Timeout)
		return
	case context.Canceled:
		return // interrupted; the session is untouched
	}
	if code != 0 {
		r.reportRunFailure(stdout.String(), stderr.String())
		return
	}
	r.printRun(stdout.String())
	r.sess = next
	// Expression results are never retained as themselves — that is what
	// keeps a bare effectful call running exactly once. `it` is the one
	// exception, and only for a pure-looking value expression.
	r.sess.dropDecl(name)
	if !strings.Contains(trailing, ": Unit :=") {
		expr := strings.TrimPrefix(trailing, "let "+name+" := ")
		r.sess.lastExp = expr
		r.bindIt(expr)
	}
}

// printRun splits the program's own output from the evaluated value.
func (r *repl) printRun(out string) {
	if i := strings.LastIndex(out, valueMarker); i >= 0 {
		if pre := out[:i]; pre != "" {
			fmt.Fprint(r.out, pre)
		}
		fmt.Fprint(r.out, out[i+len(valueMarker):])
		return
	}
	fmt.Fprint(r.out, out)
}

// reportRunFailure distinguishes a program panic from a REPL bug: the
// pipeline already type-checked, so a compile error here is ours.
func (r *repl) reportRunFailure(stdout, stderr string) {
	if stdout != "" {
		r.printRun(stdout)
	}
	switch {
	case strings.Contains(stderr, "panic:"):
		for _, line := range strings.Split(stderr, "\n") {
			if strings.Contains(line, r.dir) || strings.Contains(line, replSuffix) {
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			fmt.Fprintln(r.errOut, line)
			if strings.HasPrefix(line, "panic:") {
				break
			}
		}
	case strings.Contains(stderr, "no required module provides package"):
		fmt.Fprintf(r.errOut, "that package is not available in this session; start the REPL with -std <dir> for the goplus standard library\n")
	default:
		fmt.Fprintf(r.errOut, "internal REPL error (please report): the pipeline accepted this but the go tool did not:\n%s\nsession: %s\n", strings.TrimSpace(stderr), r.dir)
	}
}

// typeOf reports the Go type of an expression by type-checking the
// generated package. Dependent indices are erased in generated Go, so
// the answer is the erased type; :gp shows the lowered signature.
func (r *repl) typeOf(input string) {
	next := r.sess.clone()
	r.autoImport(&next, input)
	r.seq++
	name := fmt.Sprintf("%s%d", replSuffix, r.seq)
	if !r.generate(next, fmt.Sprintf("let %s := %s", name, input)) {
		return
	}
	typ, err := lookupType(r.dir, name)
	if err != nil {
		fmt.Fprintf(r.errOut, "%v\n", err)
		return
	}
	if !r.typeCaveat {
		fmt.Fprintln(r.out, "-- Go type: dependent indices are erased in generated Go (Vec a (n+1) reads as")
		fmt.Fprintln(r.out, "-- Vec[a]), quantities are gone, and a refinement shows as its base type.")
		fmt.Fprintln(r.out, "-- :gp shows the lowered .gp, which keeps the index annotations.")
		r.typeCaveat = true
	}
	fmt.Fprintln(r.out, typ)
}

// itLimit caps how large an expanded `it` may grow before the REPL stops
// carrying it forward.
const itLimit = 4096

// bindIt records the last value expression as `it`. The previous `it` is
// expanded inline rather than referenced: a binding that refers to its
// own earlier definition is an initialization cycle, since the new
// definition replaces the old one under the same name.
func (r *repl) bindIt(expr string) {
	prev := ""
	if d, ok := r.sess.lookup("it"); ok {
		prev = strings.TrimPrefix(d.src, "let it := ")
	}
	expanded := expr
	if prev != "" && referencesIt(expr) {
		expanded = substituteIt(expr, "("+prev+")")
	}
	if len(expanded) > itLimit {
		r.sess.dropDecl("it")
		fmt.Fprintln(r.out, "(it is no longer carried forward: the expression it accumulated grew too large)")
		return
	}
	parsed, perr := Parse("<it>", []byte("module "+replModule+"\nlet it := "+expanded+"\n"))
	if perr != nil || len(parsed.Decls) != 1 {
		r.sess.dropDecl("it")
		return
	}
	ld, isLet := parsed.Decls[0].(*LetDecl)
	if !isLet || looksEffectful(ld.Body) {
		// An effectful expression would re-run on every later evaluation.
		r.sess.dropDecl("it")
		return
	}
	r.sess.addDecl(sessionDecl{name: "it", kind: kLet, src: "let it := " + expanded})
}

// referencesIt reports whether an expression uses the bare name `it`.
func referencesIt(expr string) bool {
	toks, err := newLexer("<it>", []byte(expr)).tokens()
	if err != nil {
		return false
	}
	for i, t := range toks {
		if t.Kind == IDENT && t.Text == "it" && (i == 0 || toks[i-1].Kind != Dot) {
			return true
		}
	}
	return false
}

// substituteIt replaces every bare `it` with the given text.
func substituteIt(expr, with string) string {
	toks, err := newLexer("<it>", []byte(expr)).tokens()
	if err != nil {
		return expr
	}
	var b strings.Builder
	lines := strings.Split(expr, "\n")
	prevLine, prevCol := 1, 1
	emitGap := func(to Pos) {
		for prevLine < to.Line {
			if prevLine-1 < len(lines) {
				b.WriteString(lines[prevLine-1][min(prevCol-1, len(lines[prevLine-1])):])
			}
			b.WriteString("\n")
			prevLine++
			prevCol = 1
		}
		if prevLine-1 < len(lines) {
			line := lines[prevLine-1]
			b.WriteString(line[min(prevCol-1, len(line)):min(to.Col-1, len(line))])
		}
		prevCol = to.Col
	}
	for i, t := range toks {
		if t.Kind == EOF {
			break
		}
		if t.Kind == IDENT && t.Text == "it" && (i == 0 || toks[i-1].Kind != Dot) {
			emitGap(t.Pos)
			b.WriteString(with)
			prevCol = t.Pos.Col + len("it")
		}
	}
	// Tail of the final line, then any remaining lines.
	if prevLine-1 < len(lines) {
		line := lines[prevLine-1]
		b.WriteString(line[min(prevCol-1, len(line)):])
	}
	for i := prevLine; i < len(lines); i++ {
		b.WriteString("\n" + lines[i])
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
