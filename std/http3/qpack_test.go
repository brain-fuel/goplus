package http3

import (
	"errors"
	"reflect"
	"testing"

	ref "github.com/quic-go/qpack"
)

func TestAppendFieldSectionCrossDecodes(t *testing.T) {
	fields := []HeaderField{
		{Name: ":method", Value: "CONNECT"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/chat"},
		{Name: ":protocol", Value: "websocket"},
		{Name: "sec-websocket-version", Value: "13"},
		{Name: "authorization", Value: "secret", Sensitive: true},
	}
	encoded, err := AppendFieldSection(nil, fields)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeReferenceFieldSection(encoded)
	if err != nil {
		t.Fatal(err)
	}
	want := make([]ref.HeaderField, len(fields))
	for i, field := range fields {
		want[i] = ref.HeaderField{Name: field.Name, Value: field.Value}
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("decoded = %#v, want %#v", decoded, want)
	}
	native, err := decodeFieldSection(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(native, decoded) {
		t.Fatalf("native = %#v, reference = %#v", native, decoded)
	}
}

func TestNativeDecoderCoversQPACKStaticTable(t *testing.T) {
	for index := range qpackStaticTable {
		block := appendPrefixedInt([]byte{0, 0}, 0xc0, 6, uint64(index))
		native, err := decodeFieldSection(block)
		if err != nil {
			t.Fatalf("index %d: %v", index, err)
		}
		reference, err := decodeReferenceFieldSection(block)
		if err != nil {
			t.Fatalf("reference index %d: %v", index, err)
		}
		if !reflect.DeepEqual(native, reference) {
			t.Fatalf("index %d: native=%#v reference=%#v", index, native, reference)
		}
	}
}

func TestNativeDecoderEnforcesExpandedFieldSectionLimit(t *testing.T) {
	block := []byte{0, 0}
	field := qpackStaticTable[1]
	fieldSize := len(field.Name) + len(field.Value) + 32
	for range maxFieldSectionSize/fieldSize + 1 {
		block = append(block, 0xc1)
	}
	if _, err := decodeFieldSection(block); !errors.Is(err, ErrFieldSectionTooLarge) {
		t.Fatalf("expanded field section error = %v", err)
	}
}

func TestAppendFrameHeader(t *testing.T) {
	got, err := AppendFrameHeader(nil, 1, 16384)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 0x80, 0, 0x40, 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("header = %x, want %x", got, want)
	}
}

func TestEncodeHeadersFrame(t *testing.T) {
	fields := []HeaderField{
		{Name: ":method", Value: "CONNECT"},
		{Name: ":protocol", Value: "websocket"},
		{Name: "authorization", Value: "secret", Sensitive: true},
	}
	section, err := AppendFieldSection(nil, fields)
	if err != nil {
		t.Fatal(err)
	}
	want, err := AppendFrameHeader(nil, frameTypeHeaders, uint64(len(section)))
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, section...)
	got, err := encodeHeadersFrame(fields)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("frame = %x, want %x", got, want)
	}
}
