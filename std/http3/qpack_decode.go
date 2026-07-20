package http3

import (
	"errors"
	"fmt"
	"io"
	"unsafe"

	refqpack "github.com/quic-go/qpack"
	"golang.org/x/net/http2/hpack"
)

var errDynamicQPACKUnsupported = errors.New("http3: dynamic QPACK table is unsupported")

// decodeFieldSection decodes the zero-capacity QPACK representation negotiated
// by this implementation. Raw strings retain the field-section buffer, avoiding
// one allocation and copy per literal name and value.
func decodeFieldSection(block []byte) ([]refqpack.HeaderField, error) {
	pos := 0
	required, err := readQPACKInt(block, &pos, 8)
	if err != nil {
		return nil, err
	}
	if required != 0 {
		return nil, errDynamicQPACKUnsupported
	}
	base, err := readQPACKInt(block, &pos, 7)
	if err != nil {
		return nil, err
	}
	if base != 0 {
		return nil, errDynamicQPACKUnsupported
	}
	capacity := len(block) / 8
	if capacity < 1 {
		capacity = 1
	} else if capacity > 8 {
		capacity = 8
	}
	fields := make([]refqpack.HeaderField, 0, capacity)
	var decodedSize uint64
	for pos < len(block) {
		first := block[pos]
		switch {
		case first&0x80 != 0:
			if first&0x40 == 0 {
				return nil, errDynamicQPACKUnsupported
			}
			index, err := readQPACKInt(block, &pos, 6)
			if err != nil {
				return nil, err
			}
			field, ok := qpackStaticField(index)
			if !ok {
				return nil, fmt.Errorf("http3: invalid QPACK static index %d", index)
			}
			if err := addDecodedFieldSize(&decodedSize, field); err != nil {
				return nil, err
			}
			fields = append(fields, field)
		case first&0xc0 == 0x40:
			if first&0x10 == 0 {
				return nil, errDynamicQPACKUnsupported
			}
			index, err := readQPACKInt(block, &pos, 4)
			if err != nil {
				return nil, err
			}
			field, ok := qpackStaticField(index)
			if !ok {
				return nil, fmt.Errorf("http3: invalid QPACK static index %d", index)
			}
			field.Value, err = readQPACKString(block, &pos, 7, 0x80)
			if err != nil {
				return nil, err
			}
			if err := addDecodedFieldSize(&decodedSize, field); err != nil {
				return nil, err
			}
			fields = append(fields, field)
		case first&0xe0 == 0x20:
			name, err := readQPACKString(block, &pos, 3, 0x08)
			if err != nil {
				return nil, err
			}
			value, err := readQPACKString(block, &pos, 7, 0x80)
			if err != nil {
				return nil, err
			}
			field := refqpack.HeaderField{Name: name, Value: value}
			if err := addDecodedFieldSize(&decodedSize, field); err != nil {
				return nil, err
			}
			fields = append(fields, field)
		default:
			return nil, fmt.Errorf("http3: unsupported QPACK representation %#x", first)
		}
	}
	return fields, nil
}

func addDecodedFieldSize(total *uint64, field refqpack.HeaderField) error {
	size := uint64(len(field.Name) + len(field.Value) + 32)
	if size > maxFieldSectionSize-*total {
		return ErrFieldSectionTooLarge
	}
	*total += size
	return nil
}

func readQPACKInt(block []byte, pos *int, prefix uint8) (uint64, error) {
	if *pos >= len(block) {
		return 0, io.ErrUnexpectedEOF
	}
	mask := uint64(1<<prefix - 1)
	value := uint64(block[*pos]) & mask
	(*pos)++
	if value < mask {
		return value, nil
	}
	shift := uint(0)
	for {
		if *pos >= len(block) {
			return 0, io.ErrUnexpectedEOF
		}
		b := block[*pos]
		(*pos)++
		if shift >= 63 || uint64(b&0x7f) > (^uint64(0)-value)>>shift {
			return 0, errors.New("http3: overflowing QPACK integer")
		}
		value += uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return value, nil
		}
		shift += 7
	}
}

func readQPACKString(block []byte, pos *int, prefix uint8, huffmanMask byte) (string, error) {
	if *pos >= len(block) {
		return "", io.ErrUnexpectedEOF
	}
	huffman := block[*pos]&huffmanMask != 0
	length, err := readQPACKInt(block, pos, prefix)
	if err != nil {
		return "", err
	}
	if length > uint64(len(block)-*pos) {
		return "", io.ErrUnexpectedEOF
	}
	bytes := block[*pos : *pos+int(length)]
	*pos += int(length)
	if huffman {
		return hpack.HuffmanDecodeToString(bytes)
	}
	if len(bytes) == 0 {
		return "", nil
	}
	return unsafe.String(unsafe.SliceData(bytes), len(bytes)), nil
}

