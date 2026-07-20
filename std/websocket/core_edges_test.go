package websocket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestClosePayloadEdges(t *testing.T) {
	if !ValidCloseCode(3000) || ValidCloseCode(999) {
		t.Fatal("application and below-range close-code classification")
	}
	if code, reason, err := ParseClosePayload(nil); err != nil || code != 0 || reason != "" {
		t.Fatalf("empty: %d %q %v", code, reason, err)
	}
	if _, _, err := ParseClosePayload([]byte{1}); !errors.Is(err, ErrInvalidClosePayload) {
		t.Fatalf("one byte: %v", err)
	}
	valid := []byte{byte(uint16(CloseNormalClosure) >> 8), byte(uint16(CloseNormalClosure) & 0xff), 'o', 'k'}
	if code, reason, err := ParseClosePayload(valid); err != nil || code != CloseNormalClosure || reason != "ok" {
		t.Fatalf("valid: %d %q %v", code, reason, err)
	}
	invalidUTF8 := []byte{byte(uint16(CloseNormalClosure) >> 8), byte(uint16(CloseNormalClosure) & 0xff), 0xff}
	if _, _, err := ParseClosePayload(invalidUTF8); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("UTF-8: %v", err)
	}
	for _, test := range []struct {
		code   CloseCode
		reason string
		want   error
	}{
		{0, "reason", ErrInvalidClosePayload},
		{0, "", nil},
		{1005, "", ErrInvalidCloseCode},
		{CloseNormalClosure, string([]byte{0xff}), ErrInvalidUTF8},
		{CloseNormalClosure, string(bytes.Repeat([]byte{'x'}, 124)), ErrControlTooLarge},
	} {
		_, err := AppendClosePayload(nil, test.code, test.reason)
		if !errors.Is(err, test.want) {
			t.Fatalf("AppendClosePayload(%d): %v want %v", test.code, err, test.want)
		}
	}
}

func TestAssemblerEveryProtocolEdge(t *testing.T) {
	frame := func(fin bool, rsv1 bool, op Opcode, payload []byte) Header {
		return Header{FIN: fin, RSV1: rsv1, Opcode: op, Length: int64(len(payload))}
	}
	assembler := Assembler{MaxMessage: 3}
	if _, err := assembler.Feed(Header{FIN: true, Opcode: OpBinary, Length: 2}, []byte{1}); !errors.Is(err, ErrNeedMoreData) {
		t.Fatalf("length: %v", err)
	}
	for _, op := range []Opcode{OpPing, OpPong, OpClose} {
		if _, err := assembler.Feed(frame(true, true, op, nil), nil); !errors.Is(err, ErrReservedBits) {
			t.Fatalf("RSV1 opcode %x: %v", op, err)
		}
	}
	if _, err := assembler.Feed(frame(true, false, OpClose, []byte{1}), []byte{1}); !errors.Is(err, ErrInvalidClosePayload) {
		t.Fatalf("close: %v", err)
	}
	if _, err := assembler.Feed(frame(true, false, OpContinuation, nil), nil); !errors.Is(err, ErrUnexpectedContinuation) {
		t.Fatalf("unexpected continuation: %v", err)
	}
	if _, err := assembler.Feed(frame(true, false, OpBinary, []byte{1, 2, 3, 4}), []byte{1, 2, 3, 4}); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("complete limit: %v", err)
	}
	if _, err := assembler.Feed(frame(false, false, OpBinary, []byte{1, 2}), []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.Feed(frame(false, false, OpBinary, nil), nil); !errors.Is(err, ErrExpectedContinuation) {
		t.Fatalf("new data during fragments: %v", err)
	}
	if _, err := assembler.Feed(frame(false, true, OpContinuation, nil), nil); !errors.Is(err, ErrReservedBits) {
		t.Fatalf("continuation RSV1: %v", err)
	}
	if _, err := assembler.Feed(frame(false, false, OpContinuation, []byte{3}), []byte{3}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.Feed(frame(true, false, OpContinuation, []byte{4}), []byte{4}); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("fragment limit: %v", err)
	}
	assembler.Reset()
	if _, err := assembler.Feed(frame(true, false, Opcode(3), nil), nil); !errors.Is(err, ErrInvalidOpcode) {
		t.Fatalf("opcode: %v", err)
	}
	assembler = Assembler{MaxMessage: 100}
	if _, err := assembler.Feed(frame(false, true, OpBinary, []byte{1}), []byte{1}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.Feed(frame(true, false, OpContinuation, nil), nil); !errors.Is(err, ErrReservedBits) {
		t.Fatalf("missing inflater: %v", err)
	}
	wantInflate := errors.New("inflate failed")
	assembler = Assembler{MaxMessage: 100, inflate: func([]byte, int64) ([]byte, error) { return nil, wantInflate }}
	if _, err := assembler.Feed(frame(false, true, OpBinary, []byte{1}), []byte{1}); err != nil {
		t.Fatal(err)
	}
	if _, err := assembler.Feed(frame(true, false, OpContinuation, nil), nil); !errors.Is(err, wantInflate) {
		t.Fatalf("fragment inflate failure: %v", err)
	}
	assembler = Assembler{MaxMessage: 100, inflate: func([]byte, int64) ([]byte, error) { return nil, wantInflate }}
	if _, err := assembler.Feed(frame(true, true, OpBinary, []byte{1}), []byte{1}); !errors.Is(err, wantInflate) {
		t.Fatalf("inflate failure: %v", err)
	}
	assembler.inflate = func([]byte, int64) ([]byte, error) { return []byte{0xff}, nil }
	if _, err := assembler.Feed(frame(true, true, OpText, []byte{1}), []byte{1}); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("inflated UTF-8: %v", err)
	}
	assembler = Assembler{MaxMessage: 1}
	if _, err := assembler.Feed(frame(false, false, OpBinary, []byte{1, 2}), []byte{1, 2}); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("initial fragment limit: %v", err)
	}
	assembler = Assembler{MaxMessage: 100}
	if _, err := assembler.Feed(frame(true, true, OpBinary, nil), nil); !errors.Is(err, ErrReservedBits) {
		t.Fatalf("complete missing inflater: %v", err)
	}
}

