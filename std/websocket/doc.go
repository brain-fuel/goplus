// Package websocket implements RFC 6455 and RFC 7692 WebSocket clients,
// servers, framing, masking, message assembly, close validation, and bounded
// decompression.
//
// Dial and Upgrade are the normal transport entry points. Message is a closed
// Go+ sum type, so applications handle text, binary, ping, pong, and close
// without integer message-kind constants. Conn permits one concurrent reader
// and one concurrent writer and serializes writes to prevent frame interleave.
//
// Low-level users can use ParseHeader, AppendHeader, Mask, and Assembler. Those
// APIs contain the same validation used by Conn and are allocation-free on the
// framing hot path.
package websocket
