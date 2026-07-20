package websocket

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
)

func TestHeaderRoundTripAllBoundaries(t *testing.T) {
	lengths := []int64{0, 1, 125, 126, 127, 65535, 65536, 1 << 32, 1<<62 - 1}
	for _, side := range []Side{ClientSide, ServerSide} {
		for _, n := range lengths {
			h := Header{FIN: true, Opcode: OpBinary, Length: n, Masked: side == ServerSide}
			if h.Masked {
				h.Mask = [4]byte{1, 2, 3, 4}
			}
			wire, err := AppendHeader(make([]byte, 0, 14), h)
			if err != nil {
				t.Fatal(err)
			}
			got, consumed, err := ParseHeader(wire, side, false)
			if err != nil || consumed != len(wire) || got != h {
				t.Fatalf("side=%d length=%d got=%+v n=%d err=%v", side, n, got, consumed, err)
			}
		}
	}
}

func TestNonCanonicalLengths(t *testing.T) {
	for _, wire := range [][]byte{
		{0x82, 126, 0, 125},
		{0x82, 127, 0, 0, 0, 0, 0, 0, 0, 0xff},
		{0x82, 127, 0x80, 0, 0, 0, 0, 1, 0, 0},
	} {
		if _, _, err := ParseHeader(wire, ClientSide, false); !errors.Is(err, ErrNonCanonicalLength) {
			t.Fatalf("%x: %v", wire, err)
		}
	}
}

func TestAppendHeaderRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		header Header
		err    error
	}{
		{Header{FIN: true, Opcode: OpBinary, Length: -1}, ErrInvalidLength},
		{Header{FIN: true, Opcode: 3}, ErrInvalidOpcode},
		{Header{FIN: true, Opcode: OpPing, Length: 126}, ErrControlTooLarge},
		{Header{Opcode: OpPing}, ErrControlFragmented},
		{Header{FIN: true, RSV1: true, Opcode: OpClose}, ErrReservedBits},
		{Header{FIN: true, RSV1: true, Opcode: OpContinuation}, ErrReservedBits},
	}
	for _, test := range tests {
		if _, err := AppendHeader(nil, test.header); !errors.Is(err, test.err) {
			t.Fatalf("header=%+v error=%v want=%v", test.header, err, test.err)
		}
	}
}

func TestMaskMatchesRFCExampleAndOffsets(t *testing.T) {
	key := [4]byte{0x37, 0xfa, 0x21, 0x3d}
	payload := []byte("Hello")
	if off := Mask(payload, key, 0); off != 1 {
		t.Fatalf("offset=%d", off)
	}
	if !bytes.Equal(payload, []byte{0x7f, 0x9f, 0x4d, 0x51, 0x58}) {
		t.Fatalf("masked=%x", payload)
	}
	for offset := 0; offset < 4; offset++ {
		r := rand.New(rand.NewSource(int64(offset + 1)))
		data := make([]byte, 1031)
		_, _ = r.Read(data)
		original := append([]byte(nil), data...)
		Mask(data, key, offset)
		Mask(data, key, offset)
		if !bytes.Equal(data, original) {
			t.Fatalf("offset %d did not round trip", offset)
		}
	}
}

func TestMaskMatchesPortableAtEveryBoundary(t *testing.T) {
	key := [4]byte{0xde, 0xad, 0xbe, 0xef}
	for length := 0; length <= 512; length++ {
		for offset := 0; offset < 4; offset++ {
			want := make([]byte, length)
			for i := range want {
				want[i] = byte(i*31 + length)
			}
			got := append([]byte(nil), want...)
			wantOffset := maskPortable(want, key, offset)
			gotOffset := Mask(got, key, offset)
			if gotOffset != wantOffset || !bytes.Equal(got, want) {
				t.Fatalf("length=%d offset=%d next=%d wantNext=%d", length, offset, gotOffset, wantOffset)
			}
		}
	}
}

func FuzzHeaderRoundTrip(f *testing.F) {
	for _, n := range []int64{0, 125, 126, 65535, 65536, 1 << 32} {
		f.Add(n, byte(2), false)
	}
	f.Fuzz(func(t *testing.T, n int64, rawOp byte, masked bool) {
		if n < 0 || n > 1<<40 {
			return
		}
		opcodes := []Opcode{OpContinuation, OpText, OpBinary, OpClose, OpPing, OpPong}
		op := opcodes[int(rawOp)%len(opcodes)]
		if IsControl(op) && n > 125 {
			return
		}
		h := Header{FIN: true, Opcode: op, Length: n, Masked: masked}
		if masked {
			h.Mask = [4]byte{1, 2, 3, 4}
		}
		side := ClientSide
		if masked {
			side = ServerSide
		}
		wire, err := AppendHeader(nil, h)
		if err != nil {
			t.Fatal(err)
		}
		got, used, err := ParseHeader(wire, side, false)
		if err != nil || used != len(wire) || got != h {
			t.Fatalf("got=%+v used=%d err=%v", got, used, err)
		}
	})
}

func FuzzParseHeaderNeverPanics(f *testing.F) {
	f.Add([]byte{0x81, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseHeader(data, ClientSide, false)
		_, _, _ = ParseHeader(data, ServerSide, true)
	})
}
