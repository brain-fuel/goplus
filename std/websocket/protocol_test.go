package websocket

import (
	"errors"
	"testing"
)

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
