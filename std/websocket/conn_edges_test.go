package websocket

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

type testTransport struct {
	r        io.Reader
	w        io.Writer
	closeErr error
}

func (t *testTransport) Read(p []byte) (int, error)  { return t.r.Read(p) }
func (t *testTransport) Write(p []byte) (int, error) { return t.w.Write(p) }
func (t *testTransport) Close() error                { return t.closeErr }

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func transportBytes(input []byte, output io.Writer) *testTransport {
	return &testTransport{r: bytes.NewReader(input), w: output}
}

func TestConnConfigurationAndDeadlines(t *testing.T) {
	transport := transportBytes(nil, io.Discard)
	conn := NewConn(transport, ClientSide, nil, ConnConfig{MaxFrame: 10, MaxMessage: 5})
	if conn.maxFrame != 5 || conn.assembler.MaxMessage != 5 {
		t.Fatalf("limits: frame=%d message=%d", conn.maxFrame, conn.assembler.MaxMessage)
	}
	if conn.NetConn() != nil {
		t.Fatal("non-network transport exposed as net.Conn")
	}
	if err := conn.SetDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetWriteDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	network := NewConn(a, ClientSide, nil, ConnConfig{})
	if network.NetConn() != a {
		t.Fatal("network transport not exposed")
	}
	deadline := time.Now().Add(time.Second)
	if err := network.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if err := network.SetReadDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if err := network.SetWriteDeadline(deadline); err != nil {
		t.Fatal(err)
	}
}

func TestReadFrameEveryBoundaryFailure(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		max  int64
		want error
	}{
		{"base header", nil, 10, io.EOF},
		{"extended header", []byte{0x82, 126, 0}, 10, io.ErrUnexpectedEOF},
		{"invalid header", []byte{0x83, 0}, 10, ErrInvalidOpcode},
		{"frame limit", []byte{0x82, 126, 0, 126}, 125, ErrMessageTooLarge},
		{"payload", []byte{0x82, 2, 1}, 10, io.ErrUnexpectedEOF},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn := NewConn(transportBytes(test.wire, io.Discard), ClientSide, nil, ConnConfig{MaxFrame: test.max, MaxMessage: 10})
			_, _, err := conn.readFrame()
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
	wire := []byte{0x82, 127, 0, 0, 0, 0, 0, 1, 0, 0}
	conn := NewConn(transportBytes(wire, io.Discard), ClientSide, nil, ConnConfig{MaxFrame: 1, MaxMessage: 1})
	if _, _, err := conn.readFrame(); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("64-bit frame: %v", err)
	}
}

func TestReadMessageFragmentLoopAndProtocolFailures(t *testing.T) {
	wire := []byte{0x01, 1, 'a', 0x80, 1, 'b'}
	conn := NewConn(transportBytes(wire, io.Discard), ClientSide, nil, ConnConfig{})
	message, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if text, ok := message.(TextMessage); !ok || string(text.Payload) != "ab" {
		t.Fatalf("message=%#v", message)
	}
	for _, cause := range []error{ErrInvalidUTF8, ErrMessageTooLarge, ErrInvalidOpcode, io.EOF} {
		var output bytes.Buffer
		conn := NewConn(transportBytes(nil, &output), ServerSide, nil, ConnConfig{})
		conn.failProtocol(cause)
		if errors.Is(cause, io.EOF) {
			if output.Len() != 0 {
				t.Fatalf("transport error emitted close: %x", output.Bytes())
			}
		} else if output.Len() == 0 {
			t.Fatalf("protocol error %v emitted no close", cause)
		}
	}
	var output bytes.Buffer
	bad := NewConn(transportBytes([]byte{0x81, 0}, &output), ServerSide, nil, ConnConfig{})
	if _, err := bad.ReadMessage(); !errors.Is(err, ErrWrongMask) || output.Len() == 0 {
		t.Fatalf("read protocol failure: output=%x err=%v", output.Bytes(), err)
	}
	invalidText := NewConn(transportBytes([]byte{0x81, 1, 0xff}, io.Discard), ClientSide, nil, ConnConfig{})
	if _, err := invalidText.ReadMessage(); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("assembly failure: %v", err)
	}
	wantWrite := errors.New("pong write")
	ping := NewConn(transportBytes([]byte{0x89, 1, 'p'}, writerFunc(func([]byte) (int, error) { return 0, wantWrite })), ClientSide, nil, ConnConfig{})
	if _, err := ping.ReadMessage(); !errors.Is(err, wantWrite) {
		t.Fatalf("automatic pong failure: %v", err)
	}
}

