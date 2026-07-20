package http3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"testing"
	"time"

	refqpack "github.com/quic-go/qpack"
	ref "github.com/quic-go/quic-go/http3"
)

type memoryStream struct{ *bytes.Reader }

func (s *memoryStream) Write(p []byte) (int, error) { return len(p), nil }
func (s *memoryStream) Close() error                { return nil }
func (s *memoryStream) flush() error                { return nil }
func (s *memoryStream) needsContextWatch() bool     { return true }
func (s *memoryStream) abortRead(uint64)            {}
func (s *memoryStream) abortWrite(uint64)           {}

func appendTestFrame(dst []byte, frameType uint64, payload []byte) []byte {
	dst, _ = AppendFrameHeader(dst, frameType, uint64(len(payload)))
	return append(dst, payload...)
}

func TestReadResponseHandlesInformationalAndTrailers(t *testing.T) {
	info, _ := AppendFieldSection(nil, []HeaderField{{Name: ":status", Value: "103"}, {Name: "link", Value: "</style.css>; rel=preload"}})
	final, _ := AppendFieldSection(nil, []HeaderField{{Name: ":status", Value: "200"}, {Name: "content-type", Value: "text/plain"}})
	trailers, _ := AppendFieldSection(nil, []HeaderField{{Name: "x-checksum", Value: "ok"}})
	var wire []byte
	wire = appendTestFrame(wire, frameTypeHeaders, info)
	wire = appendTestFrame(wire, frameTypeHeaders, final)
	wire = appendTestFrame(wire, frameTypeData, []byte("body"))
	wire = appendTestFrame(wire, frameTypeHeaders, trailers)

	gotInfo := 0
	trace := &httptrace.ClientTrace{Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
		gotInfo = code
		if header.Get("Link") == "" {
			t.Error("missing informational Link header")
		}
		return nil
	}}
	req, _ := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), http.MethodGet, "https://example.com/", nil)
	stream := &memoryStream{Reader: bytes.NewReader(wire)}
	response, reader, err := readResponse(stream, req)
	if err != nil {
		t.Fatal(err)
	}
	response.Trailer = make(http.Header)
	response.Body = &responseBody{stream: stream, reader: reader, response: response}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if gotInfo != 103 || response.StatusCode != 200 || string(body) != "body" || response.Trailer.Get("X-Checksum") != "ok" {
		t.Fatalf("info=%d status=%d body=%q trailers=%v", gotInfo, response.StatusCode, body, response.Trailer)
	}
}

func TestResponseRejectsDataAfterTrailers(t *testing.T) {
	trailers, _ := AppendFieldSection(nil, []HeaderField{{Name: "x-checksum", Value: "ok"}})
	wire := appendTestFrame(nil, frameTypeHeaders, trailers)
	wire = appendTestFrame(wire, frameTypeData, []byte("late"))
	stream := &memoryStream{Reader: bytes.NewReader(wire)}
	response := &http.Response{Trailer: make(http.Header)}
	body := &responseBody{stream: stream, reader: &byteReader{Reader: stream}, response: response}
	if _, err := io.ReadAll(body); err == nil || !strings.Contains(err.Error(), "after trailers") {
		t.Fatalf("error = %v, want DATA-after-trailers rejection", err)
	}
}

