package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransparentSingleVariantEnum(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/transparent\n\ngo 1.26.0\n")
	source := `package sample
//goplus:derive off
//goplus:repr transparent
type Handle[n nat] enum { handleValue(ID int) Handle[n] }
func New(id nat) Handle[id] { return handleValue(int(id)) }
`
	writeRefinementTestFile(t, dir, "sample.gp", source)
	result, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("generation diagnostics: %+v", result.Diags)
	}
	generated, err := os.ReadFile(filepath.Join(dir, "sample_gp.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	if !strings.Contains(text, "type Handle = handleValue") {
		t.Fatalf("missing transparent alias:\n%s", text)
	}
	if strings.Contains(text, "type Handle interface") {
		t.Fatalf("transparent enum emitted an interface:\n%s", text)
	}
}

func TestTransparentEnumRejectsMultipleVariants(t *testing.T) {
	dir := t.TempDir()
	writeRefinementTestFile(t, dir, "go.mod", "module example.com/transparentbad\n\ngo 1.26.0\n")
	source := `package sample
//goplus:repr transparent
type Handle enum { First(ID int); Second(ID int) }
`
	writeRefinementTestFile(t, dir, "sample.gp", source)
	result, err := Run(Options{Dir: dir, Patterns: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok() {
		t.Fatal("invalid transparent enum unexpectedly accepted")
	}
	for _, diagnostic := range result.Diags {
		if strings.Contains(diagnostic.Msg, "requires exactly one variant") {
			return
		}
	}
	t.Fatalf("unexpected diagnostics: %+v", result.Diags)
}