func TestHeaderTruncationAndReservedEdges(t *testing.T) {
	for _, wire := range [][]byte{
		nil,
		{0x82},
		{0x82, 126, 0},
		{0x82, 127, 0, 0, 0, 0},
	} {
		if _, _, err := ParseHeader(wire, ClientSide, false); !errors.Is(err, ErrNeedMoreData) {
			t.Fatalf("wire %x: %v", wire, err)
		}
	}
	if _, _, err := ParseHeader([]byte{0x82, 0x80, 1, 2, 3}, ServerSide, false); !errors.Is(err, ErrNeedMoreData) {
		t.Fatalf("truncated mask: %v", err)
	}
	tooLarge := make([]byte, 10)
	tooLarge[0], tooLarge[1] = 0x82, 127
	binary.BigEndian.PutUint64(tooLarge[2:], 1<<63)
	if _, _, err := ParseHeader(tooLarge, ClientSide, false); !errors.Is(err, ErrNonCanonicalLength) {
		t.Fatalf("63-bit length: %v", err)
	}
	for _, h := range []Header{
		{FIN: true, Opcode: OpBinary, Length: 0, RSV2: true},
		{FIN: true, Opcode: OpBinary, Length: 0, RSV3: true},
	} {
		if _, err := AppendHeader(nil, h); !errors.Is(err, ErrReservedBits) {
			t.Fatalf("reserved header: %v", err)
		}
	}
	if got := appendHeader(nil, Header{FIN: true, Opcode: OpBinary, RSV2: true, RSV3: true}); got[0]&0x30 != 0x30 {
		t.Fatalf("reserved encoding: %x", got)
	}
}

func TestHandshakeTokenEdges(t *testing.T) {
	if hasToken("keep-alive", "upgrade") || validToken("") || validToken("bad token") {
		t.Fatal("token classification")
	}
	r := validServerRequest()
	r.Header.Set("Sec-WebSocket-Protocol", "valid, bad token")
	if _, err := ValidateServerRequest(r); !errors.Is(err, ErrHandshake) {
		t.Fatalf("protocol token: %v", err)
	}
	r = validServerRequest()
	r.Header.Set("Sec-WebSocket-Key", "not-base64")
	if _, err := ValidateServerRequest(r); !errors.Is(err, ErrHandshake) {
		t.Fatalf("key encoding: %v", err)
	}
	r = validServerRequest()
	r.Header.Set("Connection", "keep-alive")
	if _, err := ValidateServerRequest(r); !errors.Is(err, ErrHandshake) {
		t.Fatalf("upgrade tokens: %v", err)
	}
}

func TestFailureDetailDefaultsAndCapabilityExtraction(t *testing.T) {
	for _, err := range []error{
		ErrReservedBits,
		ErrMessageTooLarge,
		&failureDetail{cause: ErrReservedBits, bits: 7},
		&failureDetail{cause: ErrMessageTooLarge, limit: 42},
	} {
		if FailureOf(err) == nil {
			t.Fatalf("FailureOf(%v)", err)
		}
	}
	conn := NewConn(&memoryTransport{}, ServerSide, nil, ConnConfig{})
	if capabilityConn(OpenCapability{Conn: conn}) != conn || capabilityConn(CloseSentCapability{Conn: conn}) != conn || capabilityConn(ClosedCapability{Conn: conn}) != conn {
		t.Fatal("capability extraction")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("nil capability did not panic")
		}
	}()
	_ = capabilityConn(nil)
}
