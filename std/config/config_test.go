package config

import (
	"errors"
	"strings"
	"testing"
)

type testDecoder struct{}

func (testDecoder) Decode(_ []byte, base int) (int, error) { return base + 1, nil }

func TestFieldErrorsCollectPaths(t *testing.T) {
	err := Collect(At("release.tag", errors.New("required")), At("locks.ttl", errors.New("invalid")))
	if got := err.Error(); !strings.Contains(got, "release.tag: required") || !strings.Contains(got, "locks.ttl: invalid") {
		t.Fatal(got)
	}
}

func TestSchemaAppliesDefaultsBeforeDecode(t *testing.T) {
	v, err := (Schema[int]{Defaults: func() int { return 41 }, Decoder: testDecoder{}}).Load(nil)
	if err != nil || v != 42 {
		t.Fatalf("Load = %d, %v", v, err)
	}
}
