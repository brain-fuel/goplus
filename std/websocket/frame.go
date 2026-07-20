package websocket

import (
	"encoding/binary"
	"errors"
	"io"
)

type Opcode byte

const (
	OpContinuation Opcode = 0x0
	OpText         Opcode = 0x1
	OpBinary       Opcode = 0x2
	OpClose        Opcode = 0x8
	OpPing         Opcode = 0x9
	OpPong         Opcode = 0xa
)

type Side byte

const (
	ServerSide Side = iota // receives masked frames, writes unmasked frames
	ClientSide             // receives unmasked frames, writes masked frames
)

// Header is the RFC 6455 frame header. Length is limited to 63 bits.
type Header struct {
	Length int64
	Mask   [4]byte
	Opcode Opcode
	FIN    bool
	RSV1   bool
	RSV2   bool
	RSV3   bool
	Masked bool
}

var (
	ErrNeedMoreData       = io.ErrUnexpectedEOF
	ErrInvalidOpcode      = errors.New("websocket: invalid opcode")
	ErrReservedBits       = errors.New("websocket: reserved bits set")
	ErrWrongMask          = errors.New("websocket: incorrect masking for peer role")
	ErrNonCanonicalLength = errors.New("websocket: non-canonical payload length")
	ErrControlFragmented  = errors.New("websocket: fragmented control frame")
	ErrControlTooLarge    = errors.New("websocket: control payload exceeds 125 bytes")
	ErrInvalidLength      = errors.New("websocket: invalid payload length")
)

func ValidOpcode(op Opcode) bool {
	return op == OpContinuation || op == OpText || op == OpBinary || op == OpClose || op == OpPing || op == OpPong
}

func IsControl(op Opcode) bool { return op&0x8 != 0 }

// ParseHeader parses and validates a frame header without allocation. side is
// the local side, so ServerSide requires a masked incoming frame.
func ParseHeader(src []byte, side Side, allowRSV1 bool) (h Header, consumed int, err error) {
	if len(src) < 2 {
		return h, 0, ErrNeedMoreData
	}
	b0, b1 := src[0], src[1]
	h.FIN = b0&0x80 != 0
	h.RSV1, h.RSV2, h.RSV3 = b0&0x40 != 0, b0&0x20 != 0, b0&0x10 != 0
	h.Opcode = Opcode(b0 & 0x0f)
	h.Masked = b1&0x80 != 0
	if !ValidOpcode(h.Opcode) {
		return h, 0, &failureDetail{cause: ErrInvalidOpcode, opcode: byte(h.Opcode)}
	}
	if (h.RSV1 && (!allowRSV1 || IsControl(h.Opcode) || h.Opcode == OpContinuation)) || h.RSV2 || h.RSV3 {
		return h, 0, &failureDetail{cause: ErrReservedBits, bits: (b0 >> 4) & 7}
	}
	if h.Masked != (side == ServerSide) {
		return h, 0, &failureDetail{cause: ErrWrongMask, expectMasked: side == ServerSide}
	}

	n := 2
	switch x := b1 & 0x7f; x {
	case 126:
		if len(src) < n+2 {
			return h, 0, ErrNeedMoreData
		}
		h.Length = int64(binary.BigEndian.Uint16(src[n : n+2]))
		n += 2
		if h.Length < 126 {
			return h, 0, ErrNonCanonicalLength
		}
	case 127:
		if len(src) < n+8 {
			return h, 0, ErrNeedMoreData
		}
		u := binary.BigEndian.Uint64(src[n : n+8])
		n += 8
		if u>>63 != 0 || u < 65536 {
			return h, 0, ErrNonCanonicalLength
		}
		h.Length = int64(u)
	default:
		h.Length = int64(x)
	}
	if IsControl(h.Opcode) {
		if !h.FIN {
			return h, 0, ErrControlFragmented
		}
		if h.Length > 125 {
			return h, 0, ErrControlTooLarge
		}
	}
	if h.Masked {
		if len(src) < n+4 {
			return h, 0, ErrNeedMoreData
		}
		copy(h.Mask[:], src[n:n+4])
		n += 4
	}
	return h, n, nil
}

// AppendHeader validates h and appends its canonical wire representation. It
// allocates only when dst lacks capacity.
func AppendHeader(dst []byte, h Header) ([]byte, error) {
	if !ValidOpcode(h.Opcode) {
		return dst, ErrInvalidOpcode
	}
	if h.Length < 0 {
		return dst, ErrInvalidLength
	}
	if h.RSV2 || h.RSV3 || (h.RSV1 && (IsControl(h.Opcode) || h.Opcode == OpContinuation)) {
		return dst, ErrReservedBits
	}
	if IsControl(h.Opcode) {
		if !h.FIN {
			return dst, ErrControlFragmented
		}
		if h.Length > 125 {
			return dst, ErrControlTooLarge
		}
	}
	return appendHeader(dst, h), nil
}

func appendHeader(dst []byte, h Header) []byte {
	b0 := byte(h.Opcode)
	if h.FIN {
		b0 |= 0x80
	}
	if h.RSV1 {
		b0 |= 0x40
	}
	if h.RSV2 {
		b0 |= 0x20
	}
	if h.RSV3 {
		b0 |= 0x10
	}
	mask := byte(0)
	if h.Masked {
		mask = 0x80
	}
	dst = append(dst, b0)
	switch {
	case h.Length < 126:
		dst = append(dst, mask|byte(h.Length))
	case h.Length <= 65535:
		dst = append(dst, mask|126, byte(h.Length>>8), byte(h.Length))
	default:
		dst = append(dst, mask|127,
			byte(h.Length>>56), byte(h.Length>>48), byte(h.Length>>40), byte(h.Length>>32),
			byte(h.Length>>24), byte(h.Length>>16), byte(h.Length>>8), byte(h.Length))
	}
	if h.Masked {
		dst = append(dst, h.Mask[:]...)
	}
	return dst
}
