package websocket

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type cancelResponseConn struct {
	request  bytes.Buffer
	response *bytes.Reader
	cancel   context.CancelFunc
}

func (c *cancelResponseConn) Write(p []byte) (int, error) { return c.request.Write(p) }
func (c *cancelResponseConn) Read(p []byte) (int, error) {
	if c.response == nil {
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(c.request.Bytes())))
		if err != nil {
			return 0, err
		}
		response := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + AcceptKey(request.Header.Get("Sec-WebSocket-Key")) + "\r\n\r\n"
		c.response = bytes.NewReader([]byte(response))
		c.cancel()
	}
	return c.response.Read(p)
}
func (*cancelResponseConn) Close() error                     { return nil }
func (*cancelResponseConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (*cancelResponseConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (*cancelResponseConn) SetDeadline(time.Time) error      { return nil }
func (*cancelResponseConn) SetReadDeadline(time.Time) error  { return nil }
func (*cancelResponseConn) SetWriteDeadline(time.Time) error { return nil }

func TestDialInputFailures(t *testing.T) {
	tests := []struct {
		name string
		url  string
		opts DialOptions
	}{
		{"invalid protocol", "ws://example.test", DialOptions{Protocols: []string{"bad protocol"}}},
		{"duplicate protocol", "ws://example.test", DialOptions{Protocols: []string{"chat", "chat"}}},
		{"invalid compression", "ws://example.test", DialOptions{Compression: &CompressionOptions{ClientMaxWindowBits: 7}}},
		{"invalid URL", "ws://%", DialOptions{}},
		{"scheme", "http://example.test", DialOptions{}},
		{"host", "ws:///socket", DialOptions{}},
		{"fragment", "ws://example.test/#fragment", DialOptions{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if conn, _, err := Dial(context.Background(), test.url, test.opts); err == nil || conn != nil {
				t.Fatalf("Dial(%q) = %v, %v", test.url, conn, err)
			}
		})
	}
	for _, rawURL := range []string{"ws://127.0.0.1", "wss://127.0.0.1"} {
		if _, _, err := Dial(context.Background(), rawURL, DialOptions{}); err == nil {
			t.Fatalf("default port dial unexpectedly succeeded: %s", rawURL)
		}
	}
}

func TestDialTLSDefaultFailureAndConfiguredSuccess(t *testing.T) {
	handlerDone := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := Upgrade(w, r, UpgradeOptions{})
		if err == nil {
			_ = conn.Close()
		}
		close(handlerDone)
	}))
	defer server.Close()
	url := "wss" + strings.TrimPrefix(server.URL, "https")
	if _, _, err := Dial(context.Background(), url, DialOptions{}); err == nil {
		t.Fatal("default TLS trusted a self-signed certificate")
	}
	conn, _, err := Dial(context.Background(), url, DialOptions{TLSConfig: &tls.Config{InsecureSkipVerify: true}}) // test-only certificate
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	<-handlerDone
}

func TestDialEntropyFailureClosesTransport(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()
	oldReader := rand.Reader
	rand.Reader = readerFunc(func([]byte) (int, error) { return 0, errors.New("entropy") })
	_, _, dialErr := Dial(context.Background(), "ws://"+listener.Addr().String(), DialOptions{})
	rand.Reader = oldReader
	if dialErr == nil || dialErr.Error() != "entropy" {
		t.Fatalf("entropy error: %v", dialErr)
	}
	conn := <-accepted
	if conn != nil {
		_ = conn.Close()
	}
}

func TestDialRequestWriteFailure(t *testing.T) {
	transport := &limitedConn{reader: strings.NewReader(""), remaining: 0, closed: make(chan struct{})}
	_, _, err := Dial(context.Background(), "ws://example.test/socket", DialOptions{
		DialContext: func(context.Context, string, string) (net.Conn, error) { return transport, nil },
	})
	if err == nil || err.Error() != "write failed" {
		t.Fatalf("request write error: %v", err)
	}
	select {
	case <-transport.closed:
	default:
		t.Fatal("failed transport was not closed")
	}
}

func TestDialCancellationAfterValidResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := &cancelResponseConn{cancel: cancel}
	_, _, err := Dial(ctx, "ws://example.test/socket", DialOptions{
		DialContext: func(context.Context, string, string) (net.Conn, error) { return transport, nil },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("post-response cancellation: %v", err)
	}
}

func TestDialRejectsUnsolicitedProtocolAndInvalidCompressionResponse(t *testing.T) {
	serve := func(extension string) (string, func()) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
			request, err := http.ReadRequest(bufio.NewReader(conn))
			if err != nil {
				return
			}
			response := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + AcceptKey(request.Header.Get("Sec-WebSocket-Key")) + "\r\n"
			if extension == "protocol" {
				response += "Sec-WebSocket-Protocol: unsolicited\r\n"
			} else {
				response += "Sec-WebSocket-Extensions: " + extension + "\r\n"
			}
			_, _ = io.WriteString(conn, response+"\r\n")
		}()
		return "ws://" + listener.Addr().String(), func() { _ = listener.Close() }
	}
	url, closeServer := serve("protocol")
	if _, _, err := Dial(context.Background(), url, DialOptions{Protocols: []string{"chat"}}); !errors.Is(err, ErrHandshake) {
		t.Fatalf("unsolicited protocol: %v", err)
	}
	closeServer()
	url, closeServer = serve("permessage-deflate; server_max_window_bits=7")
	if _, _, err := Dial(context.Background(), url, DialOptions{Compression: &CompressionOptions{ServerMaxWindowBits: 15}}); !errors.Is(err, ErrInvalidExtension) {
		t.Fatalf("invalid extension response: %v", err)
	}
	closeServer()
}
