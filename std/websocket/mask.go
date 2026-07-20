package websocket

import (
	"encoding/binary"
	"math/bits"
)

// Mask applies RFC 6455 masking in place and returns the next mask offset.
// The 64-bit loop handles eight bytes per iteration and is safe on unaligned
// slices because encoding/binary performs portable loads and stores.
func maskPortable(payload []byte, key [4]byte, offset int) int {
	if len(payload) == 0 {
		return offset & 3
	}
	offset &= 3
	k := binary.LittleEndian.Uint32(key[:])
	k = bits.RotateLeft32(k, -8*offset)
	kw := uint64(k) | uint64(k)<<32
	i := 0
	for ; i+8 <= len(payload); i += 8 {
		binary.LittleEndian.PutUint64(payload[i:i+8], binary.LittleEndian.Uint64(payload[i:i+8])^kw)
	}
	for ; i < len(payload); i++ {
		payload[i] ^= byte(k >> (8 * (i & 3)))
	}
	return (offset + len(payload)) & 3
}
