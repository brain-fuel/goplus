package websocket

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	refhttp3 "github.com/quic-go/quic-go/http3"
	ghttp "goforge.dev/goplus/std/http"
	nativehttp3 "goforge.dev/goplus/std/http3"
)

type cachedCapabilityTransport struct{ supported bool }

func (t cachedCapabilityTransport) SupportsHTTP3(*url.URL) bool { return t.supported }
func (cachedCapabilityTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("not called")
}

func TestHTTP3AutoConsultsTransportCapability(t *testing.T) {
	u, _ := url.Parse("wss://example.com/socket")
	if shouldTryHTTP3(u, DialOptions{HTTP3Transport: cachedCapabilityTransport{}}) {
		t.Fatal("HTTP3Auto probed without learned capability")
	}
	if !shouldTryHTTP3(u, DialOptions{HTTP3Transport: cachedCapabilityTransport{supported: true}}) {
		t.Fatal("HTTP3Auto ignored learned capability")
	}
	if !shouldTryHTTP3(u, DialOptions{HTTP3: HTTP3Only, HTTP3Transport: cachedCapabilityTransport{}}) {
		t.Fatal("HTTP3Only did not override the capability cache")
	}
}

func testHTTP3TLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
}

func TestRFC9220EndToEnd(t *testing.T) {
	errCh := make(chan error, 1)
	server := &nativehttp3.XNetServer{
		TLSConfig: testHTTP3TLSConfig(t),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("response") {
			case "status":
				w.WriteHeader(http.StatusForbidden)
				return
			case "bad-header":
				w.Header().Set("Upgrade", "websocket")
				w.WriteHeader(http.StatusOK)
				return
			case "bad-protocol":
				w.Header().Set("Sec-WebSocket-Protocol", "unexpected")
				w.WriteHeader(http.StatusOK)
				return
			}
			if !IsRFC9220Request(r) {
				errCh <- errors.New("request was not RFC 9220")
				return
			}
			conn, protocol, err := Upgrade(w, r, UpgradeOptions{Protocols: []string{"chat.v3"}, Compression: &CompressionOptions{}})
			if err != nil {
				errCh <- err
				return
			}
			defer conn.Close()
			if protocol != "chat.v3" || conn.HandshakeProtocol() != RFC9220Handshake {
				errCh <- errors.New("server did not negotiate RFC 9220")
				return
			}
			message, err := conn.ReadMessage()
			if err == nil {
				err = conn.WriteMessage(message)
			}
			errCh <- err
		}),
	}
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(packetConn) }()
	defer func() {
		_ = server.Close()
		select {
		case err := <-serveDone:
			if err != nil && !errors.Is(err, http.ErrServerClosed) && !strings.Contains(err.Error(), "closed network connection") {
				t.Errorf("server close: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("HTTP/3 server did not close")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := Dial(ctx, "wss://"+packetConn.LocalAddr().String()+"/echo?transport=h3", DialOptions{
		Protocols:   []string{"chat.v3"},
		HTTP3:       HTTP3Only,
		TLSConfig:   &tls.Config{InsecureSkipVerify: true},
		Compression: &CompressionOptions{},
	})
	if err != nil {
		select {
		case serverErr := <-errCh:
			t.Fatalf("dial: %v (server: %v, response: %#v)", err, serverErr, response)
		default:
			t.Fatalf("dial: %v (response: %#v)", err, response)
		}
	}
	defer conn.Close()
	if response.ProtoMajor != 3 || conn.HandshakeProtocol() != RFC9220Handshake {
		t.Fatalf("transport = HTTP/%d, %v", response.ProtoMajor, conn.HandshakeProtocol())
	}
	if response.Body != http.NoBody {
		t.Fatal("successful response exposed the HTTP/3 tunnel body")
	}
	payload := []byte("one API over an HTTP/3 stream")
	if err := conn.WriteText(payload); err != nil {
		t.Fatal(err)
	}
	message, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	text, ok := message.(TextMessage)
	if !ok || !bytes.Equal(text.Payload, payload) {
		t.Fatalf("echo = %#v", message)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	for _, responseKind := range []string{"status", "bad-protocol"} {
		_, _, err := Dial(ctx, "wss://"+packetConn.LocalAddr().String()+"/echo?response="+responseKind, DialOptions{
			HTTP3: HTTP3Only, TLSConfig: &tls.Config{InsecureSkipVerify: true},
		})
		if err == nil {
			t.Fatalf("owned transport accepted %s response", responseKind)
		}
	}
}

func TestRFC9220RequestValidation(t *testing.T) {
	req := &http.Request{
		Method: http.MethodConnect,
		Proto:  "websocket", ProtoMajor: 3,
		Host:   "example.com",
		Header: http.Header{"Sec-Websocket-Version": {"13"}},
	}
	if !IsRFC9220Request(req) {
		t.Fatal("canonical request not recognized")
	}
	if err := validateRFC9220Request(req); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Connection", "Upgrade", "Sec-WebSocket-Key", "Sec-WebSocket-Accept", ":protocol"} {
		bad := req.Clone(context.Background())
		bad.Header = req.Header.Clone()
		bad.Header.Set(name, "forbidden")
		if err := validateRFC9220Request(bad); err == nil {
			t.Fatalf("accepted forbidden %s", name)
		}
	}
	badVersion := req.Clone(context.Background())
	badVersion.Header = req.Header.Clone()
	badVersion.Header.Del("Sec-WebSocket-Version")
	if err := validateRFC9220Request(badVersion); err == nil {
		t.Fatal("missing version accepted")
	}
	badProtocol := req.Clone(context.Background())
	badProtocol.Header = req.Header.Clone()
	badProtocol.Header.Set("Sec-WebSocket-Protocol", "bad protocol")
	if err := validateRFC9220Request(badProtocol); err == nil {
		t.Fatal("invalid subprotocol accepted")
	}
	notH3 := req.Clone(context.Background())
	notH3.ProtoMajor = 2
	if err := validateRFC9220Request(notH3); err == nil {
		t.Fatal("non-HTTP/3 request accepted")
	}
	if _, _, err := upgradeRFC8441(&recordingResponseWriter{header: make(http.Header)}, req, UpgradeOptions{}); err == nil {
		t.Fatal("RFC 8441 wrapper accepted an RFC 9220 request")
	}
}

func TestUnsupportedExtendedCONNECTReturns501(t *testing.T) {
	for _, req := range []*http.Request{
		{Method: http.MethodConnect, ProtoMajor: 3, Proto: "not-websocket"},
		{Method: http.MethodConnect, ProtoMajor: 2, Header: http.Header{":protocol": {"not-websocket"}}},
	} {
		recorder := httptest.NewRecorder()
		if _, _, err := Upgrade(recorder, req, UpgradeOptions{}); !errors.Is(err, ErrHandshake) {
			t.Fatalf("error = %v, want handshake error", err)
		}
		if recorder.Code != http.StatusNotImplemented {
			t.Fatalf("status = %d, want 501", recorder.Code)
		}
	}
}

type failingHTTP3Transport struct{ err error }

func (t failingHTTP3Transport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

type http3RoundTripFunc func(*http.Request) (*http.Response, error)

func (f http3RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRFC9220ClientEdges(t *testing.T) {
	if RFC9220Handshake.String() != "RFC 9220" {
		t.Fatal("missing RFC 9220 handshake name")
	}
	wrapped := &http3UnavailableError{cause: io.ErrUnexpectedEOF}
	if wrapped.Error() == "" || !errors.Is(wrapped, io.ErrUnexpectedEOF) {
		t.Fatal("HTTP/3 unavailable error lost its cause")
	}
	base, _ := url.Parse("wss://example.com/socket")
	if _, _, err := dialRFC9220(context.Background(), base, DialOptions{Protocols: []string{"dup", "dup"}}); err == nil {
		t.Fatal("duplicate protocols accepted")
	}
	for _, raw := range []string{"ws://example.com/socket", "wss:///socket", "wss://example.com/socket#fragment"} {
		u, _ := url.Parse(raw)
		if _, _, err := dialRFC9220(context.Background(), u, DialOptions{}); !errors.Is(err, ErrHandshake) {
			t.Fatalf("invalid URL %q: %v", raw, err)
		}
	}
	if _, _, err := dialRFC9220(context.Background(), base, DialOptions{Header: http.Header{"Upgrade": {"websocket"}}}); !errors.Is(err, ErrHandshake) {
		t.Fatalf("forbidden request header: %v", err)
	}

	response := func(status, proto int, header http.Header) *http.Response {
		return &http.Response{StatusCode: status, ProtoMajor: proto, Header: header, Body: io.NopCloser(strings.NewReader("body"))}
	}
	for _, tc := range []struct {
		name        string
		response    *http.Response
		unavailable bool
	}{
		{name: "wrong protocol", response: response(200, 2, nil), unavailable: true},
		{name: "fallback status", response: response(http.StatusNotImplemented, 3, nil), unavailable: true},
		{name: "policy rejection", response: response(http.StatusForbidden, 3, nil)},
		{name: "bad handshake header", response: response(200, 3, http.Header{"Upgrade": {"websocket"}})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := dialRFC9220(context.Background(), base, DialOptions{HTTP3Transport: http3RoundTripFunc(func(*http.Request) (*http.Response, error) { return tc.response, nil })})
			var unavailable *http3UnavailableError
			if err == nil || errors.As(err, &unavailable) != tc.unavailable {
				t.Fatalf("err=%v unavailable=%v", err, errors.As(err, &unavailable))
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	_, _, err := dialRFC9220(ctx, base, DialOptions{HTTP3Transport: http3RoundTripFunc(func(*http.Request) (*http.Response, error) {
		cancel()
		return response(200, 3, nil), nil
	})})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled handshake = %v", err)
	}
	if isHTTP3Unavailable(nil) || isHTTP3Unavailable(context.Canceled) || isHTTP3Unavailable(context.DeadlineExceeded) {
		t.Fatal("cancellation classified as transport unavailability")
	}
	if !isHTTP3Unavailable(errors.New("udp unavailable")) {
		t.Fatal("UDP failure not classified as unavailable")
	}
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := packetConn.LocalAddr().String()
	packetConn.Close()
	failCtx, failCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer failCancel()
	failedURL, _ := url.Parse("wss://" + address + "/socket")
	if _, _, err := dialRFC9220(failCtx, failedURL, DialOptions{HTTP3: HTTP3Only, TLSConfig: &tls.Config{InsecureSkipVerify: true}}); err == nil {
		t.Fatal("unreachable owned HTTP/3 transport succeeded")
	}
}

func TestRFC9220FallsBackToRFC8441(t *testing.T) {
	if !strings.Contains(os.Getenv("GODEBUG"), "http2xconnect=1") {
		t.Skip("requires GODEBUG=http2xconnect=1 at process start")
	}
	errCh := make(chan error, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := Upgrade(w, r, UpgradeOptions{})
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()
		message, err := conn.ReadMessage()
		if err == nil {
			err = conn.WriteMessage(message)
		}
		errCh <- err
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := Dial(ctx, "wss"+strings.TrimPrefix(server.URL, "https"), DialOptions{
		TLSConfig:      &tls.Config{InsecureSkipVerify: true},
		HTTP3Transport: failingHTTP3Transport{err: errors.New("udp unavailable")},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if response.ProtoMajor != 2 || conn.HandshakeProtocol() != RFC8441Handshake {
		t.Fatalf("fallback = HTTP/%d, %v", response.ProtoMajor, conn.HandshakeProtocol())
	}
	if err := conn.WriteText([]byte("fallback")); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestRFC9220FallsThroughToRFC6455(t *testing.T) {
	errCh := make(chan error, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := Upgrade(w, r, UpgradeOptions{})
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()
		message, err := conn.ReadMessage()
		if err == nil {
			err = conn.WriteMessage(message)
		}
		errCh <- err
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := Dial(ctx, "wss"+strings.TrimPrefix(server.URL, "https"), DialOptions{
		TLSConfig:      &tls.Config{InsecureSkipVerify: true},
		HTTP3Transport: failingHTTP3Transport{err: errors.New("udp unavailable")},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if response.ProtoMajor != 1 || conn.HandshakeProtocol() != RFC6455Handshake {
		t.Fatalf("fallback = HTTP/%d, %v", response.ProtoMajor, conn.HandshakeProtocol())
	}
	if err := conn.WriteText([]byte("three-tier fallback")); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestRFC9220DoesNotDowngradeCertificateFailure(t *testing.T) {
	certErr := &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}
	_, _, err := Dial(context.Background(), "wss://example.com/socket", DialOptions{
		HTTP3Transport: failingHTTP3Transport{err: certErr},
	})
	if !errors.Is(err, certErr) {
		t.Fatalf("certificate error = %v", err)
	}
}

func TestRFC9220UsesRegularHTTPCapabilityAndAlternatePort(t *testing.T) {
	var altSvc string
	errCh := make(chan error, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			w.Header().Set("Alt-Svc", altSvc)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		conn, _, err := Upgrade(w, r, UpgradeOptions{})
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()
		message, err := conn.ReadMessage()
		if err == nil {
			err = conn.WriteMessage(message)
		}
		errCh <- err
	})
	tcp := httptest.NewUnstartedServer(handler)
	tcp.EnableHTTP2 = true
	tcp.StartTLS()
	defer tcp.Close()
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	_, udpPort, _ := net.SplitHostPort(udp.LocalAddr().String())
	altSvc = `h3=":` + udpPort + `"; ma=60`
	h3server := &refhttp3.Server{TLSConfig: tcp.TLS.Clone(), Handler: handler}
	h3done := make(chan error, 1)
	go func() { h3done <- h3server.Serve(udp) }()
	defer func() {
		_ = h3server.Close()
		select {
		case <-h3done:
		case <-time.After(time.Second):
			t.Error("HTTP/3 server did not stop")
		}
	}()

	transport := &ghttp.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Fallback:        tcp.Client().Transport,
	}
	defer transport.CloseIdleConnections()
	ordinary, err := (&http.Client{Transport: transport}).Get(tcp.URL)
	if err != nil {
		t.Fatal(err)
	}
	ordinary.Body.Close()
	origin, _ := url.Parse(tcp.URL)
	if !transport.SupportsHTTP3(origin) {
		t.Fatal("regular HTTP did not publish learned HTTP/3 capability")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, response, err := Dial(ctx, "wss"+strings.TrimPrefix(tcp.URL, "https")+"/socket", DialOptions{
		HTTP3Transport: transport,
		TLSConfig:      &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		select {
		case serverErr := <-errCh:
			t.Fatalf("dial: %v (server: %v, response: %#v)", err, serverErr, response)
		default:
			t.Fatalf("dial: %v (response: %#v)", err, response)
		}
	}
	defer conn.Close()
	if response.ProtoMajor != 3 || conn.HandshakeProtocol() != RFC9220Handshake {
		t.Fatalf("protocol = HTTP/%s, %v", strconv.Itoa(response.ProtoMajor), conn.HandshakeProtocol())
	}
	if err := conn.WriteText([]byte("shared capability")); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
