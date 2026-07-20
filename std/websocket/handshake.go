package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var ErrHandshake = errors.New("websocket: invalid opening handshake")

// AcceptKey computes Sec-WebSocket-Accept without heap allocation except for
// the returned string.
func AcceptKey(key string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(strings.TrimSpace(key)))
	_, _ = h.Write([]byte(websocketGUID))
	var digest [sha1.Size]byte
	sum := h.Sum(digest[:0])
	var encoded [28]byte
	base64.StdEncoding.Encode(encoded[:], sum)
	return string(encoded[:])
}

func hasToken(value, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func joinedHeader(header http.Header, name string) string {
	return strings.Join(header.Values(name), ",")
}

func singleHeader(header http.Header, name string) (string, bool) {
	values := header.Values(name)
	if len(values) != 1 || strings.Contains(values[0], ",") {
		return "", false
	}
	return values[0], true
}

func validToken(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))) {
			return false
		}
	}
	return true
}

// ValidateServerRequest checks every mandatory RFC 6455 server-side opening
// handshake condition and returns the trimmed nonce.
func ValidateServerRequest(r *http.Request) (string, error) {
	if r.Method != http.MethodGet || r.ProtoMajor != 1 || r.ProtoMinor < 1 || r.Host == "" {
		return "", ErrHandshake
	}
	if !hasToken(joinedHeader(r.Header, "Connection"), "upgrade") || !hasToken(joinedHeader(r.Header, "Upgrade"), "websocket") {
		return "", ErrHandshake
	}
	version, ok := singleHeader(r.Header, "Sec-WebSocket-Version")
	if !ok || version != "13" {
		return "", ErrHandshake
	}
	rawKey, ok := singleHeader(r.Header, "Sec-WebSocket-Key")
	if !ok {
		return "", ErrHandshake
	}
	key := strings.TrimSpace(rawKey)
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 16 {
		return "", ErrHandshake
	}
	for _, protocol := range strings.Split(joinedHeader(r.Header, "Sec-WebSocket-Protocol"), ",") {
		protocol = strings.TrimSpace(protocol)
		if protocol != "" && !validToken(protocol) {
			return "", ErrHandshake
		}
	}
	return key, nil
}
