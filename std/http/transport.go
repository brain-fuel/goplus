// Package http provides HTTP transport capabilities that complement net/http.
// Its Transport discovers HTTP/3 through Alt-Svc and safely falls back through
// net/http's HTTP/2 and HTTP/1.1 negotiation.
package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	nethttp "net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"goforge.dev/goplus/std/http3"
)

// Mode controls transport selection.
type Mode uint8

const (
	// Auto uses HTTP/3 for origins learned through Alt-Svc (or when
	// PriorKnowledge is enabled), falling back to HTTP/2 or HTTP/1.1.
	Auto Mode = iota
	// HTTP3Only requires HTTP/3 and never falls back.
	HTTP3Only
	// HTTP2Or1Only disables HTTP/3.
	HTTP2Or1Only
)

// Transport implements net/http.RoundTripper with HTTP/3 -> HTTP/2 ->
// HTTP/1.1 selection. The zero value is ready for use.
type Transport struct {
	Mode Mode

	// PriorKnowledge tries HTTP/3 for every HTTPS origin before an Alt-Svc
	// advertisement has been observed.
	PriorKnowledge bool

	TLSClientConfig *tls.Config
	QUICConfig      *quic.Config

	// HTTP3 and Fallback permit shared or instrumented transports. Nil values
	// use an HTTP/3 transport and net/http.DefaultTransport respectively.
	HTTP3    nethttp.RoundTripper
	Fallback nethttp.RoundTripper

	mu           sync.Mutex
	h3           *http3.Transport
	fallback     *nethttp.Transport
	capabilities map[string]h3Capability
	broken       map[string]brokenH3Alternative
}

const brokenHTTP3Cooldown = 5 * time.Minute

type h3Capability struct {
	until     time.Time
	authority string
}

type brokenH3Alternative struct {
	until     time.Time
	authority string
}

// RoundTrip implements net/http.RoundTripper.
func (t *Transport) RoundTrip(req *nethttp.Request) (*nethttp.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("http: nil request or URL")
	}
	if t.Mode == HTTP2Or1Only || req.URL.Scheme != "https" {
		return t.roundTripFallback(req)
	}
	tryH3 := t.Mode == HTTP3Only || t.PriorKnowledge || t.SupportsHTTP3(req.URL)
	if !tryH3 {
		response, err := t.roundTripFallback(req)
		if err == nil {
			t.ObserveAltSvc(req.URL, response.Header.Values("Alt-Svc"), time.Now())
		}
		return response, err
	}
	// Ordinary request bodies can only be retried if net/http can reproduce
	// them. Extended CONNECT is a streaming tunnel: attempt HTTP/3, and if it
	// fails return the error to the protocol owner so it can create a fresh
	// HTTP/2 or HTTP/1.1 stream.
	extendedConnect := req.Method == nethttp.MethodConnect && req.Proto != "" && req.Proto != "HTTP/1.1"
	if t.Mode != HTTP3Only && !extendedConnect && req.Body != nil && req.Body != nethttp.NoBody && req.GetBody == nil {
		return t.roundTripFallback(req)
	}
	h3req := req.Clone(req.Context())
	response, h3err := t.roundTripHTTP3(h3req)
	if h3err == nil {
		t.MarkHTTP3(req.URL, time.Now().Add(24*time.Hour))
		return response, nil
	}
	if !fallbackEligible(h3err) {
		return nil, h3err
	}
	if t.Mode == HTTP3Only || req.Context().Err() != nil {
		return nil, h3err
	}
	t.markHTTP3Broken(req.URL, time.Now().Add(brokenHTTP3Cooldown))
	if extendedConnect {
		return nil, h3err
	}
	fallbackReq, err := replay(req)
	if err != nil {
		return nil, h3err
	}
	response, err = t.roundTripFallback(fallbackReq)
	if err == nil {
		t.ObserveAltSvc(req.URL, response.Header.Values("Alt-Svc"), time.Now())
	}
	return response, err
}

