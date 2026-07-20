package websocket

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type plainResponseWriter struct{ header http.Header }

func (w *plainResponseWriter) Header() http.Header       { return w.header }
func (*plainResponseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (*plainResponseWriter) WriteHeader(int)             {}

type failingHijacker struct{ err error }

func (w *failingHijacker) Header() http.Header                          { return make(http.Header) }
func (*failingHijacker) Write(p []byte) (int, error)                    { return len(p), nil }
func (*failingHijacker) WriteHeader(int)                                {}
func (w *failingHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) { return nil, nil, w.err }

type limitedConn struct {
	reader    io.Reader
	remaining int
	closed    chan struct{}
}

func (c *limitedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }
func (c *limitedConn) Write(p []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, errors.New("write failed")
	}
	n := len(p)
	if n > c.remaining {
		n = c.remaining
	}
	c.remaining -= n
	if n != len(p) {
		return n, errors.New("write failed")
	}
	return n, nil
}
func (c *limitedConn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}
func (*limitedConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (*limitedConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (*limitedConn) SetDeadline(time.Time) error      { return nil }
func (*limitedConn) SetReadDeadline(time.Time) error  { return nil }
func (*limitedConn) SetWriteDeadline(time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return string(a) }

type connHijacker struct {
	conn *limitedConn
	size int
}

type oneConnListener struct {
	conn net.Conn
	done bool
	err  error
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	if !l.done {
		l.done = true
		return l.conn, nil
	}
	return nil, l.err
}
func (l *oneConnListener) Close() error   { return nil }
func (l *oneConnListener) Addr() net.Addr { return dummyAddr("listener") }

func (*connHijacker) Header() http.Header         { return make(http.Header) }
func (*connHijacker) Write(p []byte) (int, error) { return len(p), nil }
func (*connHijacker) WriteHeader(int)             {}
func (h *connHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.conn, bufio.NewReadWriter(bufio.NewReader(h.conn), bufio.NewWriterSize(h.conn, h.size)), nil
}

func TestSameOriginAndSelectionEdges(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.test/socket", nil)
	r.Header["Origin"] = []string{"http://example.test", "http://example.test"}
	if SameOrigin(r) {
		t.Fatal("duplicate Origin accepted")
	}
	r.Header.Set("Origin", "http://example.test")
	r.Host = ":"
	if SameOrigin(r) {
		t.Fatal("invalid request Host accepted")
	}
	r = httptest.NewRequest(http.MethodGet, "http://example.test/socket", nil)
	r.Header.Set("Origin", "http://example.test:80")
	if !SameOrigin(r) {
		t.Fatal("explicit HTTP default port rejected")
	}
	if got := selectProtocol(" , unknown", []string{"chat"}); got != "" {
		t.Fatalf("unexpected protocol %q", got)
	}
}

func TestUpgradePreconditionFailures(t *testing.T) {
	r := validServerRequest()
	if _, _, err := Upgrade(&plainResponseWriter{header: make(http.Header)}, r, UpgradeOptions{Protocols: []string{"bad protocol"}}); !errors.Is(err, ErrHandshake) {
		t.Fatalf("protocol option: %v", err)
	}
	badRequest := validServerRequest()
	badRequest.Method = http.MethodPost
	if _, _, err := Upgrade(&plainResponseWriter{header: make(http.Header)}, badRequest, UpgradeOptions{}); !errors.Is(err, ErrHandshake) {
		t.Fatalf("request validation: %v", err)
	}
	if _, _, err := Upgrade(&plainResponseWriter{header: make(http.Header)}, r, UpgradeOptions{CheckOrigin: func(*http.Request) bool { return false }}); !errors.Is(err, ErrHandshake) {
		t.Fatalf("origin check: %v", err)
	}
	if _, _, err := Upgrade(&plainResponseWriter{header: make(http.Header)}, r, UpgradeOptions{Compression: &CompressionOptions{ClientMaxWindowBits: 7}}); !errors.Is(err, ErrInvalidExtension) {
		t.Fatalf("compression config: %v", err)
	}
	if _, _, err := Upgrade(&plainResponseWriter{header: make(http.Header)}, r, UpgradeOptions{}); err == nil || !strings.Contains(err.Error(), "cannot hijack") {
		t.Fatalf("non-hijacker: %v", err)
	}
	want := errors.New("hijack")
	if _, _, err := Upgrade(&failingHijacker{err: want}, r, UpgradeOptions{}); !errors.Is(err, want) {
		t.Fatalf("hijack failure: %v", err)
	}
}

func TestUpgradeWriteFailuresCloseHijackedConnection(t *testing.T) {
	r := validServerRequest()
	key, _ := ValidateServerRequest(r)
	base := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + AcceptKey(key) + "\r\n"
	protocol := "Sec-WebSocket-Protocol: chat\r\n"
	extension := "Sec-WebSocket-Extensions: permessage-deflate; server_no_context_takeover; client_no_context_takeover\r\n"
	tests := []struct {
		name      string
		allowance int
		size      int
		opts      UpgradeOptions
	}{
		{"initial", 0, 1, UpgradeOptions{}},
		{"protocol", len(base), 1, UpgradeOptions{Protocols: []string{"chat"}}},
		{"extension", len(base) + len(protocol), 1, UpgradeOptions{Protocols: []string{"chat"}, Compression: &CompressionOptions{}}},
		{"terminator", len(base) + len(protocol) + len(extension), 1, UpgradeOptions{Protocols: []string{"chat"}, Compression: &CompressionOptions{}}},
		{"flush", 0, 4096, UpgradeOptions{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validServerRequest()
			request.Header.Set("Sec-WebSocket-Protocol", "chat")
			request.Header.Set("Sec-WebSocket-Extensions", "permessage-deflate; client_no_context_takeover")
			conn := &limitedConn{reader: strings.NewReader(""), remaining: test.allowance, closed: make(chan struct{})}
			_, _, err := Upgrade(&connHijacker{conn: conn, size: test.size}, request, test.opts)
			if err == nil {
				t.Fatal("write failure was accepted")
			}
			select {
			case <-conn.closed:
			default:
				t.Fatal("failed hijacked connection was not closed")
			}
		})
	}
}

func TestRawServeRejectsAndAccepts(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan struct{}, 1)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- Serve(listener, func(conn *Conn) {
			served <- struct{}{}
			_ = conn.Close()
		})
	}()
	dial := func(request string) {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(conn, request)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, _ = io.ReadAll(conn)
		_ = conn.Close()
	}
	dial("not HTTP\r\n\r\n")
	dial("GET / HTTP/1.1\r\nHost: example.test\r\n\r\n")
	valid := "GET / HTTP/1.1\r\nHost: example.test\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
	dial(valid)
	select {
	case <-served:
	case <-time.After(time.Second):
		t.Fatal("handler was not called")
	}
	_ = listener.Close()
	if err := <-serveErr; err == nil {
		t.Fatal("Serve returned nil after listener close")
	}
}

func TestRawServeClosesOnHandshakeWriteFailure(t *testing.T) {
	request := "GET / HTTP/1.1\r\nHost: example.test\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
	conn := &limitedConn{reader: strings.NewReader(request), remaining: 0, closed: make(chan struct{})}
	want := errors.New("listener done")
	listener := &oneConnListener{conn: conn, err: want}
	if err := Serve(listener, func(*Conn) { t.Error("handler called after failed response write") }); !errors.Is(err, want) {
		t.Fatalf("Serve error: %v", err)
	}
	select {
	case <-conn.closed:
	case <-time.After(time.Second):
		t.Fatal("connection was not closed after response write failure")
	}
}
