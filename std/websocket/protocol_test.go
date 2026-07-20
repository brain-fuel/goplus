package websocket

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
)

type memoryTransport struct {
	bytes.Buffer
	closed bool
}

func (m *memoryTransport) Read([]byte) (int, error) { return 0, io.EOF }
func (m *memoryTransport) Close() error             { m.closed = true; return nil }

type sliceError []string

func (e sliceError) Error() string { return fmt.Sprint([]string(e)) }

func TestFailureFoldCoversEveryVariant(t *testing.T) {
	cases := FailureCases[string]{
		NeedMoreData:           func() string { return "need" },
		InvalidOpcode:          func(byte) string { return "opcode" },
		ReservedBits:           func(byte) string { return "reserved" },
		WrongMask:              func(bool) string { return "mask" },
		NonCanonicalLength:     func() string { return "length" },
		ControlFragmented:      func() string { return "fragmented" },
		ControlTooLarge:        func() string { return "control-size" },
		UnexpectedContinuation: func() string { return "unexpected" },
		ExpectedContinuation:   func() string { return "expected" },
		InvalidUTF8:            func() string { return "utf8" },
		InvalidClosePayload:    func() string { return "close-payload" },
		InvalidCloseCode:       func(CloseCode) string { return "close-code" },
		MessageTooLarge:        func(int64) string { return "message-size" },
		HandshakeRejected:      func(int, string) string { return "handshake" },
		TransportFailed:        func(error) string { return "transport" },
	}
	tests := []struct {
		failure Failure
		want    string
	}{
		{NeedMoreData{}, "need"},
		{InvalidOpcode{Opcode: 3}, "opcode"},
		{ReservedBits{Bits: 4}, "reserved"},
		{WrongMask{ExpectMasked: true}, "mask"},
		{NonCanonicalLength{}, "length"},
		{ControlFragmented{}, "fragmented"},
		{ControlTooLarge{}, "control-size"},
		{UnexpectedContinuation{}, "unexpected"},
		{ExpectedContinuation{}, "expected"},
		{InvalidUTF8{}, "utf8"},
		{InvalidClosePayload{}, "close-payload"},
		{InvalidCloseCode{Code: 1005}, "close-code"},
		{MessageTooLarge{Limit: 42}, "message-size"},
		{HandshakeRejected{Status: 400}, "handshake"},
		{TransportFailed{Err: errors.New("network")}, "transport"},
	}
	for _, test := range tests {
		if got := FailureFold(test.failure, cases); got != test.want {
			t.Fatalf("failure=%T got=%q want=%q", test.failure, got, test.want)
		}
		if !FailureEqual(test.failure, test.failure) {
			t.Fatalf("failure %T not equal to itself", test.failure)
		}
	}
	if FailureEqual(InvalidUTF8{}, InvalidClosePayload{}) {
		t.Fatal("different failures compare equal")
	}
	if !FailureEqual(TransportFailed{Err: sliceError{"network"}}, TransportFailed{Err: sliceError{"network"}}) {
		t.Fatal("equal uncomparable dynamic errors compared unequal")
	}
}

func TestIndexedSessionTransitions(t *testing.T) {
	connecting := ConnectingSession{}
	open := Open(connecting)
	if _, ok := open.(OpenSession); !ok {
		t.Fatalf("open=%T", open)
	}
	sent := SentClose(open)
	if _, ok := sent.(CloseSentSession); !ok {
		t.Fatalf("sent=%T", sent)
	}
	if _, ok := FinishSent(sent).(ClosedSession); !ok {
		t.Fatal("sent close did not finish closed")
	}
	received := ReceivedClose(open)
	if _, ok := FinishReceived(received).(ClosedSession); !ok {
		t.Fatal("received close did not finish closed")
	}
}

func TestPhaseFoldAndEquality(t *testing.T) {
	cases := PhaseCases[int]{
		ConnectingPhase:    func() int { return 0 },
		OpenPhase:          func() int { return 1 },
		CloseSentPhase:     func() int { return 2 },
		CloseReceivedPhase: func() int { return 3 },
		ClosedPhase:        func() int { return 4 },
	}
	phases := []Phase{ConnectingPhase{}, OpenPhase{}, CloseSentPhase{}, CloseReceivedPhase{}, ClosedPhase{}}
	for index, phase := range phases {
		if got := PhaseFold(phase, cases); got != index {
			t.Fatalf("phase=%T got=%d", phase, got)
		}
		if !PhaseEqual(phase, phase) {
			t.Fatalf("phase %T not equal to itself", phase)
		}
	}
	if PhaseEqual(OpenPhase{}, ClosedPhase{}) {
		t.Fatal("different phases compare equal")
	}
}

func TestLinearIndexedCapabilityLifecycle(t *testing.T) {
	transport := &memoryTransport{}
	conn := NewConn(transport, ServerSide, nil, ConnConfig{})
	open := Capability(OpenCapability{Conn: conn})
	next, err := Send(LinOf(open), BinaryMessage{Payload: []byte("data")})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := next.(OpenCapability); !ok || transport.Len() == 0 {
		t.Fatalf("next=%T bytes=%d", next, transport.Len())
	}
	attempt := BeginClose(LinOf(next), CloseNormalClosure, "done")
	started, ok := attempt.(CloseStarted)
	if !ok {
		t.Fatalf("attempt=%#v", attempt)
	}
	closed, err := FinishClose(LinOf(started.Capability))
	if err != nil || !transport.closed {
		t.Fatalf("closed=%T transport.closed=%v err=%v", closed, transport.closed, err)
	}
	if _, ok := closed.(ClosedCapability); !ok {
		t.Fatalf("closed=%T", closed)
	}
}

func TestCapabilityPreservesOwnershipOnCloseFailure(t *testing.T) {
	conn := NewConn(&memoryTransport{}, ServerSide, nil, ConnConfig{})
	attempt := BeginClose(LinOf[Capability](OpenCapability{Conn: conn}), CloseNormalClosure, string(bytes.Repeat([]byte{'x'}, 124)))
	failed, ok := attempt.(CloseFailed)
	if !ok || !errors.Is(failed.Err, ErrControlTooLarge) {
		t.Fatalf("attempt=%#v", attempt)
	}
	if _, ok := failed.Capability.(OpenCapability); !ok {
		t.Fatalf("retained=%T", failed.Capability)
	}
}

func TestCapabilityRuntimeIndexGuardForGoCallers(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected runtime index guard panic")
		}
	}()
	_, _ = Send(LinOf[Capability](CloseSentCapability{}), BinaryMessage{})
}