func fallbackEligible(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var verification *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalid x509.CertificateInvalidError
	return !errors.As(err, &verification) && !errors.As(err, &unknownAuthority) &&
		!errors.As(err, &hostname) && !errors.As(err, &invalid)
}

func replay(req *nethttp.Request) (*nethttp.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil || req.Body == nethttp.NoBody {
		return clone, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}

func (t *Transport) roundTripHTTP3(req *nethttp.Request) (*nethttp.Response, error) {
	if t.HTTP3 != nil {
		return t.HTTP3.RoundTrip(req)
	}
	t.mu.Lock()
	if t.h3 == nil {
		t.h3 = &http3.Transport{
			QUICConfig: t.QUICConfig,
			Dial: func(ctx context.Context, addr string, tlsConfig *tls.Config, config *quic.Config) (*quic.Conn, error) {
				target := t.altAuthority(addr)
				return quic.DialAddr(ctx, target, tlsConfig, config)
			},
		}
		if t.TLSClientConfig != nil {
			t.h3.TLSClientConfig = t.TLSClientConfig.Clone()
		}
	}
	h3 := t.h3
	t.mu.Unlock()
	return h3.RoundTrip(req)
}

func (t *Transport) roundTripFallback(req *nethttp.Request) (*nethttp.Response, error) {
	if t.Fallback != nil {
		return t.Fallback.RoundTrip(req)
	}
	t.mu.Lock()
	if t.fallback == nil {
		base, ok := nethttp.DefaultTransport.(*nethttp.Transport)
		if !ok {
			t.mu.Unlock()
			return nil, errors.New("http: default transport has unexpected type")
		}
		t.fallback = base.Clone()
		if t.TLSClientConfig != nil {
			t.fallback.TLSClientConfig = t.TLSClientConfig.Clone()
		}
	}
	fallback := t.fallback
	t.mu.Unlock()
	return fallback.RoundTrip(req)
}

// SupportsHTTP3 reports whether HTTP/3 capability is currently cached for u.
func (t *Transport) SupportsHTTP3(u *url.URL) bool {
	if u == nil {
		return false
	}
	origin := canonicalOrigin(u)
	t.mu.Lock()
	defer t.mu.Unlock()
	if alternative, broken := t.broken[origin]; broken {
		if time.Now().Before(alternative.until) {
			return false
		}
		delete(t.broken, origin)
	}
	capability, ok := t.capabilities[origin]
	if ok && !time.Now().Before(capability.until) {
		delete(t.capabilities, origin)
		return false
	}
	return ok
}

// MarkHTTP3 records prior knowledge for an origin until the supplied time.
func (t *Transport) MarkHTTP3(u *url.URL, until time.Time) {
	if u == nil || u.Scheme != "https" || until.IsZero() {
		return
	}
	t.mu.Lock()
	if t.capabilities == nil {
		t.capabilities = make(map[string]h3Capability)
	}
	origin := canonicalOrigin(u)
	delete(t.broken, origin)
	t.capabilities[origin] = h3Capability{until: until, authority: net.JoinHostPort(u.Hostname(), originPort(u))}
	t.mu.Unlock()
}

func (t *Transport) markHTTP3Alternative(u *url.URL, authority string, until, now time.Time) {
	if strings.HasPrefix(authority, ":") {
		authority = net.JoinHostPort(u.Hostname(), strings.TrimPrefix(authority, ":"))
	}
	t.mu.Lock()
	if t.capabilities == nil {
		t.capabilities = make(map[string]h3Capability)
	}
	origin := canonicalOrigin(u)
	if broken, exists := t.broken[origin]; exists && now.Before(broken.until) && strings.EqualFold(broken.authority, authority) {
		t.mu.Unlock()
		return
	}
	delete(t.broken, origin)
	t.capabilities[origin] = h3Capability{until: until, authority: authority}
	t.mu.Unlock()
}

func (t *Transport) altAuthority(addr string) string {
	origin := "https://" + strings.ToLower(addr)
	t.mu.Lock()
	defer t.mu.Unlock()
	capability, ok := t.capabilities[origin]
	if !ok || !time.Now().Before(capability.until) || capability.authority == "" {
		return addr
	}
	return capability.authority
}

// ForgetHTTP3 removes learned capability after a failed HTTP/3 attempt.
func (t *Transport) ForgetHTTP3(u *url.URL) {
	if u == nil {
		return
	}
	t.mu.Lock()
	origin := canonicalOrigin(u)
	delete(t.capabilities, origin)
	delete(t.broken, origin)
	t.mu.Unlock()
}

func (t *Transport) markHTTP3Broken(u *url.URL, until time.Time) {
	if u == nil {
		return
	}
	t.mu.Lock()
	if t.broken == nil {
		t.broken = make(map[string]brokenH3Alternative)
	}
	origin := canonicalOrigin(u)
	authority := net.JoinHostPort(u.Hostname(), originPort(u))
	if capability, ok := t.capabilities[origin]; ok && capability.authority != "" {
		authority = capability.authority
	}
	delete(t.capabilities, origin)
	t.broken[origin] = brokenH3Alternative{until: until, authority: authority}
	t.mu.Unlock()
}

// ObserveAltSvc records an h3 alternative, including an alternative host or
// UDP port, while retaining the origin host for TLS verification and HTTP
// authority.
func (t *Transport) ObserveAltSvc(u *url.URL, values []string, now time.Time) {
	if u == nil || u.Scheme != "https" {
		return
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), "clear") {
			t.ForgetHTTP3(u)
			return
		}
		for _, alternative := range strings.Split(value, ",") {
			parts := strings.Split(alternative, ";")
			if len(parts) == 0 {
				continue
			}
			binding := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
			if len(binding) != 2 || !strings.EqualFold(binding[0], "h3") {
				continue
			}
			authority, err := strconv.Unquote(strings.TrimSpace(binding[1]))
			if err != nil || !validAltAuthority(u, authority) {
				continue
			}
			maxAge := 24 * time.Hour
			valid := true
			for _, parameter := range parts[1:] {
				pair := strings.SplitN(strings.TrimSpace(parameter), "=", 2)
				if len(pair) == 2 && strings.EqualFold(pair[0], "ma") {
					seconds, parseErr := strconv.ParseInt(strings.Trim(pair[1], "\""), 10, 64)
					if parseErr != nil || seconds < 0 || seconds > int64((1<<63-1)/time.Second) {
						valid = false
						break
					}
					maxAge = time.Duration(seconds) * time.Second
				}
			}
			if !valid {
				continue
			}
			if maxAge == 0 {
				t.ForgetHTTP3(u)
			} else {
				t.markHTTP3Alternative(u, authority, now.Add(maxAge), now)
			}
			return
		}
	}
}

func validAltAuthority(u *url.URL, authority string) bool {
	if strings.HasPrefix(authority, ":") {
		_, err := strconv.ParseUint(strings.TrimPrefix(authority, ":"), 10, 16)
		return err == nil
	}
	host, port, err := net.SplitHostPort(authority)
	if err != nil || host == "" {
		return false
	}
	_, err = strconv.ParseUint(port, 10, 16)
	return err == nil
}

func originPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	return "443"
}

func canonicalOrigin(u *url.URL) string {
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(net.JoinHostPort(u.Hostname(), originPort(u)))
}

// CloseIdleConnections closes idle connections owned by both transport tiers.
func (t *Transport) CloseIdleConnections() {
	t.mu.Lock()
	h3 := t.h3
	fallback := t.fallback
	t.h3 = nil
	t.fallback = nil
	t.mu.Unlock()
	if h3 != nil {
		_ = h3.Close()
	}
	if fallback != nil {
		fallback.CloseIdleConnections()
	}
	if closer, ok := t.Fallback.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

// WithHTTP3Capability returns a context carrying no mutable state; it exists
// as a semantic marker for callers that explicitly choose PriorKnowledge.
func WithHTTP3Capability(ctx context.Context) context.Context { return ctx }
