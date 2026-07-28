package charset

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func TestDetectBOM(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"utf8", []byte{0xEF, 0xBB, 0xBF, 'a'}, "UTF-8"},
		{"utf16be", []byte{0xFE, 0xFF, 0, 'a'}, "UTF-16"},
		{"utf16le", []byte{0xFF, 0xFE, 'a', 0}, "UTF-16"},
		{"utf32be", []byte{0x00, 0x00, 0xFE, 0xFF}, "UTF-32"},
		{"utf32le", []byte{0xFF, 0xFE, 0x00, 0x00}, "UTF-32"},
		{"none", []byte("hello"), ""},
		{"empty", nil, ""},
	}
	for _, c := range cases {
		if got := DetectBOM(c.data); got != c.want {
			t.Errorf("%s: DetectBOM = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestClean(t *testing.T) {
	cases := [][2]string{
		{"  UTF-8  ", "UTF-8"},
		{`"utf-8"`, "utf-8"},
		{"'ISO-8859-1'", "ISO-8859-1"},
		{"", ""},
	}
	for _, c := range cases {
		if got := Clean(c[0]); got != c[1] {
			t.Errorf("Clean(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestFromContentType(t *testing.T) {
	cases := [][2]string{
		{"text/html; charset=EUC-JP", "EUC-JP"},
		{`text/html; charset="utf-8"`, "utf-8"},
		{"text/html", ""},
		{"", ""},
		{"application/xml; charset=ISO-8859-1; foo=bar", "ISO-8859-1"},
	}
	for _, c := range cases {
		if got := FromContentType(c[0]); got != c[1] {
			t.Errorf("FromContentType(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestAttrValue(t *testing.T) {
	if got := AttrValue(`<?xml version="1.0" encoding="ISO-8859-1"?>`, "encoding"); got != "ISO-8859-1" {
		t.Errorf("AttrValue encoding = %q, want ISO-8859-1", got)
	}
	if got := AttrValue("no match here", "charset"); got != "" {
		t.Errorf("AttrValue absent = %q, want empty", got)
	}
}

func TestDecodeNativeFastPaths(t *testing.T) {
	// UTF-8 (BOM stripped)
	if got := Decode([]byte{0xEF, 0xBB, 0xBF, 'h', 'i'}, "utf-8"); got != "hi" {
		t.Errorf("utf-8 = %q, want hi", got)
	}
	// name folding: latin-1 aliases
	for _, name := range []string{"ISO-8859-1", "latin1", "ISO8859_1", "l1"} {
		if got := Decode([]byte{0xE9}, name); got != "é" { // 0xE9 = é in latin-1
			t.Errorf("Decode 0xE9 as %q = %q, want é", name, got)
		}
	}
	// Windows-1252 0x80 = Euro
	if got := Decode([]byte{0x80}, "windows-1252"); got != "€" {
		t.Errorf("windows-1252 0x80 = %q, want €", got)
	}
	// UTF-16 BE with BOM
	if got := Decode([]byte{0xFE, 0xFF, 0x00, 'A', 0x00, 'B'}, "UTF-16"); got != "AB" {
		t.Errorf("utf-16be = %q, want AB", got)
	}
	// UTF-16 LE with BOM
	if got := Decode([]byte{0xFF, 0xFE, 'A', 0x00, 'B', 0x00}, "UTF-16"); got != "AB" {
		t.Errorf("utf-16le = %q, want AB", got)
	}
}

func TestDecodeViaHtmlindex(t *testing.T) {
	// Shift_JIS: 0x82 0xA0 = あ. Routed through golang.org/x/text/encoding/htmlindex.
	got := Decode([]byte{0x82, 0xA0}, "Shift_JIS")
	if got != "あ" {
		t.Errorf("Shift_JIS = %q, want あ", got)
	}
}

func TestDecodeUnknownFallsBackToUtf8(t *testing.T) {
	if got := Decode([]byte("plain"), "no-such-charset-xyz"); got != "plain" {
		t.Errorf("unknown charset = %q, want plain (utf-8 best effort)", got)
	}
}

func TestDecodeLatin1(t *testing.T) {
	// every byte maps to the identically-numbered rune
	if got := DecodeLatin1([]byte{0x00, 0x41, 0xFF}); got != "\x00Aÿ" {
		t.Errorf("DecodeLatin1 = %q", got)
	}
}

func TestMaybeGunzip(t *testing.T) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte("compressed payload"))
	w.Close()
	if got := string(MaybeGunzip(buf.Bytes())); got != "compressed payload" {
		t.Errorf("MaybeGunzip = %q", got)
	}
	// non-gzip passes through unchanged
	if got := string(MaybeGunzip([]byte("plain"))); got != "plain" {
		t.Errorf("MaybeGunzip passthrough = %q", got)
	}
}
