package lsp

import (
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go/token"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/goml"
	"goforge.dev/goplus/internal/sourcemap"
	"goforge.dev/goplus/internal/version"
)

// Server holds one editing session: unsaved buffers, the latest
// in-memory generation, forward/backward sourcemaps, and the gopls
// delegate. Effects called from the goplus dispatch layer live here.
type Server struct {
	conn *conn
	root string // workspace directory

	mu       sync.Mutex
	overlays map[string][]byte // abs .gp path → unsaved contents
	outputs  map[string][]byte // abs generated path → latest dry-run output
	maps     map[string]*sourcemap.Map // generated path → map (both directions)
	holes    map[string][]diag.HoleInfo // abs source path → open typed holes

	debounce *time.Timer
	genMu    sync.Mutex

	gopls *goplsClient // nil when unavailable
}

// Serve speaks LSP over r/w until EOF or exit.
func Serve(r io.Reader, w io.Writer) error {
	s := &Server{
		conn:     newConn(r, w),
		overlays: map[string][]byte{},
		outputs:  map[string][]byte{},
		maps:     map[string]*sourcemap.Map{},
		holes:    map[string][]diag.HoleInfo{},
	}
	for {
		m, err := s.conn.read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if done := s.handle(m); done {
			return nil
		}
	}
}

func (s *Server) handle(m *message) (exit bool) {
	switch m.Method {
	case "initialize":
		var p initializeParams
		_ = json.Unmarshal(m.Params, &p)
		s.root = uriToPath(p.RootURI)
		s.gopls = startGopls(s.root)
		caps := map[string]any{
			"textDocumentSync": map[string]any{"openClose": true, "change": 1, "save": true},
			// Hover is answered natively for typed holes, so it is offered
			// whether or not the gopls delegate started.
			"hoverProvider": true,
		}
		if s.gopls != nil {
			caps["definitionProvider"] = true
			caps["completionProvider"] = map[string]any{"triggerCharacters": []string{"."}}
		}
		_ = s.conn.respond(m.ID, initializeResult{
			Capabilities: caps,
			ServerInfo:   serverInfo{Name: "goplus", Version: version.Version},
		})
	case "initialized":
	case "shutdown":
		_ = s.conn.respond(m.ID, nil)
	case "exit":
		if s.gopls != nil {
			s.gopls.stop()
		}
		return true
	case "textDocument/didOpen":
		var p didOpenParams
		_ = json.Unmarshal(m.Params, &p)
		Dispatch(s, Opened(p.TextDocument.URI, p.TextDocument.Text))
	case "textDocument/didChange":
		var p didChangeParams
		_ = json.Unmarshal(m.Params, &p)
		if len(p.Changes) > 0 {
			Dispatch(s, Changed(p.TextDocument.URI, p.Changes[len(p.Changes)-1].Text))
		}
	case "textDocument/didClose":
		var p didCloseParams
		_ = json.Unmarshal(m.Params, &p)
		Dispatch(s, Closed(p.TextDocument.URI))
	case "textDocument/didSave":
		var p didCloseParams
		_ = json.Unmarshal(m.Params, &p)
		Dispatch(s, Saved(p.TextDocument.URI))
	case "textDocument/hover", "textDocument/definition", "textDocument/completion":
		var p positionParams
		_ = json.Unmarshal(m.Params, &p)
		var q Query
		switch m.Method {
		case "textDocument/hover":
			q = QHover(p.TextDocument.URI, p.Position.Line, p.Position.Character)
		case "textDocument/definition":
			q = QDefinition(p.TextDocument.URI, p.Position.Line, p.Position.Character)
		default:
			q = QComplete(p.TextDocument.URI, p.Position.Line, p.Position.Character)
		}
		match := Answer(s, q)
		s.replyQuery(m.ID, match)
	default:
		if m.ID != nil {
			_ = s.conn.respondErr(m.ID, -32601, "method not supported: "+m.Method)
		}
	}
	return false
}

func (s *Server) replyQuery(id *json.RawMessage, r QueryReply) {
	_ = QueryReplyFold(r, QueryReplyCases[error]{
		Raw: func(data []byte) error {
			raw := json.RawMessage(data)
			return s.conn.respond(id, &raw)
		},
		NoAnswer: func(reason string) error {
			_ = reason
			return s.conn.respond(id, nil)
		},
	})
}

