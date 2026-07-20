package websocket

import (
	"bufio"
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
