package websocket

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestFailureOfIsExhaustiveForPublicSentinels(t *testing.T) {
	tests := []struct {
		err  error
		want any
	}{
		{ErrNeedMoreData, NeedMoreData{}},
		{ErrInvalidOpcode, InvalidOpcode{}},
		{ErrReservedBits, ReservedBits{}},
		{ErrWrongMask, WrongMask{}},
		{ErrNonCanonicalLength, NonCanonicalLength{}},
		{ErrControlFragmented, ControlFragmented{}},
		{ErrControlTooLarge, ControlTooLarge{}},
		{ErrUnexpectedContinuation, UnexpectedContinuation{}},
		{ErrExpectedContinuation, ExpectedContinuation{}},
		{ErrInvalidUTF8, InvalidUTF8{}},
		{ErrInvalidClosePayload, InvalidClosePayload{}},
		{ErrInvalidCloseCode, InvalidCloseCode{}},
		{ErrMessageTooLarge, MessageTooLarge{}},
		{ErrHandshake, HandshakeRejected{}},
		{io.ErrClosedPipe, TransportFailed{}},
	}
	for _, test := range tests {
		got := FailureOf(errors.Join(test.err, errors.New("context")))
		if got == nil || reflect.TypeOf(got) != reflect.TypeOf(test.want) {
			t.Fatalf("error=%v failure=%T want=%T", test.err, got, test.want)
		}
	}
	if FailureOf(nil) != nil {
		t.Fatal("nil error produced a failure")
	}
}

func TestFailureOfPreservesProtocolDetails(t *testing.T) {
	_, _, err := ParseHeader([]byte{0x83, 0x00}, ClientSide, false)
	opcode, ok := FailureOf(err).(InvalidOpcode)
	if !ok || opcode.Opcode != 3 {
		t.Fatalf("failure=%#v", FailureOf(err))
	}
	_, _, err = ParseHeader([]byte{0x81, 0x00}, ServerSide, false)
	mask, ok := FailureOf(err).(WrongMask)
	if !ok || !mask.ExpectMasked {
		t.Fatalf("failure=%#v", FailureOf(err))
	}
	_, _, err = ParseClosePayload([]byte{0x03, 0xed}) // 1005
	closeCode, ok := FailureOf(err).(InvalidCloseCode)
	if !ok || closeCode.Code != 1005 {
		t.Fatalf("failure=%#v", FailureOf(err))
	}
}
