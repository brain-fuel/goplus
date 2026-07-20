package http3

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	ref "github.com/quic-go/qpack"
)

func FuzzFieldSectionCrossDecode(f *testing.F) {
	f.Add("x-test", "value", false)
	f.Add("authorization", "Bearer secret", true)
	f.Fuzz(func(t *testing.T, name, value string, sensitive bool) {
		if name == "" || len(name) > 256 || len(value) > 4096 {
			t.Skip()
		}
		encoded, err := AppendFieldSection(nil, []HeaderField{{Name: name, Value: value, Sensitive: sensitive}})
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeReferenceFieldSection(encoded)
		if err != nil {
			t.Fatalf("reference rejected %x: %v", encoded, err)
		}
		want := []ref.HeaderField{{Name: name, Value: value}}
		if !reflect.DeepEqual(decoded, want) {
			t.Fatalf("decoded=%#v want=%#v", decoded, want)
		}
		native, err := decodeFieldSection(encoded)
		if err != nil || !reflect.DeepEqual(native, decoded) {
			t.Fatalf("native=%#v reference=%#v err=%v", native, decoded, err)
		}
	})
}

func FuzzFieldSectionDecoderParity(f *testing.F) {
	f.Add([]byte{0, 0, 0xd1})
	f.Add([]byte{0, 0, 0x21, 'x', 1, 'y'})
	f.Fuzz(func(t *testing.T, block []byte) {
		if len(block) > 8<<10 {
			t.Skip()
		}
		native, nativeErr := decodeFieldSection(block)
		reference, referenceErr := decodeReferenceFieldSection(block)
		if (nativeErr == nil) != (referenceErr == nil) {
			t.Fatalf("acceptance mismatch for %x: native=%v reference=%v", block, nativeErr, referenceErr)
		}
		if nativeErr == nil && !slices.Equal(native, reference) {
			t.Fatalf("value mismatch for %x: native=%#v reference=%#v", block, native, reference)
		}
	})
}

func FuzzQUICVarintRoundTrip(f *testing.F) {
	for _, seed := range []uint64{0, 63, 64, 16383, 16384, 1<<30 - 1, 1 << 30, 1<<62 - 1} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value uint64) {
		value &= 1<<62 - 1
		encoded := appendQUICVarint(nil, value)
		decoded, err := readQUICVarint(bytes.NewReader(encoded))
		if err != nil || decoded != value {
			t.Fatalf("value=%d encoded=%x decoded=%d err=%v", value, encoded, decoded, err)
		}
	})
}

func FuzzParseSettings(f *testing.F) {
	f.Add([]byte{settingExtendedConnect, 1})
	f.Add([]byte{0x21, 0})
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > 8<<10 {
			t.Skip()
		}
		_, _ = parseSettings(payload)
	})
}
