package websocket

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type UpgradeOptions struct {
	Protocols   []string
	CheckOrigin func(*http.Request) bool
	Config      ConnConfig
	Compression *CompressionOptions
}

// SameOrigin accepts non-browser clients without Origin and otherwise
// requires the Origin host to match the HTTP Host header. Pass it as
// UpgradeOptions.CheckOrigin for browser-facing endpoints.
func SameOrigin(r *http.Request) bool {
	values := r.Header.Values("Origin")
	if len(values) == 0 {
		return true
	}
	if len(values) != 1 {
		return false
	}
	raw := values[0]
	origin, err := url.Parse(raw)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return false
	}
	requestURL, err := url.Parse("//" + r.Host)
	if err != nil || requestURL.Hostname() == "" {
		return false
	}
	requestScheme := "http"
	if r.TLS != nil {
		requestScheme = "https"
	}
	port := func(u *url.URL, scheme string) string {
		if value := u.Port(); value != "" {
			return value
		}
		if scheme == "https" {
			return "443"
		}
		return "80"
	}
	return strings.EqualFold(origin.Hostname(), requestURL.Hostname()) && port(origin, origin.Scheme) == port(requestURL, requestScheme)
}

func selectProtocol(header string, supported []string) string {
	offeredSet := make(map[string]struct{})
	for _, offered := range strings.Split(header, ",") {
		if offered = strings.TrimSpace(offered); offered != "" {
			offeredSet[offered] = struct{}{}
		}
	}
	for _, candidate := range supported {
		if _, offered := offeredSet[candidate]; offered {
			return candidate
		}
	}
	return ""
}

// Upgrade accepts an RFC 6455 HTTP/1.1 request and preserves bytes already
// buffered after the handshake.
func Upgrade(w http.ResponseWriter, r *http.Request, opts UpgradeOptions) (*Conn, string, error) {
	for _, protocol := range opts.Protocols {
		if !validToken(protocol) {
			return nil, "", ErrHandshake
		}
	}
	if isRFC9220Request(r) {
		return upgradeExtendedCONNECT(w, r, opts, RFC9220Handshake)
	}
	if isRFC8441Request(r) {
		return upgradeExtendedCONNECT(w, r, opts, RFC8441Handshake)
	}
	if isUnsupportedExtendedCONNECT(r) {
		w.WriteHeader(http.StatusNotImplemented)
		return nil, "", ErrHandshake
	}
	key, err := ValidateServerRequest(r)
	if err != nil {
		return nil, "", err
	}
	if opts.CheckOrigin != nil && !opts.CheckOrigin(r) {
		return nil, "", ErrHandshake
	}
	protocol := selectProtocol(joinedHeader(r.Header, "Sec-WebSocket-Protocol"), opts.Protocols)
	compressionResponse, negotiated := "", compressionSettings{}
	if opts.Compression != nil {
		if err = opts.Compression.validate(); err != nil {
			return nil, "", err
		}
		compressionResponse, negotiated, _ = negotiateCompression(joinedHeader(r.Header, "Sec-WebSocket-Extensions"), *opts.Compression)
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, "", fmt.Errorf("websocket: response writer cannot hijack")
	}
	nc, rw, err := hj.Hijack()
	if err != nil {
		return nil, "", err
	}
	fail := true
	defer func() {
		if fail {
			_ = nc.Close()
		}
	}()
	if _, err = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + AcceptKey(key) + "\r\n"); err != nil {
		return nil, "", err
	}
	if protocol != "" {
		if _, err = rw.WriteString("Sec-WebSocket-Protocol: " + protocol + "\r\n"); err != nil {
			return nil, "", err
		}
	}
	if compressionResponse != "" {
		if _, err = rw.WriteString("Sec-WebSocket-Extensions: " + compressionResponse + "\r\n"); err != nil {
			return nil, "", err
		}
	}
	if _, err = rw.WriteString("\r\n"); err != nil {
		return nil, "", err
	}
	if err = rw.Flush(); err != nil {
		return nil, "", err
	}
	fail = false
	conn := NewConn(nc, ServerSide, rw.Reader, opts.Config)
	if compressionResponse != "" {
		conn.enableCompression(negotiated)
	}
	return conn, protocol, nil
}

func isUnsupportedExtendedCONNECT(r *http.Request) bool {
	if r == nil || r.Method != http.MethodConnect {
		return false
	}
	if r.ProtoMajor == 3 {
		return r.Proto != "" && r.Proto != "websocket"
	}
	return r.ProtoMajor == 2 && r.Header.Get(":protocol") != "" && r.Header.Get(":protocol") != "websocket"
}

// IsRFC8441Request reports whether r is an HTTP/2 WebSocket extended CONNECT
// request. It does not imply that the rest of the opening handshake is valid.
func IsRFC8441Request(r *http.Request) bool {
	return isRFC8441Request(r)
}

func isRFC8441Request(r *http.Request) bool {
	return r != nil && r.ProtoMajor == 2 && r.Method == http.MethodConnect && r.Header.Get(":protocol") == "websocket"
}

