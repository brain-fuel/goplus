package websocket

import (
	"bytes"
	"io"
	"testing"

	gobwas "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
)

type resetReader struct {
	b []byte
	i int
}

func (r *resetReader) Read(p []byte) (int, error) {
	if r.i == len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

type sliceWriter struct{ b []byte }

func (w *sliceWriter) Write(p []byte) (int, error) {
	w.b = append(w.b[:0], p...)
	return len(p), nil
}

func BenchmarkParseHeaderTiny(b *testing.B) {
	raw := []byte{0x82, 0x80, 1, 2, 3, 4}
	b.Run("goplus", func(b *testing.B) {
		for b.Loop() {
			_, _, _ = ParseHeader(raw, ServerSide, false)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		r := resetReader{b: raw}
		for b.Loop() {
			r.i = 0
			_, _ = gobwas.ReadHeader(&r)
		}
	})
}

func BenchmarkParseHeader64K(b *testing.B) {
	raw := []byte{0x82, 0xff, 0, 0, 0, 0, 0, 1, 0, 0, 1, 2, 3, 4}
	b.Run("goplus", func(b *testing.B) {
		for b.Loop() {
			_, _, _ = ParseHeader(raw, ServerSide, false)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		r := resetReader{b: raw}
		for b.Loop() {
			r.i = 0
			_, _ = gobwas.ReadHeader(&r)
		}
	})
}

func BenchmarkAppendHeader(b *testing.B) {
	h := Header{FIN: true, Opcode: OpBinary, Length: 65536, Masked: true, Mask: [4]byte{1, 2, 3, 4}}
	b.Run("goplus", func(b *testing.B) {
		var storage [14]byte
		for b.Loop() {
			_, _ = AppendHeader(storage[:0], h)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		w := sliceWriter{b: make([]byte, 0, 14)}
		gh := gobwas.Header{Fin: true, OpCode: gobwas.OpBinary, Length: 65536, Masked: true, Mask: [4]byte{1, 2, 3, 4}}
		for b.Loop() {
			_ = gobwas.WriteHeader(&w, gh)
		}
	})
}

func BenchmarkMask1K(b *testing.B) {
	key := [4]byte{1, 2, 3, 4}
	b.Run("goplus", func(b *testing.B) {
		payload := make([]byte, 1024)
		for b.Loop() {
			Mask(payload, key, 0)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		payload := make([]byte, 1024)
		for b.Loop() {
			gobwas.Cipher(payload, key, 0)
		}
	})
}

func BenchmarkAppendFrame1K(b *testing.B) {
	payload := make([]byte, 1024)
	b.Run("goplus", func(b *testing.B) {
		storage := make([]byte, 0, len(payload)+14)
		h := Header{FIN: true, Opcode: OpBinary, Length: int64(len(payload))}
		for b.Loop() {
			_, _ = AppendFrame(storage[:0], h, payload)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		frame := gobwas.NewBinaryFrame(payload)
		for b.Loop() {
			_, _ = gobwas.CompileFrame(frame)
		}
	})
}

func BenchmarkDeflate1K(b *testing.B) {
	payload := bytes.Repeat([]byte("abcdefgh"), 128)
	b.Run("goplus", func(b *testing.B) {
		for b.Loop() {
			_, _ = deflateMessage(payload, 15)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		for b.Loop() {
			_, _ = wsflate.DefaultHelper.Compress(payload)
		}
	})
}

func BenchmarkInflate1K(b *testing.B) {
	payload := bytes.Repeat([]byte("abcdefgh"), 128)
	compressed, err := deflateMessage(payload, 15)
	if err != nil {
		b.Fatal(err)
	}
	refCompressed, err := wsflate.DefaultHelper.Compress(payload)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("goplus", func(b *testing.B) {
		for b.Loop() {
			_, _ = inflateMessage(compressed, 1<<20)
		}
	})
	b.Run("gobwas", func(b *testing.B) {
		for b.Loop() {
			_, _ = wsflate.DefaultHelper.Decompress(refCompressed)
		}
	})
}
