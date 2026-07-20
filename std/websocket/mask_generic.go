//go:build !amd64 && !arm64

package websocket

func Mask(payload []byte, key [4]byte, offset int) int {
	return maskPortable(payload, key, offset)
}
