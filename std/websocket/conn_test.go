package websocket

import (
	"errors"
	"fmt"
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

func TestGoConvenienceWriters(t *testing.T) {
	clientNet, serverNet := net.Pipe()
	client := NewConn(clientNet, ClientSide, nil, ConnConfig{})
	server := NewConn(serverNet, ServerSide, nil, ConnConfig{ManualControl: true})
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	writes := []struct {
		write func() error
		want  Message
	}{
		{func() error { return client.WriteText([]byte("text")) }, TextMessage{Payload: []byte("text")}},
		{func() error { return client.WriteBinaryOwned([]byte{1, 2, 3}) }, BinaryMessage{Payload: []byte{1, 2, 3}}},
		{func() error { return client.WritePing([]byte("ping")) }, PingMessage{Payload: []byte("ping")}},
		{func() error { return client.WritePong([]byte("pong")) }, PongMessage{Payload: []byte("pong")}},
	}
	for _, test := range writes {
		done := make(chan error, 1)
		go func() { done <- test.write() }()
		got, err := server.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", test.want) {
			t.Fatalf("got %#v want %#v", got, test.want)
		}
	}
	if err := client.WriteTextOwned([]byte{0xff}); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("invalid text: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- client.WriteClose(CloseNormalClosure, "done") }()
	type readResult struct {
		message Message
		err     error
	}
	serverRead := make(chan readResult, 1)
	go func() {
		message, err := server.ReadMessage()
		serverRead <- readResult{message: message, err: err}
	}()
	_, _ = client.ReadMessage()
	result := <-serverRead
	message, err := result.message, result.err
	if !errors.Is(err, io.EOF) {
		t.Fatalf("close read: %v", err)
	}
	if closeMessage, ok := message.(CloseMessage); !ok || closeMessage.Code != CloseNormalClosure || closeMessage.Reason != "done" {
		t.Fatalf("close message: %#v", message)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