var qpackStaticTable = [...]refqpack.HeaderField{
	{Name: ":authority"}, {Name: ":path", Value: "/"}, {Name: "age", Value: "0"},
	{Name: "content-disposition"}, {Name: "content-length", Value: "0"}, {Name: "cookie"},
	{Name: "date"}, {Name: "etag"}, {Name: "if-modified-since"}, {Name: "if-none-match"},
	{Name: "last-modified"}, {Name: "link"}, {Name: "location"}, {Name: "referer"},
	{Name: "set-cookie"}, {Name: ":method", Value: "CONNECT"}, {Name: ":method", Value: "DELETE"},
	{Name: ":method", Value: "GET"}, {Name: ":method", Value: "HEAD"}, {Name: ":method", Value: "OPTIONS"},
	{Name: ":method", Value: "POST"}, {Name: ":method", Value: "PUT"}, {Name: ":scheme", Value: "http"},
	{Name: ":scheme", Value: "https"}, {Name: ":status", Value: "103"}, {Name: ":status", Value: "200"},
	{Name: ":status", Value: "304"}, {Name: ":status", Value: "404"}, {Name: ":status", Value: "503"},
	{Name: "accept", Value: "*/*"}, {Name: "accept", Value: "application/dns-message"},
	{Name: "accept-encoding", Value: "gzip, deflate, br"}, {Name: "accept-ranges", Value: "bytes"},
	{Name: "access-control-allow-headers", Value: "cache-control"},
	{Name: "access-control-allow-headers", Value: "content-type"},
	{Name: "access-control-allow-origin", Value: "*"}, {Name: "cache-control", Value: "max-age=0"},
	{Name: "cache-control", Value: "max-age=2592000"}, {Name: "cache-control", Value: "max-age=604800"},
	{Name: "cache-control", Value: "no-cache"}, {Name: "cache-control", Value: "no-store"},
	{Name: "cache-control", Value: "public, max-age=31536000"}, {Name: "content-encoding", Value: "br"},
	{Name: "content-encoding", Value: "gzip"}, {Name: "content-type", Value: "application/dns-message"},
	{Name: "content-type", Value: "application/javascript"}, {Name: "content-type", Value: "application/json"},
	{Name: "content-type", Value: "application/x-www-form-urlencoded"}, {Name: "content-type", Value: "image/gif"},
	{Name: "content-type", Value: "image/jpeg"}, {Name: "content-type", Value: "image/png"},
	{Name: "content-type", Value: "text/css"}, {Name: "content-type", Value: "text/html; charset=utf-8"},
	{Name: "content-type", Value: "text/plain"}, {Name: "content-type", Value: "text/plain;charset=utf-8"},
	{Name: "range", Value: "bytes=0-"}, {Name: "strict-transport-security", Value: "max-age=31536000"},
	{Name: "strict-transport-security", Value: "max-age=31536000; includesubdomains"},
	{Name: "strict-transport-security", Value: "max-age=31536000; includesubdomains; preload"},
	{Name: "vary", Value: "accept-encoding"}, {Name: "vary", Value: "origin"},
	{Name: "x-content-type-options", Value: "nosniff"}, {Name: "x-xss-protection", Value: "1; mode=block"},
	{Name: ":status", Value: "100"}, {Name: ":status", Value: "204"}, {Name: ":status", Value: "206"},
	{Name: ":status", Value: "302"}, {Name: ":status", Value: "400"}, {Name: ":status", Value: "403"},
	{Name: ":status", Value: "421"}, {Name: ":status", Value: "425"}, {Name: ":status", Value: "500"},
	{Name: "accept-language"}, {Name: "access-control-allow-credentials", Value: "FALSE"},
	{Name: "access-control-allow-credentials", Value: "TRUE"},
	{Name: "access-control-allow-headers", Value: "*"}, {Name: "access-control-allow-methods", Value: "get"},
	{Name: "access-control-allow-methods", Value: "get, post, options"},
	{Name: "access-control-allow-methods", Value: "options"},
	{Name: "access-control-expose-headers", Value: "content-length"},
	{Name: "access-control-request-headers", Value: "content-type"},
	{Name: "access-control-request-method", Value: "get"}, {Name: "access-control-request-method", Value: "post"},
	{Name: "alt-svc", Value: "clear"}, {Name: "authorization"},
	{Name: "content-security-policy", Value: "script-src 'none'; object-src 'none'; base-uri 'none'"},
	{Name: "early-data", Value: "1"}, {Name: "expect-ct"}, {Name: "forwarded"}, {Name: "if-range"},
	{Name: "origin"}, {Name: "purpose", Value: "prefetch"}, {Name: "server"},
	{Name: "timing-allow-origin", Value: "*"}, {Name: "upgrade-insecure-requests", Value: "1"},
	{Name: "user-agent"}, {Name: "x-forwarded-for"}, {Name: "x-frame-options", Value: "deny"},
	{Name: "x-frame-options", Value: "sameorigin"},
}

func qpackStaticField(index uint64) (refqpack.HeaderField, bool) {
	if index >= uint64(len(qpackStaticTable)) {
		return refqpack.HeaderField{}, false
	}
	return qpackStaticTable[index], true
}
