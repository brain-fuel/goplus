package javaclass

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexSelectsJava25MultiReleaseClassAndChecksDescriptor(t *testing.T) {
	jar := filepath.Join(t.TempDir(), "library.jar")
	writeClassArchive(t, jar, map[string][]byte{
		"com/example/Example.class":                      testClassfile(69, "base", "()V"),
		"META-INF/versions/25/com/example/Example.class": testClassfile(69, "selected", "(I)I"),
		"META-INF/versions/26/com/example/Example.class": testClassfile(70, "tooNew", "()V"),
	})

	index, err := Open("", []string{jar}, 25)
	if err != nil {
		t.Fatal(err)
	}
	class, err := index.Lookup("com.example.Example")
	if err != nil {
		t.Fatal(err)
	}
	if !class.HasMethod("selected", "(I)I") {
		t.Fatalf("methods = %v", class.Methods)
	}
	if class.HasMethod("base", "()V") || class.HasMethod("tooNew", "()V") {
		t.Fatalf("wrong multi-release class selected: %v", class.Methods)
	}
	if _, err := index.Lookup("com.example.Missing"); err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("missing lookup error = %v", err)
	}
}

func TestIndexRejectsClassfilesNewerThanJava25(t *testing.T) {
	jar := filepath.Join(t.TempDir(), "future.jar")
	writeClassArchive(t, jar, map[string][]byte{
		"com/example/Future.class": testClassfile(70, "future", "()V"),
	})
	index, err := Open("", []string{jar}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.Lookup("com.example.Future"); err == nil || !strings.Contains(err.Error(), "newer than Java 25") {
		t.Fatalf("lookup error = %v", err)
	}
}

func TestIndexAcceptsClassfileForConfiguredRelease(t *testing.T) {
	jar := filepath.Join(t.TempDir(), "java26.jar")
	writeClassArchive(t, jar, map[string][]byte{
		"com/example/Future.class": testClassfile(70, "future", "()V"),
	})
	index, err := Open("", []string{jar}, 26)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.Lookup("com.example.Future"); err != nil {
		t.Fatalf("Java 26 class rejected for release 26: %v", err)
	}
}

func TestIndexReadsJavacReleaseSignaturesWithoutJmods(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeClassArchive(t, filepath.Join(home, "lib", "ct.sym"), map[string][]byte{
		"OP/java.base/java/lang/Example.sig": testClassfile(69, "api", "()V"),
		"O/java.base/java/lang/Old.sig":      testClassfile(69, "old", "()V"),
	})
	index, err := Open(home, nil, 25)
	if err != nil {
		t.Fatal(err)
	}
	class, err := index.Lookup("java.lang.Example")
	if err != nil {
		t.Fatal(err)
	}
	if !class.HasMethod("api", "()V") {
		t.Fatalf("methods = %v", class.Methods)
	}
	if _, err := index.Lookup("java.lang.Old"); err == nil {
		t.Fatal("release-24-only signature was indexed for release 25")
	}
}

func TestHasMethodFollowsPublicSupertypeMethods(t *testing.T) {
	jar := filepath.Join(t.TempDir(), "inheritance.jar")
	writeClassArchive(t, jar, map[string][]byte{
		"com/example/Base.class":  testClassfileFor(69, "com/example/Base", "java/lang/Object", "inherited", "()I"),
		"com/example/Child.class": testClassfileFor(69, "com/example/Child", "com/example/Base", "child", "()V"),
	})
	index, err := Open("", []string{jar}, 25)
	if err != nil {
		t.Fatal(err)
	}
	child, err := index.Lookup("com.example.Child")
	if err != nil {
		t.Fatal(err)
	}
	if !child.HasMethod("inherited", "()I") || child.HasMethod("<init>", "()V") {
		t.Fatalf("inherited lookup failed: %+v", child)
	}
}

func writeClassArchive(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, data := range entries {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func testClassfile(major uint16, method, descriptor string) []byte {
	return testClassfileFor(major, "example/Example", "java/lang/Object", method, descriptor)
}

func testClassfileFor(major uint16, className, superName, method, descriptor string) []byte {
	var out bytes.Buffer
	writeU4 := func(value uint32) { _ = binary.Write(&out, binary.BigEndian, value) }
	writeU2 := func(value uint16) { _ = binary.Write(&out, binary.BigEndian, value) }
	writeUTF8 := func(value string) {
		out.WriteByte(1)
		writeU2(uint16(len(value)))
		out.WriteString(value)
	}

	writeU4(0xcafebabe)
	writeU2(0)
	writeU2(major)
	writeU2(7)
	writeUTF8(method)     // #1
	writeUTF8(descriptor) // #2
	writeUTF8(className)  // #3
	out.WriteByte(7)      // #4 Class
	writeU2(3)
	writeUTF8(superName) // #5
	out.WriteByte(7)     // #6 Class
	writeU2(5)
	writeU2(0x0021) // public, super
	writeU2(4)      // this_class
	writeU2(6)      // super_class
	writeU2(0)      // interfaces
	writeU2(0)      // fields
	writeU2(1)      // methods
	writeU2(0x0001) // public
	writeU2(1)      // name
	writeU2(2)      // descriptor
	writeU2(0)      // method attributes
	writeU2(0)      // class attributes
	return out.Bytes()
}
