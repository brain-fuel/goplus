package http3

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	ref "github.com/quic-go/quic-go/http3"
	xquic "goforge.dev/goplus/std/internal/quic"
)

func TestXNetRawServerHandshake(t *testing.T) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	serverTLS.NextProtos = []string{"h3"}
	serverTLS.MinVersion = tls.VersionTLS13
	serverTLS.CurvePreferences = []tls.CurveID{tls.X25519}
	serverEndpoint, err := xquic.Listen("udp", "127.0.0.1:0", &xquic.Config{TLSConfig: serverTLS})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = serverEndpoint.Close(ctx)
	}()
	clientEndpoint, err := xquic.Listen("udp", "127.0.0.1:0", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = clientEndpoint.Close(ctx)
	}()
	clientTLS := &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h3"}, MinVersion: tls.VersionTLS13, CurvePreferences: []tls.CurveID{tls.X25519}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	accepted := make(chan error, 1)
	go func() { _, err := serverEndpoint.Accept(ctx); accepted <- err }()
	if _, err := clientEndpoint.Dial(ctx, "udp", serverEndpoint.LocalAddr().String(), &xquic.Config{TLSConfig: clientTLS}); err != nil {
		t.Fatal(err)
	}
	if err := <-accepted; err != nil {
		t.Fatal(err)
	}
}

func startXNetServer(t testing.TB, handler http.Handler) (string, *tls.Config) {
	t.Helper()
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	serverTLS.CurvePreferences = []tls.CurveID{tls.X25519}
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &XNetServer{TLSConfig: serverTLS, Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("server close: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("native HTTP/3 server did not stop")
		}
	})
	return "https://" + packetConn.LocalAddr().String(), serverTLS
}

func startQUICGoNativeServer(t testing.TB, handler http.Handler) (string, *tls.Config) {
	t.Helper()
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	serverTLS.CurvePreferences = []tls.CurveID{tls.X25519}
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &QUICGoServer{TLSConfig: serverTLS, Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("server close: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("native quic-go HTTP/3 server did not stop")
		}
	})
	return "https://" + packetConn.LocalAddr().String(), serverTLS
}

func startReferenceServer(t testing.TB, handler http.Handler) (string, *tls.Config) {
	t.Helper()
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	serverTLS.CurvePreferences = []tls.CurveID{tls.X25519}
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &ref.Server{TLSConfig: serverTLS.Clone(), Handler: handler}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = packetConn.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("reference HTTP/3 server did not stop")
		}
	})
	return "https://" + packetConn.LocalAddr().String(), serverTLS
}

func BenchmarkNativeStackRoundTrip(b *testing.B) {
	benchmarkNativeStack(b, false, false, 0)
}

func BenchmarkNativeStackRoundTripParallel(b *testing.B) {
	benchmarkNativeStack(b, true, false, 0)
}

func BenchmarkNativeStackHeaders(b *testing.B) {
	benchmarkNativeStack(b, false, true, 0)
}

func BenchmarkNativeStackPayload64K(b *testing.B) {
	benchmarkNativeStack(b, false, false, 64<<10)
}

