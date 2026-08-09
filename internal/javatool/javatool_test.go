package javatool

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"
)

func TestBuildRejectsSymbolicLinkOutputAncestor(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".goplus")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, err := Build(context.Background(), Toolchain{}, Config{
		Root: root, Release: 25, SourceDir: "src", RuntimeSourceDir: "runtime",
		ClassDir: ".goplus/classes", Jar: "dist/app.jar", RuntimeJar: ".goplus/runtime.jar",
	}, os.Stdout, os.Stderr)
	if err == nil {
		t.Fatal("symbolic-link output ancestor accepted")
	}
	data, readErr := os.ReadFile(sentinel)
	if readErr != nil || string(data) != "keep" {
		t.Fatalf("outside directory changed: data=%q err=%v", data, readErr)
	}
}

func TestCreateJarIsDeterministicAndNamesModule(t *testing.T) {
	root := t.TempDir()
	classes := filepath.Join(root, "classes")
	if err := os.MkdirAll(filepath.Join(classes, "com", "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classes, "com", "example", "Main.class"), []byte("class"), 0o644); err != nil {
		t.Fatal(err)
	}
	one, two := filepath.Join(root, "one.jar"), filepath.Join(root, "two.jar")
	if err := createJar(one, "com.example.demo", "com.example.Main", []jarTree{{Root: classes}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := createJar(two, "com.example.demo", "com.example.Main", []jarTree{{Root: classes}}, nil); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(one)
	b, _ := os.ReadFile(two)
	if !bytes.Equal(a, b) {
		t.Fatal("JAR bytes are not deterministic")
	}
	zr, err := zip.OpenReader(one)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	found := false
	for _, file := range zr.File {
		if file.Name == "META-INF/MANIFEST.MF" {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data := new(bytes.Buffer)
			_, _ = data.ReadFrom(reader)
			_ = reader.Close()
			if !bytes.Contains(data.Bytes(), []byte("Automatic-Module-Name: com.example.demo")) {
				t.Fatalf("manifest: %s", data.Bytes())
			}
			found = true
		}
	}
	if !found {
		t.Fatal("manifest missing")
	}
}

func TestInsideRejectsEscape(t *testing.T) {
	if _, err := inside(t.TempDir(), "../classes"); err == nil {
		t.Fatal("escape accepted")
	}
}

func TestCleanJDKEnvironmentDropsAmbientInjection(t *testing.T) {
	got := cleanJDKEnvironment([]string{
		"PATH=/bin", "CLASSPATH=/tmp/ambient", "JAVA_TOOL_OPTIONS=-cp bad",
		"_JAVA_OPTIONS=-Dbad=true", "JDK_JAVA_OPTIONS=--module-path=bad", "KEEP=value",
	})
	want := []string{"PATH=/bin", "KEEP=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("environment = %v, want %v", got, want)
	}
}

func TestCreateJarDeterministicProperty(t *testing.T) {
	law := func(classBytes []byte) bool {
		root := t.TempDir()
		classes := filepath.Join(root, "classes")
		if err := os.MkdirAll(classes, 0o755); err != nil {
			return false
		}
		if err := os.WriteFile(filepath.Join(classes, "Value.class"), classBytes, 0o644); err != nil {
			return false
		}
		one, two := filepath.Join(root, "one.jar"), filepath.Join(root, "two.jar")
		if createJar(one, "example.module", "", []jarTree{{Root: classes}}, nil) != nil ||
			createJar(two, "example.module", "", []jarTree{{Root: classes}}, nil) != nil {
			return false
		}
		left, leftErr := os.ReadFile(one)
		right, rightErr := os.ReadFile(two)
		return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
	}
	if err := quick.Check(law, &quick.Config{MaxCount: 40}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBuildManifestIsDeterministicAndRelative(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "gen", "Demo.java")
	output := filepath.Join(root, "dist", "demo.jar")
	for path, data := range map[string]string{input: "class Demo {}\n", output: "jar"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	one := filepath.Join(root, ".goplus", "one.json")
	two := filepath.Join(root, ".goplus", "two.json")
	tool := Toolchain{Major: 25, Javac: "/absolute/not-recorded/javac"}
	if err := writeBuildManifest(root, one, tool, []string{input}, []string{output}); err != nil {
		t.Fatal(err)
	}
	if err := writeBuildManifest(root, two, tool, []string{input}, []string{output}); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(one)
	b, _ := os.ReadFile(two)
	if !bytes.Equal(a, b) {
		t.Fatal("manifest is not deterministic")
	}
	if bytes.Contains(a, []byte(root)) || !bytes.Contains(a, []byte(`"schema": "goplus.java.build/v2"`)) {
		t.Fatalf("manifest contains an absolute path or lacks schema: %s", a)
	}
}