// SetOverlay records an unsaved buffer.
func (s *Server) SetOverlay(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overlays[uriToPath(uri)] = []byte(text)
}

// DropOverlay forgets a closed buffer.
func (s *Server) DropOverlay(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.overlays, uriToPath(uri))
}

// ScheduleDiagnostics debounces a workspace regeneration.
func (s *Server) ScheduleDiagnostics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debounce != nil {
		s.debounce.Stop()
	}
	s.debounce = time.AfterFunc(200*time.Millisecond, s.runDiagnostics)
}

// runDiagnostics regenerates in memory and publishes per-file errors.
func (s *Server) runDiagnostics() {
	s.genMu.Lock()
	defer s.genMu.Unlock()
	s.mu.Lock()
	overlay := make(map[string][]byte, len(s.overlays))
	openFiles := make([]string, 0, len(s.overlays))
	for p, b := range s.overlays {
		overlay[p] = b
		openFiles = append(openFiles, p)
	}
	s.mu.Unlock()

	// goml.Run transpiles any .goml buffers and brings their diagnostics
	// back to .goml positions; with none open it delegates to the same
	// gen.Run this used to call directly.
	gomlRes, err := goml.Run(goml.RunOptions{Dir: s.root, Patterns: []string{"./..."}, Overlay: overlay, DryRun: true})
	if err != nil {
		return
	}
	res := gomlRes.Gen
	byFile := map[string][]Diagnostic{}
	for _, f := range openFiles {
		byFile[f] = nil // clear stale diagnostics on every open file
	}
	holes := map[string][]diag.HoleInfo{}
	for _, h := range res.Holes {
		path := h.Pos.Filename
		if !filepath.IsAbs(path) {
			path = filepath.Join(s.root, path)
		}
		holes[path] = append(holes[path], h)
	}
	s.mu.Lock()
	s.holes = holes
	s.mu.Unlock()
	for _, d := range res.Diags {
		path := d.Pos.Filename
		if !filepath.IsAbs(path) {
			path = filepath.Join(s.root, path)
		}
		line, col := d.Pos.Line-1, d.Pos.Column-1
		if line < 0 {
			line, col = 0, 0
		}
		if col < 0 {
			col = 0
		}
		// A hole's goal is information, not a mistake — but it still
		// stops generation, so it is still reported.
		severity := 1
		if d.Kind == diag.KindHole {
			severity = 3
		}
		byFile[path] = append(byFile[path], Diagnostic{
			Range:    Range{Start: Position{line, col}, End: Position{line, col + 1}},
			Severity: severity,
			Source:   "goplus",
			Message:  d.Msg,
		})
	}
	for path, diags := range byFile {
		if diags == nil {
			diags = []Diagnostic{}
		}
		_ = s.conn.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{
			URI: pathToURI(path), Diagnostics: diags,
		})
	}

	// A clean generation refreshes the delegate's view of the world.
	if len(res.Diags) == 0 && res.Outputs != nil {
		s.mu.Lock()
		s.outputs = res.Outputs
		s.maps = map[string]*sourcemap.Map{}
		for genPath, content := range res.Outputs {
			goplusPath := sourceCounterpart(genPath)
			goplusSrc, ok := overlay[goplusPath]
			if !ok {
				continue
			}
			// A .goml file's Go is generated from its transpiled .gp text,
			// so a direct source-to-output line diff would be meaningless.
			// Delegated hover stays .gp-only until that two-hop map exists;
			// hole goals need no map and work in both surfaces.
			if strings.HasSuffix(goplusPath, ".goml") {
				continue
			}
			s.maps[genPath] = sourcemap.Build(goplusPath, goplusSrc, content)
		}
		s.mu.Unlock()
		if s.gopls != nil {
			s.gopls.syncOutputs(res.Outputs)
		}
	}
}

// HoverHole answers a hover that lands on a typed hole. A hole has no
// counterpart in generated Go — it is why there is no generated Go — so
// the goal is served from here rather than forwarded to the delegate.
func (s *Server) HoverHole(uri string, line, ch int) QueryReply {
	path := uriToPath(uri)
	s.mu.Lock()
	holes := s.holes[path]
	s.mu.Unlock()
	for _, h := range holes {
		// Positions are 1-based in diagnostics, 0-based in LSP. The hole
		// spans its `?` and name.
		if h.Pos.Line-1 != line {
			continue
		}
		start := h.Pos.Column - 1
		if ch < start || ch > start+len(h.Name) {
			continue
		}
		body, err := json.Marshal(map[string]any{
			"contents": map[string]any{
				"kind":  "markdown",
				"value": "```\n" + h.String() + "\n```",
			},
			"range": Range{
				Start: Position{line, start},
				End:   Position{line, start + len(h.Name) + 1},
			},
		})
		if err != nil {
			return NoAnswer(err.Error())
		}
		return Raw(body)
	}
	return NoAnswer("not a hole")
}

