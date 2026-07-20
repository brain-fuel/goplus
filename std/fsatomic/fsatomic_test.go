package fsatomic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileReplacesContents(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state")
	if err := WriteFile(p, []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(p, []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "two" {
		t.Fatalf("read = %q, %v", b, err)
	}
}