// IsRFC9220Request reports whether r is an HTTP/3 WebSocket extended CONNECT
// request. It does not imply that the rest of the handshake is valid.
func IsRFC9220Request(r *http.Request) bool { return isRFC9220Request(r) }

func isRFC9220Request(r *http.Request) bool {
	return r != nil && r.ProtoMajor == 3 && r.Method == http.MethodConnect && r.Proto == "websocket"
}

func validateRFC8441Request(r *http.Request) error {
	if !isRFC8441Request(r) || r.Host == "" {
		return ErrHandshake
	}
	for _, forbidden := range []string{"Connection", "Upgrade", "Sec-WebSocket-Key", "Sec-WebSocket-Accept"} {
		if len(r.Header.Values(forbidden)) != 0 {
			return ErrHandshake
		}
	}
	version, ok := singleHeader(r.Header, "Sec-WebSocket-Version")
	if !ok || version != "13" {
		return ErrHandshake
	}
	for _, protocol := range strings.Split(joinedHeader(r.Header, "Sec-WebSocket-Protocol"), ",") {
		protocol = strings.TrimSpace(protocol)
		if protocol != "" && !validToken(protocol) {
			return ErrHandshake
		}
	}
	return nil
}

func validateRFC9220Request(r *http.Request) error {
	if !isRFC9220Request(r) || r.Host == "" {
		return ErrHandshake
	}
	for _, forbidden := range []string{"Connection", "Upgrade", "Sec-WebSocket-Key", "Sec-WebSocket-Accept", ":protocol"} {
		if len(r.Header.Values(forbidden)) != 0 {
			return ErrHandshake
		}
	}
	version, ok := singleHeader(r.Header, "Sec-WebSocket-Version")
	if !ok || version != "13" {
		return ErrHandshake
	}
	for _, protocol := range strings.Split(joinedHeader(r.Header, "Sec-WebSocket-Protocol"), ",") {
		protocol = strings.TrimSpace(protocol)
		if protocol != "" && !validToken(protocol) {
			return ErrHandshake
		}
	}
	return nil
}

func upgradeRFC8441(w http.ResponseWriter, r *http.Request, opts UpgradeOptions) (*Conn, string, error) {
	return upgradeExtendedCONNECT(w, r, opts, RFC8441Handshake)
}

func upgradeExtendedCONNECT(w http.ResponseWriter, r *http.Request, opts UpgradeOptions, handshake HandshakeProtocol) (*Conn, string, error) {
	var err error
	if handshake == RFC9220Handshake {
		err = validateRFC9220Request(r)
	} else {
		err = validateRFC8441Request(r)
	}
	if err != nil {
		return nil, "", err
	}
	if opts.CheckOrigin != nil && !opts.CheckOrigin(r) {
		return nil, "", ErrHandshake
	}
	protocol := selectProtocol(joinedHeader(r.Header, "Sec-WebSocket-Protocol"), opts.Protocols)
	compressionResponse, negotiated := "", compressionSettings{}
	if opts.Compression != nil {
		if err := opts.Compression.validate(); err != nil {
			return nil, "", err
		}
		compressionResponse, negotiated, _ = negotiateCompression(joinedHeader(r.Header, "Sec-WebSocket-Extensions"), *opts.Compression)
	}
	if protocol != "" {
		w.Header().Set("Sec-WebSocket-Protocol", protocol)
	}
	if compressionResponse != "" {
		w.Header().Set("Sec-WebSocket-Extensions", compressionResponse)
	}
	w.WriteHeader(http.StatusOK)
	controller := http.NewResponseController(w)
	if err := controller.Flush(); err != nil {
		return nil, "", err
	}
	_, cancel := context.WithCancel(r.Context())
	stream := newStreamConn(r.Body, &responseStreamWriter{w: w, flusher: controller}, cancel)
	stream.setReadDeadline = controller.SetReadDeadline
	stream.setWriteDeadline = controller.SetWriteDeadline
	conn := NewConn(stream, ServerSide, nil, opts.Config)
	conn.handshake = handshake
	if compressionResponse != "" {
		conn.enableCompression(negotiated)
	}
	return conn, protocol, nil
}

// Serve accepts raw TCP connections and applies handler after an HTTP upgrade.
// It is intentionally small; net/http Upgrade is the preferred server API.
func Serve(listener net.Listener, handler func(*Conn)) error {
	for {
		nc, err := listener.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			br := bufio.NewReader(c)
			req, err := http.ReadRequest(br)
			if err != nil {
				_ = c.Close()
				return
			}
			key, err := ValidateServerRequest(req)
			if err != nil {
				_ = c.Close()
				return
			}
			_, err = fmt.Fprintf(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", AcceptKey(key))
			if err != nil {
				_ = c.Close()
				return
			}
			handler(NewConn(c, ServerSide, br, ConnConfig{}))
		}(nc)
	}
}
