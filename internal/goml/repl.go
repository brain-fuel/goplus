package goml

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"goforge.dev/goplus/internal/version"
)

// REPLOptions configures an interactive session.
type REPLOptions struct {
	Dir     string        // session directory; "" creates (and removes) a temp one
	Keep    bool          // keep the session directory and report its path
	Std     string        // goforge.dev/goplus/std checkout to make importable
	Timeout time.Duration // per-evaluation timeout for the compiled program
	Offline bool          // forbid module downloads
	Env     []string      // extra environment for the go tool
}

const replModule = "main"

// replSuffix names the synthesized binding an expression is evaluated
// through; the leading underscore keeps it out of the user's namespace.
const replSuffix = "_replValue"

type repl struct {
	out, errOut io.Writer
	opts        REPLOptions
	dir         string
	cleanup     func()
	sess        session
	seq         int
	interactive bool
	typeCaveat  bool // :type's erasure caveat has been shown
	lastRender  string
	pending     map[string]bool // names introduced by the input being evaluated
	lastGood    string          // last render that generated cleanly
	mu          sync.Mutex
	cancelRun   context.CancelFunc // cancels the evaluation in flight
	warmed      bool
}

// REPL runs an interactive goml session, returning a process exit code.
//
// goml has no interpreter: every evaluation transpiles the session,
// generates Go through the ordinary pipeline, and runs the go tool. The
// consequence users must know is that retained bindings re-execute on
// every evaluation.
func REPL(in io.Reader, out, errOut io.Writer, opts REPLOptions) int {
	r := &repl{out: out, errOut: errOut, opts: opts}
	if code := r.setup(); code != 0 {
		return code
	}
	defer r.finish()

	r.interactive = isTerminal(in)
	if r.interactive {
		fmt.Fprintf(r.out, "goml %s — :help for commands, :quit to leave\n", version.Version)
	}
	lines := newLineReader(in, r.out, r.interactive)
	defer lines.Close()
	if lines.Raw() {
		// Raw mode disables the driver's newline translation, so every
		// write of ours has to carry the carriage return itself.
		r.out = crlf{r.out}
		r.errOut = crlf{r.errOut}
	}

	// Interrupts kill the compiled program, not the session: a runaway
	// evaluation is recoverable without losing what has been defined.
	stopSignals := r.watchInterrupts()
	defer stopSignals()

	// The first evaluation pays for a cold build cache; warm it while
	// the user is still typing.
	go r.prewarm()

	var buf []string
	block := false
	for {
		prompt := "goml> "
		if len(buf) > 0 {
			prompt = "....> "
		}
		line, err := lines.ReadLine(prompt)
		if err != nil {
			if r.interactive {
				fmt.Fprintln(r.out)
			}
			return 0
		}
		blank := strings.TrimSpace(line) == ""
		if len(buf) == 0 && blank {
			continue
		}
		// An explicit block runs from :{ to :}, whatever it parses as.
		if len(buf) == 0 && strings.TrimSpace(line) == ":{" {
			block = true
			continue
		}
		if block {
			if strings.TrimSpace(line) == ":}" {
				block = false
			} else {
				buf = append(buf, line)
				continue
			}
		} else {
			buf = append(buf, line)
			// The first line of a continuation keeps reading while the
			// input is merely incomplete. Once continuing, only a blank
			// line submits: a clausal definition parses after its first
			// clause, but the user is probably still typing clauses.
			if len(buf) == 1 {
				if r.incomplete(strings.Join(buf, "\n")) {
					continue
				}
			} else if !blank {
				continue
			}
		}
		input := strings.TrimSpace(strings.Join(buf, "\n"))
		buf = nil
		if input == "" {
			continue
		}
		if r.dispatch(input) {
			return 0
		}
	}
}

// setup creates the session directory and its module.
func (r *repl) setup() int {
	dir := r.opts.Dir
	if dir == "" {
		tmp, err := os.MkdirTemp("", "goml-repl-")
		if err != nil {
			fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
			return 2
		}
		// Resolve symlinks: on macOS /var is a link to /private/var, and
		// the go tool reports the resolved path in diagnostics.
		if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
			tmp = resolved
		}
		dir = tmp
		if !r.opts.Keep {
			r.cleanup = func() { os.RemoveAll(dir) }
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return 2
	}
	r.dir = dir

	mod := "module example.com/gomlrepl\n\ngo 1.26.0\n"
	if r.opts.Std != "" {
		std, err := filepath.Abs(r.opts.Std)
		if err != nil {
			fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
			return 2
		}
		mod += fmt.Sprintf("\nrequire goforge.dev/goplus/std v0.0.0\n\nreplace goforge.dev/goplus/std => %s\n", std)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: %v\n", err)
		return 2
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintf(r.errOut, "goml repl: the go tool is required to evaluate expressions: %v\n", err)
		return 2
	}
	return 0
}

