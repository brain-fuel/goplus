// Package http3 contains optimized HTTP/3 wire primitives used by the Go+
// HTTP transport. The primitives deliberately avoid mutable QPACK state, so a
// field section can be produced concurrently without blocking on an encoder
// stream.
package http3

import (
	"encoding/binary"
	"errors"
	"io"

	refqpack "github.com/quic-go/qpack"
	"golang.org/x/net/http2/hpack"
)

func decodeReferenceFieldSection(block []byte) ([]refqpack.HeaderField, error) {
	next := refqpack.NewDecoder().Decode(block)
	var fields []refqpack.HeaderField
	for {
		field, err := next()
		if err == io.EOF {
			return fields, nil
		}
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
}

// HeaderField is one QPACK field line.
type HeaderField struct {
	Name      string
	Value     string
	Sensitive bool
}

var ErrFieldSectionTooLarge = errors.New("http3: field section too large")

// AppendFieldSection appends a complete QPACK field section using only the
// static table and literal representations. It never mutates shared state and
// is therefore safe to call concurrently when dst is not shared.
func AppendFieldSection(dst []byte, fields []HeaderField) ([]byte, error) {
	if err := enforceFieldSectionLimit(fields, maxFieldSectionSize); err != nil {
		return nil, err
	}
	// Required Insert Count and Delta Base are both zero: no dynamic table.
	dst = append(dst, 0, 0)
	for i := range fields {
		var err error
		dst, err = appendField(dst, &fields[i])
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func appendField(dst []byte, field *HeaderField) ([]byte, error) {
	if field.Name == "" {
		return nil, errors.New("http3: empty field name")
	}
	exact, nameIndex, found := staticField(field.Name, field.Value)
	if exact >= 0 && !field.Sensitive {
		return appendPrefixedInt(dst, 0xc0, 6, uint64(exact)), nil
	}
	if found {
		first := byte(0x50)
		if field.Sensitive {
			first |= 0x20
		}
		dst = appendPrefixedInt(dst, first, 4, uint64(nameIndex))
		return appendString(dst, field.Value, 7), nil
	}
	first := byte(0x20)
	if field.Sensitive {
		first |= 0x10
	}
	dst = appendPrefixedInt(dst, first, 3, uint64(len(field.Name)))
	dst = append(dst, field.Name...)
	return appendString(dst, field.Value, 7), nil
}

func encodeHeadersFrame(fields []HeaderField) ([]byte, error) {
	return encodeHeadersFrameLimit(fields, maxFieldSectionSize)
}

func encodeHeadersFrameLimit(fields []HeaderField, limit uint64) ([]byte, error) {
	if err := enforceFieldSectionLimit(fields, limit); err != nil {
		return nil, err
	}
	frame := make([]byte, 9, 9+128)
	var err error
	frame, err = AppendFieldSection(frame, fields)
	if err != nil {
		return nil, err
	}
	return finishHeadersFrame(frame), nil
}

func enforceFieldSectionLimit(fields []HeaderField, limit uint64) error {
	limit = min(limit, uint64(maxFieldSectionSize))
	var total uint64
	for i := range fields {
		size := uint64(len(fields[i].Name) + len(fields[i].Value) + 32)
		if size > limit-total {
			return ErrFieldSectionTooLarge
		}
		total += size
	}
	return nil
}

func finishHeadersFrame(frame []byte) []byte {
	sectionLength := len(frame) - 9
	var prefix [9]byte
	prefixLength := EncodeHeadersFrameHeader(&prefix, uint64(sectionLength))
	copy(frame[:prefixLength], prefix[:prefixLength])
	copy(frame[prefixLength:], frame[9:])
	return frame[:prefixLength+sectionLength]
}

func appendString(dst []byte, value string, prefix uint8) []byte {
	// Short HTTP values gain little from Huffman coding and dominate request
	// latency. For longer values, require a material reduction before paying
	// the encoding cost.
	if len(value) >= 48 {
		length := hpack.HuffmanEncodeLength(value)
		if length+8 <= uint64(len(value)) {
			dst = appendPrefixedInt(dst, 0x80, prefix, length)
			return hpack.AppendHuffmanString(dst, value)
		}
	}
	dst = appendPrefixedInt(dst, 0, prefix, uint64(len(value)))
	return append(dst, value...)
}

func appendPrefixedInt(dst []byte, first byte, prefix uint8, value uint64) []byte {
	max := uint64(1<<prefix - 1)
	if value < max {
		return append(dst, first|byte(value))
	}
	dst = append(dst, first|byte(max))
	value -= max
	for value >= 128 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

// staticField returns an exact static index, a name index, and whether the
// name exists. The branch-oriented common path avoids nested map lookups.
func staticField(name, value string) (exact, nameIndex int, found bool) {
	switch name {
	case ":authority":
		return -1, 0, true
	case ":path":
		if value == "/" {
			return 1, 1, true
		}
		return -1, 1, true
	case ":method":
		switch value {
		case "CONNECT":
			return 15, 15, true
		case "DELETE":
			return 16, 15, true
		case "GET":
			return 17, 15, true
		case "HEAD":
			return 18, 15, true
		case "OPTIONS":
			return 19, 15, true
		case "POST":
			return 20, 15, true
		case "PUT":
			return 21, 15, true
		default:
			return -1, 15, true
		}
	case ":scheme":
		if value == "http" {
			return 22, 22, true
		}
		if value == "https" {
			return 23, 22, true
		}
		return -1, 22, true
	case ":status":
		switch value {
		case "100":
			return 63, 24, true
		case "103":
			return 24, 24, true
		case "200":
			return 25, 24, true
		case "204":
			return 64, 24, true
		case "206":
			return 65, 24, true
		case "302":
			return 66, 24, true
		case "304":
			return 26, 24, true
		case "400":
			return 67, 24, true
		case "403":
			return 68, 24, true
		case "404":
			return 27, 24, true
		case "421":
			return 69, 24, true
		case "425":
			return 70, 24, true
		case "500":
			return 71, 24, true
		case "503":
			return 28, 24, true
		default:
			return -1, 24, true
		}
	case "content-length":
		if value == "0" {
			return 4, 4, true
		}
		return -1, 4, true
	case "content-type":
		switch value {
		case "application/json":
			return 46, 44, true
		case "text/html; charset=utf-8":
			return 52, 44, true
		case "text/plain":
			return 53, 44, true
		case "text/plain;charset=utf-8":
			return 54, 44, true
		default:
			return -1, 44, true
		}
	case "accept":
		if value == "*/*" {
			return 29, 29, true
		}
		return -1, 29, true
	case "accept-encoding":
		if value == "gzip, deflate, br" {
			return 31, 31, true
		}
		return -1, 31, true
	case "authorization":
		return -1, 84, true
	case "cookie":
		return -1, 5, true
	case "date":
		return -1, 6, true
	case "location":
		return -1, 12, true
	case "origin":
		return -1, 90, true
	case "referer":
		return -1, 13, true
	case "server":
		return -1, 92, true
	case "set-cookie":
		return -1, 14, true
	case "user-agent":
		return -1, 95, true
	default:
		return -1, -1, false
	}
}

// AppendFrameHeader appends an RFC 9114 frame type and payload length.
func AppendFrameHeader(dst []byte, frameType, payloadLength uint64) ([]byte, error) {
	if frameType >= 1<<62 || payloadLength >= 1<<62 {
		return nil, ErrFieldSectionTooLarge
	}
	dst = appendQUICVarint(dst, frameType)
	dst = appendQUICVarint(dst, payloadLength)
	return dst, nil
}

// AppendHeadersFrameHeader appends a HEADERS frame header without dispatching
// on a runtime frame type.
func AppendHeadersFrameHeader(dst []byte, payloadLength uint64) ([]byte, error) {
	if payloadLength >= 1<<62 {
		return nil, ErrFieldSectionTooLarge
	}
	return appendHeadersFrameHeader(dst, payloadLength), nil
}

func appendHeadersFrameHeader(dst []byte, payloadLength uint64) []byte {
	switch {
	case payloadLength < 1<<6:
		return append(dst, frameTypeHeaders, byte(payloadLength))
	case payloadLength < 1<<14:
		return append(dst, frameTypeHeaders, byte(payloadLength>>8)|0x40, byte(payloadLength))
	case payloadLength < 1<<30:
		return append(dst, frameTypeHeaders, byte(payloadLength>>24)|0x80, byte(payloadLength>>16), byte(payloadLength>>8), byte(payloadLength))
	default:
		return append(dst, frameTypeHeaders, byte(payloadLength>>56)|0xc0, byte(payloadLength>>48), byte(payloadLength>>40), byte(payloadLength>>32), byte(payloadLength>>24), byte(payloadLength>>16), byte(payloadLength>>8), byte(payloadLength))
	}
}

// AppendDataFrameHeader appends a DATA frame header without dispatching on a
// runtime frame type.
func AppendDataFrameHeader(dst []byte, payloadLength uint64) ([]byte, error) {
	if payloadLength >= 1<<62 {
		return nil, ErrFieldSectionTooLarge
	}
	return appendDataFrameHeader(dst, payloadLength), nil
}

func appendDataFrameHeader(dst []byte, payloadLength uint64) []byte {
	switch {
	case payloadLength < 1<<6:
		return append(dst, frameTypeData, byte(payloadLength))
	case payloadLength < 1<<14:
		return append(dst, frameTypeData, byte(payloadLength>>8)|0x40, byte(payloadLength))
	case payloadLength < 1<<30:
		return append(dst, frameTypeData, byte(payloadLength>>24)|0x80, byte(payloadLength>>16), byte(payloadLength>>8), byte(payloadLength))
	default:
		return append(dst, frameTypeData, byte(payloadLength>>56)|0xc0, byte(payloadLength>>48), byte(payloadLength>>40), byte(payloadLength>>32), byte(payloadLength>>24), byte(payloadLength>>16), byte(payloadLength>>8), byte(payloadLength))
	}
}

// EncodeHeadersFrameHeader writes a HEADERS frame header into dst and returns
// its length. payloadLength must be less than 2^62. Fixed-buffer encoding is
// intended for QUIC writers that already own per-stream scratch space.
func EncodeHeadersFrameHeader(dst *[9]byte, payloadLength uint64) int {
	dst[0] = frameTypeHeaders
	switch {
	case payloadLength < 1<<6:
		dst[1] = byte(payloadLength)
		return 2
	case payloadLength < 1<<14:
		dst[1], dst[2] = byte(payloadLength>>8)|0x40, byte(payloadLength)
		return 3
	case payloadLength < 1<<30:
		binary.BigEndian.PutUint32(dst[1:5], uint32(payloadLength)|0x80000000)
		return 5
	default:
		binary.BigEndian.PutUint64(dst[1:9], payloadLength|0xc000000000000000)
		return 9
	}
}

// EncodeDataFrameHeader writes a DATA frame header into dst and returns its
// length. payloadLength must be less than 2^62.
func EncodeDataFrameHeader(dst *[9]byte, payloadLength uint64) int {
	dst[0] = frameTypeData
	switch {
	case payloadLength < 1<<6:
		dst[1] = byte(payloadLength)
		return 2
	case payloadLength < 1<<14:
		dst[1], dst[2] = byte(payloadLength>>8)|0x40, byte(payloadLength)
		return 3
	case payloadLength < 1<<30:
		binary.BigEndian.PutUint32(dst[1:5], uint32(payloadLength)|0x80000000)
		return 5
	default:
		binary.BigEndian.PutUint64(dst[1:9], payloadLength|0xc000000000000000)
		return 9
	}
}

func appendQUICVarint(dst []byte, value uint64) []byte {
	switch {
	case value < 1<<6:
		return append(dst, byte(value))
	case value < 1<<14:
		return append(dst, byte(value>>8)|0x40, byte(value))
	case value < 1<<30:
		return append(dst, byte(value>>24)|0x80, byte(value>>16), byte(value>>8), byte(value))
	default:
		return append(dst, byte(value>>56)|0xc0, byte(value>>48), byte(value>>40), byte(value>>32), byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
	}
}