func TestResponseBodyEnforcesContentLength(t *testing.T) {
	for name, tc := range map[string]struct {
		payload  string
		expected int64
		wantErr  string
	}{
		"exact":  {payload: "body", expected: 4},
		"short":  {payload: "bad", expected: 4, wantErr: "unexpected EOF"},
		"excess": {payload: "extra", expected: 4, wantErr: "exceeds content length"},
	} {
		t.Run(name, func(t *testing.T) {
			wire := appendTestFrame(nil, frameTypeData, []byte(tc.payload))
			stream := &memoryStream{Reader: bytes.NewReader(wire)}
			body := &responseBody{stream: stream, reader: &byteReader{Reader: stream}, expected: tc.expected, enforceLength: true}
			got, err := io.ReadAll(body)
			if tc.wantErr == "" {
				if err != nil || string(got) != tc.payload {
					t.Fatalf("body=%q err=%v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error=%v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestResponseFieldValidation(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	for name, fields := range map[string][]refqpack.HeaderField{
		"duplicate status": {{Name: ":status", Value: "200"}, {Name: ":status", Value: "204"}},
		"late status":      {{Name: "x-test", Value: "value"}, {Name: ":status", Value: "200"}},
		"bad status":       {{Name: ":status", Value: "99"}},
		"uppercase name":   {{Name: ":status", Value: "200"}, {Name: "X-Test", Value: "value"}},
		"connection field": {{Name: ":status", Value: "200"}, {Name: "connection", Value: "close"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := responseFromFields(fields, req); err == nil {
				t.Fatal("invalid response fields accepted")
			}
		})
	}
}

func TestSuccessfulConnectIgnoresContentLength(t *testing.T) {
	req, _ := http.NewRequest(http.MethodConnect, "https://example.com/socket", nil)
	response, err := responseFromFields([]refqpack.HeaderField{
		{Name: ":status", Value: "200"},
		{Name: "content-length", Value: "not-a-length"},
	}, req)
	if err != nil {
		t.Fatal(err)
	}
	if response.ContentLength != -1 {
		t.Fatalf("CONNECT content length = %d", response.ContentLength)
	}
}

func TestRequestRejectsIllegalTE(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.Header.Set("TE", "gzip")
	if _, err := requestFields(req, false); err == nil {
		t.Fatal("illegal TE value accepted")
	}
	req.Header.Set("TE", "trailers")
	if _, err := requestFields(req, false); err != nil {
		t.Fatalf("TE trailers rejected: %v", err)
	}
	if _, err := encodeRequestHeadersFrame(req, false, maxFieldSectionSize); err != nil {
		t.Fatalf("direct encoder rejected TE trailers: %v", err)
	}
}

func TestDirectRequestHeadersFrameMatchesFieldEncoder(t *testing.T) {
	req, _ := http.NewRequest(http.MethodConnect, "https://example.com/chat?q=1", nil)
	req.Proto = "websocket"
	req.Header.Set("Authorization", "secret")
	req.Header.Set("Content-Length", "1")
	fields, err := requestFields(req, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range fields {
		if field.Name == "content-length" {
			t.Fatal("CONNECT request encoded Content-Length")
		}
	}
	want, err := encodeHeadersFrame(fields)
	if err != nil {
		t.Fatal(err)
	}
	got, err := encodeRequestHeadersFrame(req, true, maxFieldSectionSize)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("direct frame = %x, field frame = %x", got, want)
	}
}

func TestDirectRequestHeadersFrameHonorsPeerLimit(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	if _, err := encodeRequestHeadersFrame(req, false, 1); !errors.Is(err, ErrFieldSectionTooLarge) {
		t.Fatalf("peer field-section limit error = %v", err)
	}
}

func TestNativeClientEndpointUsesRouteSelectedAddress(t *testing.T) {
	endpoint, err := newNativeClientEndpoint("127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	addr := endpoint.LocalAddr().Addr()
	if !addr.IsValid() || addr.IsUnspecified() {
		t.Fatalf("native client endpoint address = %v, want a concrete route-selected address", addr)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := endpoint.Close(ctx); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type cancellationStream struct {
	readCode  chan uint64
	writeCode chan uint64
}

func (s *cancellationStream) Read([]byte) (int, error)    { return 0, io.EOF }
func (s *cancellationStream) Write(p []byte) (int, error) { return len(p), nil }
func (s *cancellationStream) Close() error                { return nil }
func (s *cancellationStream) flush() error                { return nil }
func (s *cancellationStream) needsContextWatch() bool     { return true }
func (s *cancellationStream) abortRead(code uint64)       { s.readCode <- code }
func (s *cancellationStream) abortWrite(code uint64)      { s.writeCode <- code }

func TestStreamContextCancellationUsesH3RequestCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream := &cancellationStream{readCode: make(chan uint64, 1), writeCode: make(chan uint64, 1)}
	watch := watchStreamContext(ctx, stream)
	cancel()
	defer watch.stop()
	for name, codes := range map[string]<-chan uint64{"read": stream.readCode, "write": stream.writeCode} {
		select {
		case code := <-codes:
			if code != errorRequestCancelled {
				t.Fatalf("%s cancellation code = %#x, want %#x", name, code, errorRequestCancelled)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s cancellation", name)
		}
	}
}

func TestTransportInteroperatesWithQuicGoServer(t *testing.T) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
	server := &ref.Server{
		TLSConfig: serverTLS,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.RequestURI() != "/native?q=1" || r.Header.Get("X-Test") != "field" {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "native HTTP/3")
		}),
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	defer func() { _ = server.Close(); <-done }()

	for _, backend := range []QUICBackend{QUICGo, XNetQUIC} {
		t.Run(map[QUICBackend]string{QUICGo: "quic-go", XNetQUIC: "x-net"}[backend], func(t *testing.T) {
			transport := &Transport{Backend: backend, TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			defer transport.Close()
			req, _ := http.NewRequest(http.MethodGet, "https://"+packetConn.LocalAddr().String()+"/native?q=1", nil)
			req.Header.Set("X-Test", "field")
			response, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.ProtoMajor != 3 || response.StatusCode != http.StatusOK || string(body) != "native HTTP/3" {
				t.Fatalf("response = HTTP/%d %d %q", response.ProtoMajor, response.StatusCode, body)
			}
		})
	}
}

func TestTransportStreamsRequestBody(t *testing.T) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
	server := &ref.Server{TLSConfig: serverTLS, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || r.ContentLength != 7 {
			http.Error(w, "bad body", 400)
			return
		}
		_, _ = w.Write(body)
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	defer func() { _ = server.Close(); <-done }()
	transport := &Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer transport.Close()
	response, err := (&http.Client{Transport: transport}).Post("https://"+packetConn.LocalAddr().String()+"/", "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != "payload" {
		t.Fatalf("body=%q err=%v status=%d", body, err, response.StatusCode)
	}
}

func TestParseSettings(t *testing.T) {
	payload := appendQUICVarint(nil, 0x21)
	payload = appendQUICVarint(payload, 99)
	payload = appendQUICVarint(payload, settingExtendedConnect)
	payload = appendQUICVarint(payload, 1)
	payload = appendQUICVarint(payload, settingMaxFieldSection)
	payload = appendQUICVarint(payload, 4096)
	settings, err := parseSettings(payload)
	if err != nil || !settings.extended || settings.maxFieldSection != 4096 {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	duplicate := append(payload, appendQUICVarint(appendQUICVarint(nil, settingExtendedConnect), 0)...)
	if _, err := parseSettings(duplicate); err == nil {
		t.Fatal("duplicate setting accepted")
	}
	reserved := appendQUICVarint(appendQUICVarint(nil, 0x2), 0)
	if _, err := parseSettings(reserved); err == nil {
		t.Fatal("reserved HTTP/2 setting accepted")
	}
	invalid := appendQUICVarint(appendQUICVarint(nil, settingExtendedConnect), 2)
	if _, err := parseSettings(invalid); err == nil {
		t.Fatal("invalid extended CONNECT setting accepted")
	}
}

func TestControlStreamRejectsForbiddenFrames(t *testing.T) {
	for name, tc := range map[string]struct {
		frameType uint64
		payload   []byte
		wantCode  uint64
	}{
		"duplicate settings": {frameType: frameTypeSettings, wantCode: errorSettings},
		"data":               {frameType: frameTypeData, wantCode: errorFrameUnexpected},
		"invalid goaway":     {frameType: 0x7, payload: appendQUICVarint(nil, 1), wantCode: errorID},
	} {
		t.Run(name, func(t *testing.T) {
			wire := appendTestFrame(nil, tc.frameType, tc.payload)
			code, err := validateControlFrames(&byteReader{Reader: bytes.NewReader(wire)})
			if err == nil || code != tc.wantCode {
				t.Fatalf("code=%#x err=%v, want code %#x", code, err, tc.wantCode)
			}
		})
	}
}

func TestTransportExtendedConnect(t *testing.T) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer packetConn.Close()
	serverErr := make(chan error, 1)
	server := &ref.Server{
		TLSConfig: serverTLS,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect || r.Proto != "websocket" || r.ProtoMajor != 3 {
				http.Error(w, "bad extended connect", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := http.NewResponseController(w).Flush(); err != nil {
				serverErr <- err
				return
			}
			var one [1]byte
			_, err := io.ReadFull(r.Body, one[:])
			if err == nil {
				_, err = w.Write(one[:])
			}
			serverErr <- err
		}),
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	defer func() { _ = server.Close(); <-done }()

	transport := &Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer transport.Close()
	reader, writer := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodConnect, "https://"+packetConn.LocalAddr().String()+"/socket", reader)
	req.Proto = "websocket"
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if _, err := writer.Write([]byte{'x'}); err != nil {
		t.Fatal(err)
	}
	var echoed [1]byte
	if _, err := io.ReadFull(response.Body, echoed[:]); err != nil {
		t.Fatal(err)
	}
	if echoed[0] != 'x' {
		t.Fatalf("echo = %q", echoed[:])
	}
	writer.Close()
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func BenchmarkRoundTripWarm(b *testing.B) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	server := &ref.Server{TLSConfig: serverTLS, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	b.Cleanup(func() { _ = server.Close(); _ = packetConn.Close(); <-done })
	url := "https://" + packetConn.LocalAddr().String() + "/bench"
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	b.Run("goplus", func(b *testing.B) {
		transport := &Transport{TLSClientConfig: tlsConfig}
		defer transport.Close()
		client := &http.Client{Transport: transport}
		warm, err := client.Get(url)
		if err != nil {
			b.Fatal(err)
		}
		warm.Body.Close()
		b.ResetTimer()
		for b.Loop() {
			response, err := client.Get(url)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	})
	b.Run("goplus-xnet", func(b *testing.B) {
		transport := &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig}
		defer transport.Close()
		client := &http.Client{Transport: transport}
		warm, err := client.Get(url)
		if err != nil {
			b.Fatal(err)
		}
		warm.Body.Close()
		b.ResetTimer()
		for b.Loop() {
			response, err := client.Get(url)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	})
	b.Run("quicgo", func(b *testing.B) {
		transport := &ref.Transport{TLSClientConfig: tlsConfig}
		defer transport.Close()
		client := &http.Client{Transport: transport}
		warm, err := client.Get(url)
		if err != nil {
			b.Fatal(err)
		}
		warm.Body.Close()
		b.ResetTimer()
		for b.Loop() {
			response, err := client.Get(url)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
		}
	})
}

func BenchmarkRoundTripParallel(b *testing.B) {
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	server := &ref.Server{TLSConfig: serverTLS, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	b.Cleanup(func() { _ = server.Close(); _ = packetConn.Close(); <-done })
	url := "https://" + packetConn.LocalAddr().String() + "/parallel"
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	for _, tc := range []struct {
		name          string
		makeTransport func() http.RoundTripper
	}{
		{name: "goplus", makeTransport: func() http.RoundTripper { return &Transport{TLSClientConfig: tlsConfig} }},
		{name: "goplus-xnet", makeTransport: func() http.RoundTripper { return &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig} }},
		{name: "quicgo", makeTransport: func() http.RoundTripper { return &ref.Transport{TLSClientConfig: tlsConfig} }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			transport := tc.makeTransport()
			client := &http.Client{Transport: transport}
			warm, err := client.Get(url)
			if err != nil {
				b.Fatal(err)
			}
			warm.Body.Close()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					response, err := client.Get(url)
					if err != nil {
						b.Error(err)
						return
					}
					_, _ = io.Copy(io.Discard, response.Body)
					response.Body.Close()
				}
			})
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

func BenchmarkRoundTripThroughput(b *testing.B) {
	payload := bytes.Repeat([]byte("0123456789abcdef"), 64<<10)
	seed := httptest.NewTLSServer(http.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	server := &ref.Server{TLSConfig: serverTLS, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) })}
	done := make(chan error, 1)
	go func() { done <- server.Serve(packetConn) }()
	b.Cleanup(func() { _ = server.Close(); _ = packetConn.Close(); <-done })
	url := "https://" + packetConn.LocalAddr().String() + "/throughput"
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	for _, tc := range []struct {
		name          string
		makeTransport func() http.RoundTripper
	}{
		{name: "goplus", makeTransport: func() http.RoundTripper { return &Transport{TLSClientConfig: tlsConfig} }},
		{name: "goplus-xnet", makeTransport: func() http.RoundTripper { return &Transport{Backend: XNetQUIC, TLSClientConfig: tlsConfig} }},
		{name: "quicgo", makeTransport: func() http.RoundTripper { return &ref.Transport{TLSClientConfig: tlsConfig} }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			transport := tc.makeTransport()
			defer transport.(interface{ Close() error }).Close()
			client := &http.Client{Transport: transport}
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()
			for b.Loop() {
				response, err := client.Get(url)
				if err != nil {
					b.Fatal(err)
				}
				if n, err := io.Copy(io.Discard, response.Body); err != nil || n != int64(len(payload)) {
					b.Fatalf("body bytes=%d err=%v", n, err)
				}
				response.Body.Close()
			}
		})
	}
}
