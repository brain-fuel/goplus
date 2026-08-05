package javatool

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goforge.dev/goplus/compiler"
	"goforge.dev/goplus/internal/artifactio"
)

func TestJava25CompileAndRun(t *testing.T) {
	ctx := context.Background()
	tool, err := Resolve(ctx, 25)
	if err != nil {
		t.Skip(err)
	}
	root := t.TempDir()
	writeJavaToolFile(t, filepath.Join(root, "go.mod"), "module example.com/demo\n\ngo 1.26.0\n")
	writeJavaToolFile(t, filepath.Join(root, "main.gp"), `package main

import util "java:package/java.util"

type Pair struct { Left int; Right int }
type FloatPair struct { Value float64 }
type LocalError struct{}

func sum(pair Pair) int { return pair.Left + pair.Right }
func add(left, right int) int { return left + right }
func (LocalError) Error() string { return "local error" }
func localError() error { return LocalError{} }
func (pair Pair) setLeft(value int) { pair.Left = value }
func (pair *Pair) setRight(value int) { pair.Right = value }
func (pair *Pair) printRight() { println("method-defer", pair.Right) }
func printDeferred() { println("deferred") }
func deferred() { defer printDeferred(); println("body") }
func deferredReceiver() {
	pair := &Pair{Right: 1}
	defer pair.printRight()
	pair = &Pair{Right: 2}
}

func main() {
	pair := Pair{Left: 2, Right: 3}
	pair.setLeft(40)
	pair.setRight(5)
	items := make([]int, 2, 3)
	items[0] = 1
	alias := items
	alias[0] = 4
	grown := append(items, 9)
	ch := make(chan int)
	go func() { ch <- 7; close(ch) }()
	println("hello", sum(pair), len("é"), len(items), len(grown), items[0], <-ch)
	buffered := make(chan int, 1)
	buffered <- 11
	select {
	case selected := <-buffered:
		println("select", selected)
	default:
		println("wrong")
	}
	checked := make(chan int, 1)
	checked <- 13
	close(checked)
	select {
	case selected, ok := <-checked:
		println("select-ok", selected, ok)
	}
	select {
	case <-buffered:
		println("wrong")
	default:
		println("default")
	}
	closed := make(chan int, 2)
	closed <- 1
	closed <- 2
	close(closed)
	total := 0
	for value := range closed { total += value }
	println("range", total)
	zeros := make([]int, 1)
	counts := make(map[Pair]int)
	counts[Pair{Left: 8, Right: 9}] = 12
	println("zero", zeros[0], counts[Pair{}], counts[Pair{Left: 8, Right: 9}])
	zeros[0] = 4
	clear(zeros)
clear(counts)
	println("clear", zeros[0], len(counts))
	var nilSlice []int
	var nilMap map[string]int
	var nilChan chan int
	println("nil", nilSlice == nil, nilMap == nil, nilChan == nil)
	array := [3]int{1}
	arrayCopy := array
	arrayCopy[0] = 9
	view := array[:]
	view[1] = 4
	spare := make([]int, 0, 2)
	expanded := spare[:2]
	nested := [1][1]int{{2}}
	nestedCopy := nested
	nestedCopy[0][0] = 8
	println("array", array[0], arrayCopy[0], array[1], array[2], expanded[0], nested[0][0], nestedCopy[0][0])
	overlap := []int{1, 2, 3}
	overlap = append(overlap[:1], overlap...)
	println("append-overlap", overlap[0], overlap[1], overlap[2], overlap[3])
	key := Pair{Left: 1, Right: 2}
	snapshot := map[Pair]int{key: 3}
	key.Left = 9
	println("map-key", snapshot[Pair{Left: 1, Right: 2}])
	lookup, present := snapshot[Pair{Left: 1, Right: 2}]
	missing, absent := snapshot[Pair{}]
	println("lookup", lookup, present, missing, absent)
	pointers := make(chan *Pair, 1)
	pointers <- nil
	println("chan-nil", <-pointers == nil)
	done := make(chan int)
	close(done)
	closedValue, open := <-done
	println("receive-ok", closedValue, open)
	switch 2 {
	default:
		println("wrong-switch")
	case 2:
		println("switch", 2)
	}
	rangeArray := [2]int{1, 2}
	rangeSum := 0
	for index, value := range rangeArray {
		if index == 0 { rangeArray[1] = 9 }
		rangeSum += value
	}
	println("range-array", rangeSum, rangeArray[1])
	deferred()
	deferredReceiver()
	rawBytes := []byte{255, 97}
	rawString := string(rawBytes)
	bytesAgain := []byte(rawString)
	runeString := string([]rune{'世'})
	runesAgain := []rune(runeString)
	badRune := string(rune(0xd800))
	println("conversions", len(rawString), bytesAgain[0], bytesAgain[1], len(runeString), runesAgain[0], len(badRune))
	var signed8 int8 = 127
	signed8++
	signed8 += 2
	unsigned8 := uint8(255)
	quotient8 := unsigned8 / 2
	unsigned8++
	unsigned8--
	unsigned16 := uint16(65535)
	unsigned16++
	unsigned32 := uint32(4294967295)
	unsigned32++
	println("widths", signed8, unsigned8, quotient8, unsigned16, unsigned32)
	arrayMap := make(map[[2]int]int)
	arrayMap[[2]int{1, 2}] = 14
	zeroFloat := 0.0
	floatMap := make(map[float64]int)
	floatMap[zeroFloat] = 15
	nan := zeroFloat / zeroFloat
	floatMap[nan] = 16
	_, nanPresent := floatMap[nan]
	floatPair := FloatPair{Value: nan}
	nanArray := [1]float64{nan}
	pointerA := &Pair{Left: 1}
	pointerB := &Pair{Left: 1}
	pointerAlias := pointerA
	pointerArray := [1]*Pair{pointerA}
	pointerArrayCopy := pointerArray
	pointerSlice := []*Pair{pointerA}
	pointerSliceCopy := make([]*Pair, 1)
	copy(pointerSliceCopy, pointerSlice)
	high := uint64(1) << 63
	bits := uint8(0)
	fn := add
	println("keys", arrayMap[[2]int{1, 2}], floatMap[-zeroFloat], nanPresent, len(floatMap))
	println("identity", floatPair == floatPair, nanArray == nanArray, pointerA == pointerB, pointerA == pointerAlias, pointerArrayCopy[0] == pointerA, pointerSliceCopy[0] == pointerA)
	println("minmax", min(high, uint64(3)) == 3, max(high, uint64(3)) == high, min("z", "a"), max("z", "a"), ^bits)
	println("unsigned", ^uint64(0))
	println("function", fn(2, 3), localError())
	println("ffi", JavaMax(3, 8), JavaConcat("go", "+"))
	parsed, parseErr := JavaParseInt("27")
	_, badErr := JavaParseInt("bad")
	println("ffi-result", parsed, parseErr == nil, badErr != nil, badErr.Error())
	println("ffi-constructor", JavaBuilder(4) != nil)
	list := new(util.ArrayList[string], 4)
	println("direct", list)
}
`)
	writeJavaToolFile(t, filepath.Join(root, "ffi.gp"), `//go:build goplus_java

package main

import StringBuilder "java:type/java.lang.StringBuilder"

//goplus:java kind=static owner=java.lang.Math member=max descriptor="(II)I" null=nonnull throws=none
func JavaMax(a, b int32) int32

//goplus:java kind=virtual owner=java.lang.String member=concat descriptor="(Ljava/lang/String;)Ljava/lang/String;" null=nonnull throws=none string=java
func JavaConcat(receiver, value string) string

//goplus:java kind=static owner=java.lang.Integer member=parseInt descriptor="(Ljava/lang/String;)I" null=nonnull throws=result string=java
func JavaParseInt(value string) (int32, error)

//goplus:java kind=constructor owner=java.lang.StringBuilder descriptor="(I)V" null=nonnull throws=none
func JavaBuilder(capacity int32) *StringBuilder
`)
	writeJavaToolFile(t, filepath.Join(root, "main_test.gp"), `package main

import "testing"

func TestSum(t *testing.T) {
	if sum(Pair{Left: 4, Right: 5}) != 9 { t.Fatalf("wrong sum") }
	t.Run("nested", func(t *testing.T) { t.Log("ok") })
}
`)
	compiled, err := compiler.Compile(ctx, compiler.Request{
		Dir: root, Patterns: []string{"."}, Targets: []compiler.Target{compiler.TargetJava},
		Java: compiler.JavaOptions{
			Release: 25, Kind: "app", SourceDir: "gen/java",
			RuntimeSourceDir: ".goplus/build/java/runtime-src",
			PackagePrefix:    "com.example.demo", ModuleName: "com.example.demo",
			MainPackage: "example.com/demo", IncludeTests: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.Ok() {
		t.Fatalf("diagnostics: %+v", compiled.Diagnostics)
	}
	set := compiled.ArtifactSets[0]
	if _, err := artifactio.Sync(root, set, artifactio.Options{}); err != nil {
		t.Fatal(err)
	}
	var javacOutput bytes.Buffer
	built, err := Build(ctx, tool, Config{
		Root: root, Release: 25, SourceDir: "gen/java",
		RuntimeSourceDir: ".goplus/build/java/runtime-src",
		ClassDir:         ".goplus/build/java/classes", Jar: "dist/demo.jar",
		RuntimeJar: ".goplus/build/java/runtime.jar", ModuleName: set.ModuleName,
		MainClass: set.MainClass,
	}, &javacOutput, &javacOutput)
	if err != nil {
		t.Fatalf("build: %v\n%s", err, &javacOutput)
	}
	projectJar, err := os.ReadFile(built.Jar)
	if err != nil {
		t.Fatal(err)
	}
	runtimeJar, err := os.ReadFile(built.RuntimeJar)
	if err != nil {
		t.Fatal(err)
	}
	javacOutput.Reset()
	rebuilt, err := Build(ctx, tool, Config{
		Root: root, Release: 25, SourceDir: "gen/java",
		RuntimeSourceDir: ".goplus/build/java/runtime-src",
		ClassDir:         ".goplus/build/java/classes", Jar: "dist/demo.jar",
		RuntimeJar: ".goplus/build/java/runtime.jar", ModuleName: set.ModuleName,
		MainClass: set.MainClass,
	}, &javacOutput, &javacOutput)
	if err != nil {
		t.Fatalf("rebuild: %v\n%s", err, &javacOutput)
	}
	rebuiltProject, _ := os.ReadFile(rebuilt.Jar)
	rebuiltRuntime, _ := os.ReadFile(rebuilt.RuntimeJar)
	if !bytes.Equal(projectJar, rebuiltProject) || !bytes.Equal(runtimeJar, rebuiltRuntime) {
		t.Fatal("repeated Java build produced different JAR bytes")
	}
	built = rebuilt
	var stdout, stderr bytes.Buffer
	if err := Run(ctx, tool, built, nil, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, &stderr)
	}
	if stdout.String() != "hello 7 2 2 3 4 7\nselect 11\nselect-ok 13 true\ndefault\nrange 3\nzero 0 0 12\nclear 0 0\nnil true true true\narray 1 9 4 0 0 2 8\nappend-overlap 1 1 2 3\nmap-key 3\nlookup 3 true 0 false\nchan-nil true\nreceive-ok 0 false\nswitch 2\nrange-array 3 9\nbody\ndeferred\nmethod-defer 1\nconversions 2 255 97 3 19990 3\nwidths -126 255 127 0 0\nkeys 14 15 false 2\nidentity false false false true true true\nminmax true true a z 255\nunsigned 18446744073709551615\nfunction 5 local error\nffi 8 go+\nffi-result 27 true true For input string: \"bad\"\nffi-constructor true\ndirect []\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if len(set.TestClasses) != 1 {
		t.Fatalf("test classes = %v", set.TestClasses)
	}
	testBuild := built
	testBuild.MainClass = set.TestClasses[0]
	stdout.Reset()
	stderr.Reset()
	if err := Run(ctx, tool, testBuild, nil, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("test run: %v\nstdout: %s\nstderr: %s", err, &stdout, &stderr)
	}
	if !strings.Contains(stdout.String(), "--- PASS: TestSum") || !strings.Contains(stdout.String(), "--- PASS: TestSum/nested") {
		t.Fatalf("test stdout: %s\nstderr: %s", &stdout, &stderr)
	}
}

func TestJava25StrongModule(t *testing.T) {
	ctx := context.Background()
	tool, err := Resolve(ctx, 25)
	if err != nil {
		t.Skip(err)
	}
	for _, bundle := range []bool{false, true} {
		name := "thin"
		if bundle {
			name = "bundled"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeJavaToolFile(t, filepath.Join(root, "go.mod"), "module example.com/strong\n\ngo 1.26.0\n")
			writeJavaToolFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() { println(\"strong\") }\n")
			compiled, err := compiler.Compile(ctx, compiler.Request{
				Dir: root, Patterns: []string{"."}, Targets: []compiler.Target{compiler.TargetJava},
				Java: compiler.JavaOptions{
					Release: 25, Kind: "app", SourceDir: "gen/java",
					RuntimeSourceDir: ".goplus/build/java/runtime-src",
					PackagePrefix:    "com.example.strong", ModuleName: "com.example.strong",
					MainPackage: "example.com/strong", StrongModule: true, Bundle: bundle,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if !compiled.Ok() {
				t.Fatalf("diagnostics: %+v", compiled.Diagnostics)
			}
			set := compiled.ArtifactSets[0]
			if _, err := artifactio.Sync(root, set, artifactio.Options{}); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			built, err := Build(ctx, tool, Config{
				Root: root, Release: 25, SourceDir: "gen/java",
				RuntimeSourceDir: ".goplus/build/java/runtime-src",
				ClassDir:         ".goplus/build/java/classes", Jar: "dist/strong.jar",
				RuntimeJar: ".goplus/build/java/runtime.jar", ModuleName: set.ModuleName,
				MainClass: set.MainClass, StrongModule: true, Bundle: bundle,
			}, &output, &output)
			if err != nil {
				t.Fatalf("build: %v\n%s", err, &output)
			}
			var stdout, stderr bytes.Buffer
			if err := Run(ctx, tool, built, nil, strings.NewReader(""), &stdout, &stderr); err != nil {
				t.Fatalf("run: %v\nstderr: %s", err, &stderr)
			}
			if stdout.String() != "strong\n" {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func writeJavaToolFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
