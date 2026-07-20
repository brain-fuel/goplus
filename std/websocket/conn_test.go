package websocket

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestConnClientServerRoundTrip(t *testing.T) {
	clientNet, serverNet := net.Pipe()
	client := NewConn(clientNet, ClientSide, nil, ConnConfig{})
	server := NewConn(serverNet, ServerSide, nil, ConnConfig{})
	defer client.Close()
	defer server.Close()
	errCh := make(chan error, 1)
	go func() {
		m, err := server.ReadMessage()
		if err == nil {
			err = server.WriteMessage(m)
		}
		errCh <- err
	}()
	if err := client.WriteMessage(TextMessage{Payload: []byte("hello")}); err != nil {
		t.Fatal(err)
	}
	m, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	text, ok := m.(TextMessage)
	if !ok || string(text.Payload) != "hello" {
		t.Fatalf("message=%#v", m)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestConnSendsAtMostOneCloseAndRejectsLaterData(t *testing.T) {
	a, b := net.Pipe()
	conn := NewConn(a, ClientSide, nil, ConnConfig{})
	defer conn.Close()
	defer b.Close()
	drained := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, b)
		close(drained)
	}()
	if err := conn.WriteMessage(CloseMessage{Code: CloseNormalClosure}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(CloseMessage{Code: CloseNormalClosure}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("second close error=%v", err)
	}
	if err := conn.WriteMessage(TextMessage{Payload: []byte("late")}); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("late data error=%v", err)
	}
	_ = conn.Close()
	<-drained
}

func TestConnAutomaticallyAnswersPing(t *testing.T) {
	a, b := net.Pipe()
	client := NewConn(a, ClientSide, nil, ConnConfig{})
	server := NewConn(b, ServerSide, nil, ConnConfig{})
	defer client.Close()
	defer server.Close()
	result := make(chan Message, 1)
	go func() { m, _ := server.ReadMessage(); result <- m }()
	if err := client.WriteMessage(PingMessage{Payload: []byte("p")}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	m, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	pong, ok := m.(PongMessage)
	if !ok || string(pong.Payload) != "p" {
		t.Fatalf("pong=%#v", m)
	}
	go client.WriteMessage(TextMessage{Payload: []byte("done")})
	if m := <-result; string(m.(TextMessage).Payload) != "done" {
		t.Fatalf("server=%#v", m)
	}
}

func TestConnCanSurfacePingForManualControl(t *testing.T) {
	a, b := net.Pipe()
	client := NewConn(a, ClientSide, nil, ConnConfig{})
	server := NewConn(b, ServerSide, nil, ConnConfig{ManualControl: true})
	defer client.Close()
	defer server.Close()
	writeErr := make(chan error, 1)
	go func() { writeErr <- client.WriteMessage(PingMessage{Payload: []byte("manual")}) }()
	message, err := server.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	ping, ok := message.(PingMessage)
	if !ok || string(ping.Payload) != "manual" {
		t.Fatalf("message=%#v", message)
	}
	if err = <-writeErr; err != nil {
		t.Fatal(err)
	}
}
