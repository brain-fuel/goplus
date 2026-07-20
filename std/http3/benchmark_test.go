package http3

import (
	"bytes"
	"net/http"
	"testing"

	ref "github.com/quic-go/qpack"
	"github.com/quic-go/quic-go/quicvarint"
)

var benchmarkFields = []HeaderField{
	{Name: ":method", Value: "CONNECT"},
	{Name: ":scheme", Value: "https"},
	{Name: ":authority", Value: "example.com"},
	{Name: ":path", Value: "/chat?room=blue"},
	{Name: ":protocol", Value: "websocket"},
	{Name: "sec-websocket-version", Value: "13"},
	{Name: "sec-websocket-protocol", Value: "chat.v3"},
}

func BenchmarkEncodeFrameHeader(b *testing.B) {
	b.Run("goplus", func(b *testing.B) {
		var storage [9]byte
		for b.Loop() {
			_ = EncodeHeadersFrameHeader(&storage, 64<<10)
		}
	})
	b.Run("quicgo", func(b *testing.B) {
		var storage [16]byte
		for b.Loop() {
			buf := quicvarint.Append(storage[:0], frameTypeHeaders)
			_ = quicvarint.Append(buf, 64<<10)
		}
	})
}

var benchmarkRegularFields = []HeaderField{
	{Name: ":method", Value: "GET"},
	{Name: ":scheme", Value: "https"},
	{Name: ":authority", Value: "api.example.com"},
	{Name: ":path", Value: "/v1/assays?limit=100"},
	{Name: "accept", Value: "application/json"},
	{Name: "user-agent", Value: "goplus-http3/1"},
	{Name: "authorization", Value: "Bearer token", Sensitive: true},
}

var benchmarkDecodedFields []ref.HeaderField
var benchmarkRequestFrame []byte

func BenchmarkEncodeRequestHeadersFrame(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1/assays?limit=100", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goplus-http3/1")
	req.Header.Set("Authorization", "Bearer token")
	b.Run("direct", func(b *testing.B) {
		for b.Loop() {
			benchmarkRequestFrame, _ = encodeRequestHeadersFrame(req, false, maxFieldSectionSize)
		}
	})
	b.Run("fields", func(b *testing.B) {
		for b.Loop() {
			fields, _ := requestFields(req, false)
			benchmarkRequestFrame, _ = encodeHeadersFrame(fields)
		}
	})
}

func BenchmarkDecodeRegularFields(b *testing.B) {
	encoded, err := AppendFieldSection(nil, benchmarkRegularFields)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("goplus", func(b *testing.B) {
		for b.Loop() {
			benchmarkDecodedFields, _ = decodeFieldSection(encoded)
		}
	})
	b.Run("quicgo", func(b *testing.B) {
		for b.Loop() {
			benchmarkDecodedFields, _ = decodeReferenceFieldSection(encoded)
		}
	})
}

func BenchmarkEncodeWebSocketFields(b *testing.B) {
	b.Run("goplus", func(b *testing.B) {
		buf := make([]byte, 0, 256)
		for b.Loop() {
			buf, _ = AppendFieldSection(buf[:0], benchmarkFields)
		}
	})
	b.Run("quicgo", func(b *testing.B) {
		var buf bytes.Buffer
		encoder := ref.NewEncoder(&buf)
		for b.Loop() {
			buf.Reset()
			for _, field := range benchmarkFields {
				_ = encoder.WriteField(ref.HeaderField{Name: field.Name, Value: field.Value})
			}
			_ = encoder.Close()
		}
	})
}

func BenchmarkEncodeRegularFields(b *testing.B) {
	b.Run("goplus", func(b *testing.B) {
		buf := make([]byte, 0, 256)
		for b.Loop() {
			buf, _ = AppendFieldSection(buf[:0], benchmarkRegularFields)
		}
	})
	b.Run("quicgo", func(b *testing.B) {
		var buf bytes.Buffer
		encoder := ref.NewEncoder(&buf)
		for b.Loop() {
			buf.Reset()
			for _, field := range benchmarkRegularFields {
				_ = encoder.WriteField(ref.HeaderField{Name: field.Name, Value: field.Value})
			}
			_ = encoder.Close()
		}
	})
}
