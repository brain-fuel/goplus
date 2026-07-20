package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"crypto/tls"
	"crypto/x509"

	"github.com/quic-go/quic-go/http3"
)

type roundTripFunc func(*nethttp.Request) (*nethttp.Response, error)

func (f roundTripFunc) RoundTrip(r *nethttp.Request) (*nethttp.Response, error) { return f(r) }

func response(proto int, header nethttp.Header) *nethttp.Response {
	return &nethttp.Response{StatusCode: 200, ProtoMajor: proto, Header: header, Body: nethttp.NoBody}
}

func TestTransportLearnsAltSvcThenUsesHTTP3(t *testing.T) {
	var h3Calls, fallbackCalls atomic.Int32
	tr := &Transport{
		HTTP3: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			h3Calls.Add(1)
			return response(3, make(nethttp.Header)), nil
		}),
		Fallback: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			fallbackCalls.Add(1)
			return response(2, nethttp.Header{"Alt-Svc": {`h3=":443"; ma=60`}}), nil
		}),
	}
	req, _ := nethttp.NewRequest(nethttp.MethodGet, "https://example.com/resource", nil)
	first, err := tr.RoundTrip(req)
	if err != nil || first.ProtoMajor != 2 {
		t.Fatalf("first response = %#v, %v", first, err)
	}
	second, err := tr.RoundTrip(req)
	if err != nil || second.ProtoMajor != 3 || h3Calls.Load() != 1 || fallbackCalls.Load() != 1 {
		t.Fatalf("second response = %#v, %v; h3=%d fallback=%d", second, err, h3Calls.Load(), fallbackCalls.Load())
	}
}

func TestTransportFallsBackWithReplayableBody(t *testing.T) {
	var gotBody string
	tr := &Transport{
		PriorKnowledge: true,
		HTTP3: roundTripFunc(func(r *nethttp.Request) (*nethttp.Response, error) {
			_, _ = io.ReadAll(r.Body)
			return nil, errors.New("udp unavailable")
		}),
		Fallback: roundTripFunc(func(r *nethttp.Request) (*nethttp.Response, error) {
			body, _ := io.ReadAll(r.Body)
			gotBody = string(body)
			return response(1, make(nethttp.Header)), nil
		}),
	}
	req, _ := nethttp.NewRequest(nethttp.MethodPost, "https://example.com", bytes.NewBufferString("replay me"))
	got, err := tr.RoundTrip(req)
	if err != nil || got.ProtoMajor != 1 || gotBody != "replay me" {
		t.Fatalf("response=%#v err=%v body=%q", got, err, gotBody)
	}
}

func TestTransportDoesNotSpeculativelyConsumeUnreplayableBody(t *testing.T) {
	var h3Calls atomic.Int32
	tr := &Transport{
		PriorKnowledge: true,
		HTTP3: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			h3Calls.Add(1)
			return nil, errors.New("unexpected")
		}),
		Fallback: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			return response(2, make(nethttp.Header)), nil
		}),
	}
	req, _ := nethttp.NewRequestWithContext(context.Background(), nethttp.MethodPost, "https://example.com", io.NopCloser(bytes.NewBufferString("once")))
	if _, err := tr.RoundTrip(req); err != nil || h3Calls.Load() != 0 {
		t.Fatalf("err=%v h3=%d", err, h3Calls.Load())
	}
}

func TestTransportDoesNotDowngradeCertificateFailure(t *testing.T) {
	certErr := &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}
	var fallbackCalls atomic.Int32
	tr := &Transport{
		PriorKnowledge: true,
		HTTP3:          roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) { return nil, certErr }),
		Fallback: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			fallbackCalls.Add(1)
			return response(2, make(nethttp.Header)), nil
		}),
	}
	req, _ := nethttp.NewRequest(nethttp.MethodGet, "https://example.com", nil)
	_, err := tr.RoundTrip(req)
	if !errors.Is(err, certErr) || fallbackCalls.Load() != 0 {
		t.Fatalf("err=%v fallback calls=%d", err, fallbackCalls.Load())
	}
}

func TestAltSvcExpiryAndRemoval(t *testing.T) {
	u, _ := url.Parse("https://example.com")
	tr := new(Transport)
	now := time.Now()
	tr.ObserveAltSvc(u, []string{`h3=":443"; ma=60`}, now)
	if !tr.SupportsHTTP3(u) {
		t.Fatal("h3 capability not recorded")
	}
	tr.ObserveAltSvc(u, []string{`h3=":443"; ma=0`}, now)
	if tr.SupportsHTTP3(u) {
		t.Fatal("ma=0 did not remove capability")
	}
	tr.ObserveAltSvc(u, []string{`h3=":8443"; ma=60`}, now)
	if !tr.SupportsHTTP3(u) {
		t.Fatal("different-port alternative was not accepted")
	}
	tr.ObserveAltSvc(u, []string{"clear"}, now)
	if tr.SupportsHTTP3(u) {
		t.Fatal("Alt-Svc clear did not remove capability")
	}
	tr.ObserveAltSvc(u, []string{`h3=":443"; ma=invalid`}, now)
	if tr.SupportsHTTP3(u) {
		t.Fatal("malformed max-age recorded capability")
	}
}