func TestWriteVariantsAndFailures(t *testing.T) {
	var output bytes.Buffer
	conn := NewConn(transportBytes(nil, &output), ServerSide, nil, ConnConfig{})
	if err := conn.WriteBinary([]byte{1}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessageOwned(PingMessage{Payload: []byte("p")}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(PongMessage{Payload: []byte("p")}); err != nil {
		t.Fatal(err)
	}
	if err := conn.writeMessage(nil, false); err == nil {
		t.Fatal("nil message accepted")
	}
	if err := conn.WriteMessage(CloseMessage{Code: 1005}); !errors.Is(err, ErrInvalidCloseCode) {
		t.Fatalf("invalid close message: %v", err)
	}
	if err := conn.WriteClose(1005, ""); !errors.Is(err, ErrInvalidCloseCode) {
		t.Fatalf("invalid close: %v", err)
	}
	if err := conn.writeFrame(OpPing, bytes.Repeat([]byte{'x'}, 126)); !errors.Is(err, ErrControlTooLarge) {
		t.Fatalf("large control: %v", err)
	}
	if err := conn.WriteClose(CloseNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteBinary(nil); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("data after close: %v", err)
	}
	if err := conn.WriteClose(CloseNormalClosure, ""); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("second close: %v", err)
	}
}

func TestCompressionDefaultsAndWriteFailure(t *testing.T) {
	var output bytes.Buffer
	conn := NewConn(transportBytes(nil, &output), ServerSide, nil, ConnConfig{})
	conn.enableCompression(compressionSettings{})
	if conn.writeWindow != 15 {
		t.Fatalf("write window=%d", conn.writeWindow)
	}
	if err := conn.WriteBinary(bytes.Repeat([]byte("compress"), 32)); err != nil || output.Len() == 0 {
		t.Fatalf("compressed write: bytes=%d err=%v", output.Len(), err)
	}
	conn.writeWindow = 1
	if err := conn.WriteBinary(nil); err == nil {
		t.Fatal("invalid compression window accepted")
	}
}

func TestMaskEntropyAndWriterFailures(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = readerFunc(func([]byte) (int, error) { return 0, errors.New("entropy") })
	t.Cleanup(func() { rand.Reader = oldReader })
	client := NewConn(transportBytes(nil, io.Discard), ClientSide, nil, ConnConfig{})
	if err := client.WriteBinary(nil); err == nil || err.Error() != "entropy" {
		t.Fatalf("entropy error: %v", err)
	}
	rand.Reader = oldReader
	wantWrite := errors.New("write")
	failed := NewConn(transportBytes(nil, writerFunc(func([]byte) (int, error) { return 0, wantWrite })), ServerSide, nil, ConnConfig{})
	if err := failed.WriteBinary(nil); !errors.Is(err, wantWrite) {
		t.Fatalf("header write: %v", err)
	}
	if err := writeAll(writerFunc(func([]byte) (int, error) { return 0, nil }), []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("zero write: %v", err)
	}
	if err := writeAll(writerFunc(func(p []byte) (int, error) { return len(p) + 1, nil }), []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("oversized write: %v", err)
	}
	var chunks bytes.Buffer
	if err := writeAll(writerFunc(func(p []byte) (int, error) { return chunks.Write(p[:1]) }), []byte{1, 2, 3}); err != nil || !bytes.Equal(chunks.Bytes(), []byte{1, 2, 3}) {
		t.Fatalf("partial writes: %x %v", chunks.Bytes(), err)
	}
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestNetConnWritevPath(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	conn := NewConn(a, ServerSide, nil, ConnConfig{})
	done := make(chan error, 1)
	go func() { done <- conn.WriteBinary([]byte("wire")) }()
	header := make([]byte, 2)
	if _, err := io.ReadFull(b, header); err != nil {
		t.Fatal(err)
	}
	body := make([]byte, 4)
	if _, err := io.ReadFull(b, body); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil || string(body) != "wire" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}