func benchmarkNativeStack(b *testing.B, parallel, headerRich bool, responseSize int) {
	payload := make([]byte, responseSize)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if headerRich {
			for i := range 16 {
				w.Header().Set("X-Response-"+strconv.Itoa(i), "representative metadata value")
			}
		}
		if len(payload) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write(payload)
	})
	tlsConfig := &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}
	for _, tc := range []struct {
		name          string
		startServer   func(testing.TB, http.Handler) (string, *tls.Config)
		makeTransport func() http.RoundTripper
	}{
		{name: "goplus", startServer: startXNetServer, makeTransport: func() http.RoundTripper { return &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig} }},
		{name: "goplus-quicgo-client", startServer: startXNetServer, makeTransport: func() http.RoundTripper {
			return &Transport{Backend: QUICGo, TLSClientConfig: tlsConfig}
		}},
		{name: "goplus-quicgo-stack", startServer: startQUICGoNativeServer, makeTransport: func() http.RoundTripper {
			return &Transport{Backend: QUICGo, TLSClientConfig: tlsConfig}
		}},
		{name: "quicgo", startServer: startReferenceServer, makeTransport: func() http.RoundTripper { return &ref.Transport{TLSClientConfig: tlsConfig} }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			serverURL, _ := tc.startServer(b, handler)
			transport := tc.makeTransport()
			client := &http.Client{Transport: transport}
			warm, err := client.Get(serverURL)
			if err != nil {
				b.Fatal(err)
			}
			warm.Body.Close()
			b.ResetTimer()
			run := func() error {
				req, _ := http.NewRequest(http.MethodGet, serverURL, nil)
				if headerRich {
					for i := range 16 {
						req.Header.Set("X-Request-"+strconv.Itoa(i), "representative metadata value")
					}
				}
				response, err := client.Do(req)
				if err != nil {
					return err
				}
				_, _ = io.Copy(io.Discard, response.Body)
				return response.Body.Close()
			}
			if parallel {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						if err := run(); err != nil {
							b.Error(err)
							return
						}
					}
				})
			} else {
				for b.Loop() {
					if err := run(); err != nil {
						b.Fatal(err)
					}
				}
			}
			b.StopTimer()
			switch transport := transport.(type) {
			case *Transport:
				_ = transport.Close()
			case *ref.Transport:
				_ = transport.Close()
			}
		})
	}
}

func TestXNetServerInteroperatesWithClients(t *testing.T) {
	baseURL, _ := startXNetServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || r.ProtoMajor != 3 || r.URL.RequestURI() != "/native?q=1" || r.Header.Get("X-Test") != "field" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(append([]byte("native:"), body...))
	}))
	tlsConfig := &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}
	for _, tc := range []struct {
		name      string
		transport http.RoundTripper
		close     func() error
	}{
		{name: "goplus-xnet", transport: &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig}},
		{name: "quic-go", transport: &ref.Transport{TLSClientConfig: tlsConfig}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			switch transport := tc.transport.(type) {
			case *Transport:
				defer transport.Close()
			case *ref.Transport:
				defer transport.Close()
			}
			req, _ := http.NewRequest(http.MethodPost, baseURL+"/native?q=1", strings.NewReader("body"))
			req.Header.Set("X-Test", "field")
			response, err := tc.transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 200 || string(body) != "native:body" {
				t.Fatalf("status=%d body=%q err=%v", response.StatusCode, body, err)
			}
		})
	}
}

func TestQUICGoNativeServerInteroperatesWithClients(t *testing.T) {
	baseURL, _ := startQUICGoNativeServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || r.ProtoMajor != 3 || r.URL.RequestURI() != "/native?q=1" || r.Header.Get("X-Test") != "field" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_, _ = w.Write(append([]byte("native:"), body...))
	}))
	tlsConfig := &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}
	for _, tc := range []struct {
		name      string
		transport http.RoundTripper
	}{
		{name: "goplus-xnet", transport: &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig}},
		{name: "goplus-quicgo", transport: &Transport{Backend: QUICGo, TLSClientConfig: tlsConfig}},
		{name: "quic-go", transport: &ref.Transport{TLSClientConfig: tlsConfig}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			switch transport := tc.transport.(type) {
			case *Transport:
				defer transport.Close()
			case *ref.Transport:
				defer transport.Close()
			}
			req, _ := http.NewRequest(http.MethodPost, baseURL+"/native?q=1", strings.NewReader("body"))
			req.Header.Set("X-Test", "field")
			response, err := tc.transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != 200 || string(body) != "native:body" {
				t.Fatalf("status=%d body=%q err=%v", response.StatusCode, body, err)
			}
		})
	}
}