func TestBrokenHTTP3AlternativeIsNotImmediatelyRelearned(t *testing.T) {
	var h3Calls, fallbackCalls atomic.Int32
	tr := &Transport{
		HTTP3: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			h3Calls.Add(1)
			return nil, errors.New("UDP path failed")
		}),
		Fallback: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
			fallbackCalls.Add(1)
			return response(2, nethttp.Header{"Alt-Svc": {`h3=":443"; ma=3600`}}), nil
		}),
	}
	u, _ := url.Parse("https://example.com/resource")
	tr.MarkHTTP3(u, time.Now().Add(time.Hour))
	req, _ := nethttp.NewRequest(nethttp.MethodGet, u.String(), nil)
	for range 2 {
		if _, err := tr.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
	}
	if h3Calls.Load() != 1 || fallbackCalls.Load() != 2 || tr.SupportsHTTP3(u) {
		t.Fatalf("h3=%d fallback=%d supported=%v", h3Calls.Load(), fallbackCalls.Load(), tr.SupportsHTTP3(u))
	}
	tr.ObserveAltSvc(u, []string{`h3=":8443"; ma=60`}, time.Now())
	if !tr.SupportsHTTP3(u) {
		t.Fatal("different H3 alternative was suppressed by cooldown")
	}

	tr.markHTTP3Broken(u, time.Now().Add(-time.Second))
	tr.ObserveAltSvc(u, []string{`h3=":8443"; ma=60`}, time.Now())
	if !tr.SupportsHTTP3(u) {
		t.Fatal("expired broken-alternative cooldown prevented recovery")
	}
}

func TestTransportRealHTTP3AfterHTTP2AltSvc(t *testing.T) {
	handler := nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("X-Protocol", strconv.Itoa(r.ProtoMajor))
		_, _ = io.WriteString(w, "HTTP/"+strconv.Itoa(r.ProtoMajor))
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
	h3server := &http3.Server{TLSConfig: tcp.TLS.Clone(), Handler: handler}
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

	fallback := tcp.Client().Transport
	tr := &Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Fallback:        fallback,
	}
	client := &nethttp.Client{Transport: tr}
	first, err := client.Get(tcp.URL)
	if err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	if first.ProtoMajor != 2 {
		t.Fatalf("first protocol = HTTP/%d", first.ProtoMajor)
	}
	u, _ := url.Parse(tcp.URL)
	_, udpPort, err := net.SplitHostPort(udp.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	tr.ObserveAltSvc(u, []string{`h3=":` + udpPort + `"; ma=60`}, time.Now())
	second, err := client.Get(tcp.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Body.Close()
	body, err := io.ReadAll(second.Body)
	if err != nil {
		t.Fatal(err)
	}
	if second.ProtoMajor != 3 || string(body) != "HTTP/3" {
		t.Fatalf("second protocol = HTTP/%d, body=%q", second.ProtoMajor, body)
	}
}

func TestOwnedFallbackUsesTLSClientConfig(t *testing.T) {
	server := httptest.NewUnstartedServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		_, _ = io.WriteString(w, strconv.Itoa(r.ProtoMajor))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	transport := &Transport{Mode: HTTP2Or1Only, TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer transport.CloseIdleConnections()
	response, err := (&nethttp.Client{Transport: transport}).Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.ProtoMajor != 2 || string(body) != "2" {
		t.Fatalf("protocol=HTTP/%d body=%q err=%v", response.ProtoMajor, body, err)
	}
}

func TestRegularHTTPRealFallbackMatrix(t *testing.T) {
	for _, tc := range []struct {
		name        string
		enableHTTP2 bool
		wantMajor   int
	}{
		{name: "HTTP2", enableHTTP2: true, wantMajor: 2},
		{name: "HTTP1.1", enableHTTP2: false, wantMajor: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
				_, _ = io.WriteString(w, r.Proto)
			}))
			server.EnableHTTP2 = tc.enableHTTP2
			server.StartTLS()
			defer server.Close()

			transport := &Transport{
				PriorKnowledge:  true,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				HTTP3: roundTripFunc(func(*nethttp.Request) (*nethttp.Response, error) {
					return nil, errors.New("forced UDP failure")
				}),
			}
			defer transport.CloseIdleConnections()
			response, err := (&nethttp.Client{Transport: transport}).Get(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.ProtoMajor != tc.wantMajor || string(body) != response.Proto {
				t.Fatalf("protocol=%s body=%q err=%v", response.Proto, body, err)
			}
		})
	}
}