// Forward maps a .gp position through the latest generation, asks
// gopls, and maps result positions back.
func (s *Server) Forward(method, uri string, line, ch int) QueryReply {
	if s.gopls == nil {
		return NoAnswer("gopls unavailable")
	}
	goplusPath := uriToPath(uri)
	genPath := generatedCounterpart(goplusPath)
	s.mu.Lock()
	m := s.maps[genPath]
	s.mu.Unlock()
	if m == nil {
		return NoAnswer("no generation for " + goplusPath)
	}
	fwd, ok := forwardPos(m, line, ch)
	if !ok {
		return NoAnswer("position not generated")
	}
	raw, err := s.gopls.request(method, positionParams{
		TextDocument: textDocumentIdentifier{URI: pathToURI(genPath)},
		Position:     fwd,
	})
	if err != nil {
		return NoAnswer(err.Error())
	}
	mapped := s.mapBack(raw)
	return Raw(mapped)
}

// mapBack rewrites generated-file URIs/ranges inside a gopls result to
// their .gp counterparts, structurally, wherever they appear.
func (s *Server) mapBack(raw json.RawMessage) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	v = s.mapBackValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

func (s *Server) mapBackValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		if uriv, hasURI := x["uri"].(string); hasURI {
			if rng, hasRange := x["range"].(map[string]any); hasRange {
				if nu, nr, ok := s.mapBackLocation(uriv, rng); ok {
					x["uri"], x["range"] = nu, nr
				}
			}
		}
		for k, val := range x {
			x[k] = s.mapBackValue(val)
		}
		return x
	case []any:
		for i := range x {
			x[i] = s.mapBackValue(x[i])
		}
		return x
	}
	return v
}

func (s *Server) mapBackLocation(uri string, rng map[string]any) (string, map[string]any, bool) {
	path := uriToPath(uri)
	s.mu.Lock()
	m := s.maps[path]
	s.mu.Unlock()
	if m == nil {
		return "", nil, false
	}
	conv := func(key string) bool {
		posv, ok := rng[key].(map[string]any)
		if !ok {
			return false
		}
		lf, _ := posv["line"].(float64)
		cf, _ := posv["character"].(float64)
		mapped, mok := m.Map(tokenPosition(int(lf)+1, int(cf)+1))
		if !mok {
			return false
		}
		posv["line"] = mapped.Line - 1
		posv["character"] = mapped.Column - 1
		return true
	}
	if !conv("start") || !conv("end") {
		return "", nil, false
	}
	return pathToURI(m.GoplusPath), rng, true
}

func forwardPos(m *sourcemap.Map, line, ch int) (Position, bool) {
	mapped, ok := m.Forward(tokenPosition(line+1, ch+1))
	if !ok {
		return Position{}, false
	}
	return Position{Line: mapped.Line - 1, Character: mapped.Column - 1}, true
}

func uriToPath(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		return u.Path
	}
	return uri
}

func pathToURI(path string) string {
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
	}
	return "file://" + path
}

// generatedCounterpart names the emitted twin of a source file. Each
// surface has its own suffix: foo.gp emits foo_gp.go, foo.goml foo_gml.go.
func generatedCounterpart(sourcePath string) string {
	if strings.HasSuffix(sourcePath, ".goml") {
		return strings.TrimSuffix(sourcePath, ".goml") + "_gml.go"
	}
	return strings.TrimSuffix(sourcePath, ".gp") + "_gp.go"
}

// sourceCounterpart inverts generatedCounterpart.
func sourceCounterpart(genPath string) string {
	if strings.HasSuffix(genPath, "_gml.go") {
		return strings.TrimSuffix(genPath, "_gml.go") + ".goml"
	}
	return strings.TrimSuffix(genPath, "_gp.go") + ".gp"
}

// tokenPosition builds a 1-based position for sourcemap calls.
func tokenPosition(line, col int) token.Position {
	return token.Position{Line: line, Column: col}
}
