package gen

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExpandPatternsStopsAtNestedModules(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"pkg", "nested/child"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "go.mod"), []byte("module example.test/nested\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	dirs, err := expandPatterns(root, []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{root, filepath.Join(root, "pkg")}
	if !slices.Equal(dirs, want) {
		t.Fatalf("expanded directories = %q, want %q", dirs, want)
	}

	nested, err := expandPatterns(filepath.Join(root, "nested"), []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	wantNested := []string{filepath.Join(root, "nested"), filepath.Join(root, "nested", "child")}
	if !slices.Equal(nested, wantNested) {
		t.Fatalf("nested expanded directories = %q, want %q", nested, wantNested)
	}
}
