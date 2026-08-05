package projectconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadDefaultsAtModuleBoundary(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module example.com/demo\n")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(sub)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Root != root || !reflect.DeepEqual(cfg.DefaultTargets, []string{"go"}) {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Java.Release != 25 || cfg.Java.SourceDir != "gen/java" {
		t.Fatalf("unexpected Java defaults: %+v", cfg.Java)
	}
}

func TestParseJavaTarget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, FileName)
	write(t, path, `schema_version = 1
default_targets = ["go", "java"]

[targets.go]

[targets.java]
release = 25
kind = "app"
source_dir = "build/generated/java"
class_dir = ".goplus/classes"
jar = "dist/demo.jar"
runtime_jar = ".goplus/runtime.jar"
package_prefix = "com.example.demo"
module_name = "com.example.demo"
main_package = "example.com/demo/cmd/demo"
classpath_files = ["deps/classpath.txt"]
modulepath_files = ["deps/modulepath.txt"]
bundle = true
strong_module = true
`)
	cfg, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.DefaultTargets, []string{"go", "java"}) {
		t.Fatalf("targets = %v", cfg.DefaultTargets)
	}
	if cfg.Java.Kind != "app" || cfg.Java.PackagePrefix != "com.example.demo" || !cfg.Java.Bundle || !cfg.Java.StrongModule {
		t.Fatalf("Java config = %+v", cfg.Java)
	}
}

func TestParseRejectsUnknownAndUnsafeOutput(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want string
	}{
		"unknown":         {"schema_version = 1\n[targets.java]\njra = \"x\"\n", "unknown targets.java key"},
		"escape":          {"schema_version = 1\n[targets.java]\njar = \"../x.jar\"\n", "must stay within"},
		"old":             {"schema_version = 1\n[targets.java]\nrelease = 24\n", "at least 25"},
		"same jars":       {"schema_version = 1\n[targets.java]\njar = \"same.jar\"\nruntime_jar = \"same.jar\"\n", "must be different"},
		"jar in classes":  {"schema_version = 1\n[targets.java]\njar = \".goplus/build/java/classes/project.jar\"\n", "jar must not be inside class_dir"},
		"duplicate key":   {"schema_version = 1\nschema_version = 1\n", "declared twice"},
		"duplicate table": {"schema_version = 1\n[targets.java]\n[targets.java]\n", "declared twice"},
		"non toml bool":   {"schema_version = 1\n[targets.java]\nbundle = TRUE\n", "expected boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), FileName)
			write(t, path, tc.body)
			_, err := Parse(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
