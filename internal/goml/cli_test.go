package goml

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cliOption = `module option

type Option (a : Type) :=
  | Some (value : a)
  | None

let UnwrapOr (o : Option a) (d : a) : a :=
  match o with
  | Some v => v
  | None => d
`

func TestCLIConvertPrintsGpText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "option.goml")
	if err := os.WriteFile(path, []byte(cliOption), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := CLIRun([]string{"convert", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "type Option[a any] enum {") {
		t.Fatalf("missing enum lowering:\n%s", stdout.String())
	}
}

func TestCLIGenAndCheck(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "option.goml"), []byte(cliOption), 0o644); err != nil {
		t.Fatal(err)
	}
	// Generating a match resolves it against type information, which needs
	// a module; without one gen refuses rather than writing a skeleton.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	var stdout, stderr bytes.Buffer
	if code := CLIRun([]string{"gen", "."}, &stdout, &stderr); code != 0 {
		t.Fatalf("gen exit %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "option_gml.go") {
		t.Fatalf("gen did not report the written file:\n%s", stdout.String())
	}
	stdout.Reset()
	if code := CLIRun([]string{"gen", "-check", "."}, &stdout, &stderr); code != 0 {
		t.Fatalf("fresh -check exit %d: %s%s", code, stdout.String(), stderr.String())
	}

	// Stale output: -check exits 1 and names it. The edit stays well-typed,
	// because the fixture is a module and so the output is type-checked.
	if err := os.WriteFile(filepath.Join(dir, "option.goml"),
		[]byte(strings.Replace(cliOption, "| Some v => v", "| Some x => x", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := CLIRun([]string{"gen", "-check", "."}, &stdout, &stderr); code != 1 {
		t.Fatalf("stale -check exit %d: %s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "option_gml.go") {
		t.Fatalf("stale report misses the output:\n%s", stdout.String())
	}
}

func TestCLIConvertErrorsArePositioned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.goml")
	if err := os.WriteFile(path, []byte("module m\nlet X := match y with | 0 => 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := CLIRun([]string{"convert", path}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "bad.goml:2:") {
		t.Fatalf("error not positioned: %s", stderr.String())
	}
}

func TestCLIVersionAndUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := CLIRun([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatal("version failed")
	}
	if !strings.HasPrefix(stdout.String(), "goml version v") {
		t.Fatalf("version output: %s", stdout.String())
	}
	stdout.Reset()
	if code := CLIRun(nil, &stdout, &stderr); code != 2 {
		t.Fatal("no-args should exit 2")
	}
	if !strings.Contains(stderr.String(), "goml gen") {
		t.Fatalf("usage missing: %s", stderr.String())
	}
}
