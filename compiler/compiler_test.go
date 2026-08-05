package compiler

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompileGoAndJavaFromOneShadow(t *testing.T) {
	dir := t.TempDir()
	writeCompilerFile(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26.0\n")
	writeCompilerFile(t, filepath.Join(dir, "demo.gp"), `package demo

type Pair struct { Left int; Text string }

func Add(a, b int) int { return a + b }
func Greeting() string { return "hello" }
`)
	result, err := Compile(context.Background(), Request{
		Dir: dir, Patterns: []string{"."}, Targets: []Target{TargetGo, TargetJava},
		Java: JavaOptions{Release: 25, Kind: "library", SourceDir: "gen/java"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if len(result.ArtifactSets) != 2 {
		t.Fatalf("artifact sets = %d", len(result.ArtifactSets))
	}
	goSource := artifactContaining(t, result.ArtifactSets[0], "demo_gp.go")
	if !strings.Contains(goSource, "func Add") {
		t.Fatalf("Go source:\n%s", goSource)
	}
	javaSource := artifactContaining(t, result.ArtifactSets[1], "GpPackage.java")
	if !strings.Contains(javaSource, "public static long Add(long a, long b)") ||
		!strings.Contains(javaSource, "GpString.fromBase64(\"aGVsbG8=\")") {
		t.Fatalf("Java source:\n%s", javaSource)
	}
	structSource := artifactContaining(t, result.ArtifactSets[1], "Pair.java")
	if !strings.Contains(structSource, "public final class Pair") || !strings.Contains(structSource, "GpCopy<Pair>") {
		t.Fatalf("Pair source:\n%s", structSource)
	}
}

func TestCompileRejectsUnsupportedAddressTaking(t *testing.T) {
	dir := t.TempDir()
	writeCompilerFile(t, filepath.Join(dir, "go.mod"), "module example.com/demo\n\ngo 1.26.0\n")
	writeCompilerFile(t, filepath.Join(dir, "demo.go"), "package demo\nfunc Pointer() *int { value := 1; return &value }\n")
	result, err := Compile(context.Background(), Request{
		Dir: dir, Patterns: []string{"."}, Targets: []Target{TargetJava},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok() || len(result.Diagnostics) == 0 || !strings.Contains(result.Diagnostics[0].Message, "address") {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestJavaArtifactsAreIndependentOfCheckoutPath(t *testing.T) {
	compile := func(dir string) ArtifactSet {
		t.Helper()
		writeCompilerFile(t, filepath.Join(dir, "go.mod"), "module example.com/deterministic\n\ngo 1.26.0\n")
		writeCompilerFile(t, filepath.Join(dir, "value.gp"), "package deterministic\nfunc Value() int { return 42 }\n")
		result, err := Compile(context.Background(), Request{
			Dir: dir, Patterns: []string{"."}, Targets: []Target{TargetJava},
			Java: JavaOptions{Release: 25, Kind: "library", SourceDir: "gen/java"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Ok() {
			t.Fatalf("diagnostics: %+v", result.Diagnostics)
		}
		return result.ArtifactSets[0]
	}

	one := compile(t.TempDir())
	two := compile(t.TempDir())
	if !reflect.DeepEqual(one, two) {
		t.Fatalf("Java artifact sets depend on checkout path\none:  %#v\ntwo:  %#v", one, two)
	}
}

func TestJavaCompilationIncludesModuleLocalDependencies(t *testing.T) {
	dir := t.TempDir()
	writeCompilerFile(t, filepath.Join(dir, "go.mod"), "module example.com/dependencies\n\ngo 1.26.0\n")
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeCompilerFile(t, filepath.Join(dir, "lib", "lib.go"), "package lib\nfunc Value() int { return 7 }\n")
	writeCompilerFile(t, filepath.Join(dir, "cmd", "app", "main.go"), `package main
import "example.com/dependencies/lib"
func main() { println(lib.Value()) }
`)
	result, err := Compile(context.Background(), Request{
		Dir: dir, Patterns: []string{"./cmd/app"}, Targets: []Target{TargetJava},
		Java: JavaOptions{
			Release: 25, Kind: "app", SourceDir: "gen/java",
			PackagePrefix: "com.example.dependencies", MainPackage: "example.com/dependencies/cmd/app",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	set := result.ArtifactSets[0]
	lib := artifactContaining(t, set, "/lib/GpPackage.java")
	if !strings.Contains(lib, "public static long Value()") {
		t.Fatalf("library source:\n%s", lib)
	}
	app := artifactContaining(t, set, "/cmd/app/GpPackage.java")
	if !strings.Contains(app, "com.example.dependencies.lib.GpPackage.Value()") {
		t.Fatalf("app source:\n%s", app)
	}
}

func TestTargetsElaborateWithIndependentBuildTags(t *testing.T) {
	dir := t.TempDir()
	writeCompilerFile(t, filepath.Join(dir, "go.mod"), "module example.com/tagged\n\ngo 1.26.0\n")
	writeCompilerFile(t, filepath.Join(dir, "platform_go.gp"), `//go:build !goplus_java

package tagged
func Platform() int { return 1 }
`)
	writeCompilerFile(t, filepath.Join(dir, "platform_java.gp"), `//go:build goplus_java

package tagged
func Platform() int { return 2 }
`)
	result, err := Compile(context.Background(), Request{
		Dir: dir, Patterns: []string{"."}, Targets: []Target{TargetGo, TargetJava},
		Java: JavaOptions{Release: 25, SourceDir: "gen/java"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok() {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	java := artifactContaining(t, result.ArtifactSets[1], "GpPackage.java")
	if !strings.Contains(java, "return 2L;") || strings.Contains(java, "return 1L;") {
		t.Fatalf("Java target selected the wrong build-tagged source:\n%s", java)
	}
}

func artifactContaining(t *testing.T, set ArtifactSet, suffix string) string {
	t.Helper()
	for _, artifact := range set.Artifacts {
		if strings.HasSuffix(artifact.Path, suffix) {
			return string(artifact.Data)
		}
	}
	t.Fatalf("no artifact ending in %q: %+v", suffix, set.Artifacts)
	return ""
}

func writeCompilerFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