// watchInterrupts routes SIGINT to the running child, if any.
func (r *repl) watchInterrupts() func() {
	if !r.interactive {
		return func() {}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				r.mu.Lock()
				cancel := r.cancelRun
				r.mu.Unlock()
				if cancel != nil {
					cancel()
					fmt.Fprintln(r.out, "\ninterrupted")
					continue
				}
				fmt.Fprintln(r.out, "\n(nothing running — :quit to leave)")
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}

// prewarm compiles a trivial session so the first real evaluation does
// not pay for a cold toolchain and linker cache. It runs in its own
// directory: the build cache it warms is global, and sharing the live
// session directory would race with the user's own input.
func (r *repl) prewarm() {
	dir, err := os.MkdirTemp("", "goml-warm-")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)
	if resolved, rerr := filepath.EvalSymlinks(dir); rerr == nil {
		dir = resolved
	}
	mod := "module example.com/gomlwarm\n\ngo 1.26.0\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		return
	}
	var sink strings.Builder
	warm := &repl{out: &sink, errOut: &sink, opts: REPLOptions{Offline: r.opts.Offline}, dir: dir}
	warm.evalExpr("0")
}

func (r *repl) finish() {
	if r.opts.Keep && r.dir != "" {
		fmt.Fprintf(r.out, "session kept at %s\n", r.dir)
	}
	if r.cleanup != nil {
		r.cleanup()
	}
}

// incomplete reports whether input is a prefix of a valid declaration
// rather than a mistake. The signal is positional: a parse error whose
// position is the very end of the input means the parser ran out of
// tokens. Unterminated comments and strings report their opening
// delimiter, so they are named explicitly.
func (r *repl) incomplete(input string) bool {
	candidate := "module " + replModule + "\n" + r.wrap(input)
	_, err := Parse("<stdin>", []byte(candidate))
	if err == nil {
		return false
	}
	gerr, ok := err.(*Error)
	if !ok {
		return false
	}
	if strings.Contains(gerr.Msg, "unterminated") {
		return true
	}
	end := eofPos([]byte(candidate))
	return gerr.Pos == end
}

// wrap renders an input the way the session will, so the incompleteness
// probe parses exactly what evaluation would.
func (r *repl) wrap(input string) string {
	if classify(input) == inputExpr {
		return fmt.Sprintf("let %s%d := %s\n", replSuffix, r.seq, input)
	}
	return input + "\n"
}

// eofPos is the position the lexer reports for end of input: one past
// the last rune, counting lines the same way.
func eofPos(src []byte) Pos {
	line, col := 1, 1
	for i := 0; i < len(src); {
		rn, w := utf8.DecodeRune(src[i:])
		if rn == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i += w
	}
	return Pos{Line: line, Col: col}
}

type inputKind int

const (
	inputExpr inputKind = iota
	inputDecl
	inputImport
	inputModule
)

// classify decides what an input is by its leading significant token.
func classify(input string) inputKind {
	toks, err := newLexer("<stdin>", []byte(input)).tokens()
	if err != nil || len(toks) == 0 {
		return inputExpr
	}
	i := 0
	for i < len(toks) && toks[i].Kind == COMMENT {
		i++
	}
	if i >= len(toks) {
		return inputExpr
	}
	switch toks[i].Kind {
	case KwImport, KwOpen:
		return inputImport
	case KwModule:
		return inputModule
	case KwLet:
		// `let` at the head is always a declaration; let-in needs parens.
		return inputDecl
	case KwTotal, KwType, KwClass, KwInstance, KwNamespace, At:
		return inputDecl
	}
	return inputExpr
}

// dispatch handles one complete input, returning true to exit.
func (r *repl) dispatch(input string) bool {
	if input == "" {
		return false
	}
	if strings.HasPrefix(input, ":") {
		return r.command(input)
	}
	switch classify(input) {
	case inputModule:
		fmt.Fprintln(r.errOut, "the REPL's module is fixed; use :save <file.goml> to write a named module")
	case inputImport:
		r.addImportInput(input)
	case inputDecl:
		r.evalDecl(input)
	default:
		r.evalExpr(input)
	}
	return false
}

// addImportInput records an `import "path" [as alias]` line.
func (r *repl) addImportInput(input string) {
	file, err := Parse("<stdin>", []byte("module "+replModule+"\n"+input+"\n"))
	if err != nil {
		r.reportParse(err, input)
		return
	}
	for _, imp := range file.Imports {
		r.sess.addImport(sessionImport{path: imp.Path, alias: imp.Alias})
	}
}

