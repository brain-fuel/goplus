package javaffi

import (
	"strings"
	"testing"
)

func TestPreparePackageImportAndConstructor(t *testing.T) {
	source := []byte(`//go:build goplus_java

package demo

import util "java:package/java.util"

func value() {
	list := new(util.ArrayList[string], 4)
	_ = list
}
`)
	got, changed, err := prepareFile("demo.gp", source)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("Java import was not prepared")
	}
	text := string(got)
	for _, want := range []string{
		"__goplusJavaNew[__goplusJava_util_ArrayList[string]](4)",
		"//goplus:java-type __goplusJava_util_ArrayList java.util.ArrayList",
		"type __goplusJava_util_ArrayList[J0 any] struct{}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "java:package") {
		t.Fatalf("java import remained:\n%s", text)
	}
}

func TestPrepareDirectGenericType(t *testing.T) {
	source := []byte(`package demo
import List "java:type/java.util.List"
func use(value List[string]) { _ = value }
`)
	got, changed, err := prepareFile("demo.gp", source)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("Java import was not prepared")
	}
	if !strings.Contains(string(got), "type List[J0 any] struct{}") {
		t.Fatalf("prepared:\n%s", got)
	}
}

func TestPrepareLeavesRichGoPlusWithoutJavaImportsToFrontEnd(t *testing.T) {
	source := []byte("package demo\n\ntype Result[T any] enum { Ok(Value T); Failed(Message string) }\n")
	got, changed, err := prepareFile("demo.gp", source)
	if err != nil {
		t.Fatal(err)
	}
	if changed || string(got) != string(source) {
		t.Fatalf("non-FFI Go+ source was rewritten: changed=%v\n%s", changed, got)
	}
}
