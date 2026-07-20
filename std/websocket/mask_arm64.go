//go:build arm64

package websocket

import (
	"encoding/binary"
	"math/bits"
)

// maskBlocks XORs a multiple of 64 bytes using four 128-bit NEON registers.
//
//go:noescape
func maskBlocks(payload []byte, key uint32)

func Mask(payload []byte, key [4]byte, offset int) int {
	offset &= 3
	if len(payload) == 0 {
		return offset
	}
	k := binary.LittleEndian.Uint32(key[:])
	k = bits.RotateLeft32(k, -8*offset)
	n := len(payload) &^ 63
	if n != 0 {
		maskBlocks(payload[:n], k)
	}
	for i := n; i < len(payload); i++ {
		payload[i] ^= byte(k >> (8 * (i & 3)))
	}
	return (offset + len(payload)) & 3
}
