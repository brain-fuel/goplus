package http

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerServesOneHandlerOverHTTP2AndHTTP3(t *testing.T) {
	seed := httptest.NewTLSServer(nethttp.NotFoundHandler())
	serverTLS := seed.TLS.Clone()
	seed.Close()
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		tcp.Close()
		t.Fatal(err)
	}
	server := &Server{
		TLSConfig: serverTLS,
		Handler: nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			_, _ = io.WriteString(w, r.Proto)
		}),
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(tcp, udp) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
				t.Errorf("Serve: %v", err)
			}
		case <-ctx.Done():
			t.Error("server did not stop")
		}
	}()
	fallback := &nethttp.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		ForceAttemptHTTP2: true,
	}
	clientProtocols := new(nethttp.Protocols)
	clientProtocols.SetHTTP1(true)
	clientProtocols.SetHTTP2(true)
	fallback.Protocols = clientProtocols
	transport := &Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Fallback:        fallback,
	}
	defer transport.CloseIdleConnections()
	client := &nethttp.Client{Transport: transport}
	url := "https://" + tcp.Addr().String()
	first, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	firstBody, _ := io.ReadAll(first.Body)
	first.Body.Close()
	if first.ProtoMajor != 2 || string(firstBody) != "HTTP/2.0" {
		t.Fatalf("first = HTTP/%d %q, ALPN=%q", first.ProtoMajor, firstBody, first.TLS.NegotiatedProtocol)
	}
	second, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	secondBody, _ := io.ReadAll(second.Body)
	second.Body.Close()
	if second.ProtoMajor != 3 || string(secondBody) != "HTTP/3.0" {
		t.Fatalf("second = HTTP/%d %q", second.ProtoMajor, secondBody)
	}
}