// evalDecl type-checks a declaration and retains it. Declarations run no
// go command: the pipeline's backstop is a full go/types check, so
// anything that generates cleanly compiles.
func (r *repl) evalDecl(input string) {
	file, err := Parse("<stdin>", []byte("module "+replModule+"\n"+input+"\n"))
	if err != nil {
		r.reportParse(err, input)
		return
	}
	if len(file.Decls) == 0 {
		fmt.Fprintln(r.errOut, "no declaration found")
		return
	}
	next := r.sess.clone()
	r.autoImport(&next, input)
	var added []sessionDecl
	for _, d := range file.Decls {
		sd := sessionDecl{src: input}
		switch d := d.(type) {
		case *LetDecl:
			sd.name, sd.kind = d.Name, kLet
			sd.effectful = looksEffectful(d.Body)
		case *TypeDecl:
			sd.name, sd.kind = d.Name, kType
		case *ClassDecl:
			sd.name, sd.kind = d.Name, kClass
		case *InstanceDecl:
			sd.name, sd.kind = d.Name, kInstance
		case *NamespaceDecl:
			sd.name, sd.kind = d.Name, kNamespace
		}
		if len(file.Decls) > 1 {
			sd.src = "" // multi-decl input is retained once, below
		}
		added = append(added, sd)
	}
	// Keep a multi-declaration input as one unit under the first name.
	if len(added) > 1 {
		added = added[:1]
		added[0].src = input
	}
	r.pending = map[string]bool{}
	for _, sd := range added {
		next.addDecl(sd)
		r.pending[sd.name] = true
	}
	defer func() { r.pending = nil }()
	if !r.generate(next, "") {
		return
	}
	r.sess = next
	for _, sd := range added {
		if sd.effectful {
			fmt.Fprintf(r.out, "note: %s re-runs on every evaluation (its body looks effectful); :drop %s to remove it\n", sd.name, sd.name)
		}
	}
}

// evalExpr evaluates an expression and prints its value.
func (r *repl) evalExpr(input string) {
	next := r.sess.clone()
	r.autoImport(&next, input)
	r.seq++
	name := fmt.Sprintf("%s%d", replSuffix, r.seq)

	// Value mode: bind the expression so Go infers its type, then print.
	// An input with no single value (a do block, an effectful call, a
	// multi-result Go call) falls back to a procedure.
	trailing := fmt.Sprintf("let %s := %s", name, input)
	if isStatementInput(input) {
		trailing = fmt.Sprintf("let %s () : Unit := %s", name, input)
		if !r.generate(next, trailing) {
			return
		}
	} else {
		res, ok := r.tryGenerate(next, trailing)
		if !ok && wantsStatement(res) {
			trailing = fmt.Sprintf("let %s () : Unit := %s", name, input)
			if !r.generate(next, trailing) {
				return
			}
		} else if !ok {
			if res != nil {
				r.reportDiags(next, r.lastRender, trailing, res)
			}
			return
		}
	}
	r.runValue(next, name, trailing)
}

// isStatementInput reports whether an input is a do block, which has no
// value and must be lowered as a procedure.
func isStatementInput(input string) bool {
	toks, err := newLexer("<stdin>", []byte(input)).tokens()
	if err != nil || len(toks) == 0 {
		return false
	}
	return toks[0].Kind == KwDo
}

func (r *repl) autoImport(s *session, input string) {
	for _, imp := range s.missingImports(input) {
		s.addImport(imp)
		fmt.Fprintf(r.out, "(imported %s)\n", imp.path)
	}
}

func (r *repl) reportParse(err error, input string) {
	gerr, ok := err.(*Error)
	if !ok {
		fmt.Fprintf(r.errOut, "%v\n", err)
		return
	}
	// The probe prepends one `module` line; report the user's own line.
	line := gerr.Pos.Line - 1
	fmt.Fprintf(r.errOut, "<stdin>:%d:%d: %s\n", line, gerr.Pos.Col, gerr.Msg)
	lines := strings.Split(input, "\n")
	if line >= 1 && line <= len(lines) {
		fmt.Fprintf(r.errOut, "  | %s\n", lines[line-1])
		fmt.Fprintf(r.errOut, "  | %s^\n", strings.Repeat(" ", max(gerr.Pos.Col-1, 0)))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// goEnv is the environment every child go command runs with. GOWORK=off
// keeps a stray go.work above TMPDIR from poisoning the session.
func (r *repl) goEnv() []string {
	env := append([]string{"GOWORK=off", "GOFLAGS=-mod=mod"}, r.opts.Env...)
	if r.opts.Offline {
		env = append(env, "GOPROXY=off")
	}
	return env
}
