package websocket

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDialUpgradeQueryProtocolAndCompression(t *testing.T) {
	requestURI := make(chan string, 1)
	serverErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI <- r.URL.RequestURI()
		conn, protocol, err := Upgrade(w, r, UpgradeOptions{Protocols: []string{"assay.v1"}, Compression: &CompressionOptions{}})
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		if protocol != "assay.v1" {
			serverErr <- ErrHandshake
			return
		}
		message, err := conn.ReadMessage()
		if err == nil {
			err = conn.WriteMessage(message)
		}
		serverErr <- err
	}))
	defer server.Close()
	rawURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/socket/path?case=12%2E1&agent=Go%2B"
	conn, response, err := Dial(context.Background(), rawURL, DialOptions{Protocols: []string{"assay.v1"}, Compression: &CompressionOptions{}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	exts, parseErr := parseExtensions(response.Header.Get("Sec-WebSocket-Extensions"))
	if response.Header.Get("Sec-WebSocket-Protocol") != "assay.v1" || parseErr != nil || len(exts) != 1 || exts[0].name != perMessageDeflate {
		t.Fatalf("unexpected negotiation: %v", response.Header)
	}
	payload := strings.Repeat("semantic assay data ", 100)
	if err = conn.WriteMessage(TextMessage{Payload: []byte(payload)}); err != nil {
		t.Fatal(err)
	}
	message, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if text, ok := message.(TextMessage); !ok || string(text.Payload) != payload {
		t.Fatalf("message = %#v", message)
	}
	if got := <-requestURI; got != "/socket/path?case=12%2E1&agent=Go%2B" {
		t.Fatalf("request URI = %q", got)
	}
	if err = <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestDialHonorsContextDuringHTTPHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		close(accepted)
		defer conn.Close()
		_, _ = io.Copy(io.Discard, conn)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, _, err = Dial(ctx, "ws://"+listener.Addr().String()+"/", DialOptions{})
	if err == nil || time.Since(started) > time.Second {
		t.Fatalf("error=%v elapsed=%v", err, time.Since(started))
	}
	<-accepted
}

func TestDialRejectsAmbiguousServerHandshake(t *testing.T) {
	tests := map[string]func(string) string{
		"duplicate accept": func(accept string) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
		},
		"multiple protocols": func(accept string) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\nSec-WebSocket-Protocol: one, two\r\n\r\n"
		},
		"unsolicited extension": func(accept string) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\nSec-WebSocket-Extensions: unknown\r\n\r\n"
		},
		"HTTP 1.0": func(accept string) string {
			return "HTTP/1.0 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n"
		},
	}
	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			go func() {
				conn, acceptErr := listener.Accept()
				if acceptErr != nil {
					return
				}
				defer conn.Close()
				request, readErr := http.ReadRequest(bufio.NewReader(conn))
				if readErr != nil {
					return
				}
				_, _ = io.WriteString(conn, response(AcceptKey(request.Header.Get("Sec-WebSocket-Key"))))
			}()
			_, _, err = Dial(context.Background(), "ws://"+listener.Addr().String()+"/", DialOptions{Protocols: []string{"one", "two"}})
			if err == nil {
				t.Fatal("ambiguous handshake was accepted")
			}
		})
	}
}

func TestWriteRejectsInvalidText(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	conn := NewConn(a, ClientSide, nil, ConnConfig{})
	if err := conn.WriteMessage(TextMessage{Payload: []byte{0xc3, 0x28}}); err != ErrInvalidUTF8 {
		t.Fatalf("error = %v", err)
	}
}