func TestXNetServerExtendedConnect(t *testing.T) {
	errCh := make(chan error, 1)
	baseURL, _ := startXNetServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Proto != "websocket" {
			errCh <- errors.New("missing extended CONNECT metadata")
			return
		}
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush()
		var one [2]byte
		_, err := io.ReadFull(r.Body, one[:])
		if err == nil {
			_, err = w.Write(one[:])
		}
		errCh <- err
	}))
	transport := &Transport{Backend: XNetQUIC, TLSClientConfig: &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}}
	defer transport.Close()
	reader, writer := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodConnect, baseURL+"/socket", reader)
	req.Proto = "websocket"
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Length") != "" {
		t.Fatalf("successful CONNECT exposed Content-Length %q", response.Header.Get("Content-Length"))
	}
	if _, err := writer.Write([]byte("xy")); err != nil {
		t.Fatal(err)
	}
	var echoed [2]byte
	if _, err := io.ReadFull(response.Body, echoed[:]); err != nil || string(echoed[:]) != "xy" {
		t.Fatalf("echo=%q err=%v", echoed, err)
	}
	_ = writer.Close()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestQUICGoNativeServerExtendedConnect(t *testing.T) {
	errCh := make(chan error, 1)
	baseURL, _ := startQUICGoNativeServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Proto != "websocket" {
			errCh <- errors.New("missing extended CONNECT metadata")
			return
		}
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush()
		var one [2]byte
		_, err := io.ReadFull(r.Body, one[:])
		if err == nil {
			_, err = w.Write(one[:])
		}
		errCh <- err
	}))
	transport := &Transport{Backend: QUICGo, TLSClientConfig: &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}}
	defer transport.Close()
	reader, writer := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodConnect, baseURL+"/chat", reader)
	request.Proto = "websocket"
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if response.Header.Get("Content-Length") != "" {
		t.Fatalf("successful CONNECT exposed Content-Length %q", response.Header.Get("Content-Length"))
	}
	if _, err := writer.Write([]byte("xy")); err != nil {
		t.Fatal(err)
	}
	var echoed [2]byte
	if _, err := io.ReadFull(response.Body, echoed[:]); err != nil || string(echoed[:]) != "xy" {
		t.Fatalf("echo = %q, err = %v", echoed, err)
	}
	_ = writer.Close()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestXNetServerInformationalResponseAndTrailers(t *testing.T) {
	baseURL, _ := startXNetServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Link", "</style.css>; rel=preload")
		w.WriteHeader(http.StatusEarlyHints)
		w.Header().Set("Trailer", "X-Checksum")
		_, _ = io.WriteString(w, "body")
		w.Header().Set("X-Checksum", "ok")
	}))
	transport := &Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, CurvePreferences: []tls.CurveID{tls.X25519}}}
	defer transport.Close()
	gotInfo := 0
	trace := &httptrace.ClientTrace{Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
		gotInfo = code
		return nil
	}}
	req, _ := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), http.MethodGet, baseURL, nil)
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || gotInfo != 103 || string(body) != "body" || response.Trailer.Get("X-Checksum") != "ok" {
		t.Fatalf("info=%d body=%q trailer=%q err=%v", gotInfo, body, response.Trailer.Get("X-Checksum"), err)
	}
}

func TestNativeStackRequestTrailers(t *testing.T) {
	baseURL, _ := startXNetServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "body" || r.Trailer.Get("X-Checksum") != "ok" {
			http.Error(w, "bad trailers", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	transport := &Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer transport.Close()
	req, _ := http.NewRequest(http.MethodPost, baseURL, strings.NewReader("body"))
	req.Trailer = http.Header{"X-Checksum": {"ok"}}
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestXNetServerCancelsRequestContextOnConnectionClose(t *testing.T) {
	entered := make(chan struct{})
	canceled := make(chan struct{})
	baseURL, _ := startXNetServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush()
		close(entered)
		<-r.Context().Done()
		close(canceled)
	}))
	transport := &Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	req, _ := http.NewRequest(http.MethodGet, baseURL, nil)
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	<-entered
	_ = transport.Close()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("request context was not canceled")
	}
}
