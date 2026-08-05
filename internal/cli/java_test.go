package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goforge.dev/goplus/internal/javatool"
)

func TestGenJavaUsesConfigAndIsCheckable(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26.0\n")
	writeCLIFile(t, filepath.Join(dir, "goplus.toml"), `schema_version = 1
default_targets = ["java"]

[targets.java]
release = 25
kind = "library"
source_dir = "build/java-src"
class_dir = ".goplus/classes"
jar = "dist/demo.jar"
runtime_jar = ".goplus/runtime.jar"
package_prefix = "com.example.demo"
module_name = "com.example.demo"
`)
	writeCLIFile(t, filepath.Join(dir, "demo.gp"), `package demo

func Sum(a, b int) int { return a + b }
`)
	withWorkingDirectory(t, dir, func() {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"gen", "."}, &stdout, &stderr); code != 0 {
			t.Fatalf("gen exit %d\nstdout:\n%s\nstderr:\n%s", code, &stdout, &stderr)
		}
		generated, err := os.ReadFile(filepath.Join(dir, "build", "java-src", "com", "example", "demo", "GpPackage.java"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(generated), "public static long Sum") {
			t.Fatalf("generated:\n%s", generated)
		}
		stdout.Reset()
		stderr.Reset()
		if code := Run([]string{"gen", "-check", "."}, &stdout, &stderr); code != 0 {
			t.Fatalf("check exit %d: %s", code, &stderr)
		}
	})
}

func TestGenTargetOverridesConfiguredDefault(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26.0\n")
	writeCLIFile(t, filepath.Join(dir, "goplus.toml"), "schema_version = 1\ndefault_targets = [\"java\"]\n")
	writeCLIFile(t, filepath.Join(dir, "demo.gp"), "package demo\nfunc Value() int { return 1 }\n")
	withWorkingDirectory(t, dir, func() {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"gen", "--target", "go", "."}, &stdout, &stderr); code != 0 {
			t.Fatalf("exit %d: %s", code, &stderr)
		}
		if _, err := os.Stat(filepath.Join(dir, "demo_gp.go")); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "gen", "java")); !os.IsNotExist(err) {
			t.Fatalf("Java output unexpectedly exists: %v", err)
		}
	})
}

func TestGenAcceptsRepeatableTargets(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "go.mod"), "module example.com/both\n\ngo 1.26.0\n")
	writeCLIFile(t, filepath.Join(dir, "demo.gp"), "package both\nfunc Value() int { return 1 }\n")
	withWorkingDirectory(t, dir, func() {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"gen", "--target", "go", "--target", "java", "."}, &stdout, &stderr); code != 0 {
			t.Fatalf("exit %d: %s", code, &stderr)
		}
		for _, path := range []string{
			filepath.Join(dir, "demo_gp.go"),
			filepath.Join(dir, "gen", "java", "com", "example", "both", "GpPackage.java"),
		} {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing generated target %s: %v", path, err)
			}
		}
	})
}

func TestGoCheckRetainsLegacyStaleDiagnostic(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "go.mod"), "module example.com/stale\n\ngo 1.26.0\n")
	writeCLIFile(t, filepath.Join(dir, "stale.gp"), "package stale\nfunc Value() int { return 1 }\n")
	withWorkingDirectory(t, dir, func() {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"gen", "-check", "."}, &stdout, &stderr); code != 1 {
			t.Fatalf("exit %d, stderr: %s", code, &stderr)
		}
		if !strings.Contains(stderr.String(), "stale generated code") {
			t.Fatalf("stderr: %s", &stderr)
		}
	})
}

func TestJavaBuildRunAndTestCommands(t *testing.T) {
	if _, err := javatool.Resolve(context.Background(), 25); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "go.mod"), "module example.com/app\n\ngo 1.26.0\n")
	writeCLIFile(t, filepath.Join(dir, "goplus.toml"), `schema_version = 1
default_targets = ["java"]

[targets.java]
release = 25
kind = "app"
source_dir = "gen/java"
class_dir = ".goplus/classes"
jar = "dist/app.jar"
runtime_jar = ".goplus/runtime.jar"
package_prefix = "com.example.app"
module_name = "com.example.app"
main_package = "example.com/app"
`)
	writeCLIFile(t, filepath.Join(dir, "main.gp"), "package main\nfunc main() { println(\"cli-java\") }\n")
	writeCLIFile(t, filepath.Join(dir, "main_test.gp"), `package main
import "testing"
func TestJavaCLI(t *testing.T) { t.Log("ok") }
`)
	withWorkingDirectory(t, dir, func() {
		for _, command := range [][]string{{"build"}, {"run"}, {"test"}} {
			var stdout, stderr bytes.Buffer
			if code := Run(command, &stdout, &stderr); code != 0 {
				t.Fatalf("%v exit %d\nstdout:\n%s\nstderr:\n%s", command, code, &stdout, &stderr)
			}
			switch command[0] {
			case "build":
				if !strings.Contains(stdout.String(), "dist/app.jar") {
					t.Fatalf("build stdout: %s", &stdout)
				}
			case "run":
				if stdout.String() != "cli-java\n" {
					t.Fatalf("run stdout: %q", stdout.String())
				}
			case "test":
				if !strings.Contains(stdout.String(), "--- PASS: TestJavaCLI") {
					t.Fatalf("test stdout: %s", &stdout)
				}
			}
		}
	})
}

func withWorkingDirectory(t *testing.T, dir string, body func()) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	body()
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
