package websocket

import (
	"bytes"
	"errors"
	"testing"

	"github.com/klauspost/compress/flate"
)

func TestRFC7692Hello(t *testing.T) {
	compressed := []byte{0xf2, 0x48, 0xcd, 0xc9, 0xc9, 0x07, 0x00}
	got, err := inflateMessage(compressed, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Hello" {
		t.Fatalf("got %q", got)
	}
}

func TestCompressionAllRFCWindowSizes(t *testing.T) {
	want := bytes.Repeat([]byte("window-sensitive semantic payload"), 200)
	for bits := 8; bits <= 15; bits++ {
		compressed, err := deflateMessage(want, bits)
		if err != nil {
			t.Fatalf("bits %d: %v", bits, err)
		}
		got, err := inflateMessage(compressed, int64(len(want)))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("bits %d: round trip error=%v", bits, err)
		}
	}
}

func TestContextTakeoverAcrossMessages(t *testing.T) {
	first := bytes.Repeat([]byte("assay-context-0123456789"), 20)
	second := append([]byte(nil), first...)
	var output bytes.Buffer
	w, err := flate.NewWriterWindow(&output, 1<<9)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(payload []byte) []byte {
		output.Reset()
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
		wire := append([]byte(nil), output.Bytes()...)
		if len(wire) < 4 {
			t.Fatal("missing sync-flush tail")
		}
		return wire[:len(wire)-4]
	}
	one, two := encode(first), encode(second)
	decoder := &messageInflater{window: 9, context: true}
	got, err := decoder.inflate(one, 4096)
	if err != nil || !bytes.Equal(got, first) {
		t.Fatalf("first: error=%v", err)
	}
	got, err = decoder.inflate(two, 4096)
	if err != nil || !bytes.Equal(got, second) {
		t.Fatalf("second: error=%v", err)
	}
}

func TestCompressionNegotiation(t *testing.T) {
	opts := CompressionOptions{ClientMaxWindowBits: 9, ServerMaxWindowBits: 10, AllowServerContextTakeover: true}
	offer, err := compressionOffer(opts)
	if err != nil {
		t.Fatal(err)
	}
	if want := "permessage-deflate; client_no_context_takeover; server_max_window_bits=10; client_max_window_bits=9"; offer != want {
		t.Fatalf("offer=%q want=%q", offer, want)
	}
	ok, settings, err := acceptCompressionResponse("permessage-deflate; client_no_context_takeover; server_max_window_bits=9; client_max_window_bits=8", opts)
	if err != nil || !ok {
		t.Fatalf("accepted=%v settings=%+v error=%v", ok, settings, err)
	}
	if settings.writeWindow != 8 || settings.readWindow != 9 || !settings.readContext {
		t.Fatalf("settings=%+v", settings)
	}
	response, serverSettings, err := negotiateCompression(offer, CompressionOptions{ClientMaxWindowBits: 8, ServerMaxWindowBits: 9, AllowServerContextTakeover: true})
	if err != nil || response == "" {
		t.Fatalf("response=%q error=%v", response, err)
	}
	if serverSettings.writeWindow != 9 || serverSettings.readWindow != 8 {
		t.Fatalf("server settings=%+v", serverSettings)
	}
}

func TestCompressionRejectsInvalidNegotiation(t *testing.T) {
	invalid := []string{
		"permessage-deflate; client_max_window_bits=7",
		"permessage-deflate; client_max_window_bits=16",
		"permessage-deflate; unknown_parameter",
		"permessage-deflate; client_no_context_takeover=1",
		"permessage-deflate; client_max_window_bits=9; client_max_window_bits=10",
	}
	for _, header := range invalid {
		if _, _, err := acceptCompressionResponse(header, CompressionOptions{ClientMaxWindowBits: 15}); err == nil {
			t.Fatalf("accepted %q", header)
		}
	}
}

func TestCompressionRoundTripAndLimit(t *testing.T) {
	want := bytes.Repeat([]byte("high-integrity websocket data "), 100)
	compressed, err := deflateMessage(want, 15)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inflateMessage(compressed, int64(len(want)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("round trip changed payload")
	}
	if _, err = inflateMessage(compressed, int64(len(want)-1)); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("limit error = %v", err)
	}
}

func TestAssemblerCompressedFragmentedMessage(t *testing.T) {
	compressed, err := deflateMessage([]byte("Hello"), 15)
	if err != nil {
		t.Fatal(err)
	}
	a := Assembler{MaxMessage: 1024, inflate: inflateMessage}
	cut := len(compressed) / 2
	msg, err := a.Feed(Header{Opcode: OpText, RSV1: true, Length: int64(cut)}, compressed[:cut])
	if err != nil || msg != nil {
		t.Fatalf("first fragment: message=%v error=%v", msg, err)
	}
	msg, err = a.Feed(Header{Opcode: OpContinuation, FIN: true, Length: int64(len(compressed) - cut)}, compressed[cut:])
	if err != nil {
		t.Fatal(err)
	}
	text, ok := msg.(TextMessage)
	if !ok || string(text.Payload) != "Hello" {
		t.Fatalf("message = %#v", msg)
	}
}

func TestRSV1ForbiddenOnControlAndContinuation(t *testing.T) {
	a := Assembler{inflate: inflateMessage}
	for _, h := range []Header{
		{FIN: true, RSV1: true, Opcode: OpPing},
		{FIN: true, RSV1: true, Opcode: OpPong},
		{FIN: true, RSV1: true, Opcode: OpClose},
		{FIN: true, RSV1: true, Opcode: OpContinuation},
	} {
		if _, err := a.Feed(h, nil); err != ErrReservedBits {
			t.Fatalf("opcode %d: error = %v", h.Opcode, err)
		}
	}
}
