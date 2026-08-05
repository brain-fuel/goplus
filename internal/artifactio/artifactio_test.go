package artifactio

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"

	"goforge.dev/goplus/compiler"
)

func TestSyncWriteCheckAndOwnedOrphan(t *testing.T) {
	root := t.TempDir()
	set := compiler.ArtifactSet{Artifacts: []compiler.Artifact{
		{Path: "gen/A.java", Role: compiler.ArtifactSource, Data: []byte("a\n")},
		{Path: "gen/goplus-artifacts.json", Role: compiler.ArtifactManifest, Data: []byte("{\"artifacts\":[\"gen/A.java\"]}\n")},
	}}
	first, err := Sync(root, set, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Written) != 2 {
		t.Fatalf("written = %v", first.Written)
	}
	clean, err := Sync(root, set, Options{Check: true})
	if err != nil || len(clean.Stale) != 0 {
		t.Fatalf("clean = %+v, err=%v", clean, err)
	}
	set.Artifacts = []compiler.Artifact{{
		Path: "gen/goplus-artifacts.json", Role: compiler.ArtifactManifest,
		Data: []byte("{\"artifacts\":[]}\n"),
	}}
	checked, err := Sync(root, set, Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(checked.Orphans, []string{"gen/A.java"}) {
		t.Fatalf("orphans = %v", checked.Orphans)
	}
	if _, err := os.Stat(filepath.Join(root, "gen", "A.java")); err != nil {
		t.Fatal(err)
	}
}

func TestSyncRejectsEscapingArtifact(t *testing.T) {
	_, err := Sync(t.TempDir(), compiler.ArtifactSet{Artifacts: []compiler.Artifact{{Path: "../bad"}}}, Options{})
	if err == nil {
		t.Fatal("escaping artifact accepted")
	}
}

func TestSyncRejectsDuplicateArtifactPath(t *testing.T) {
	_, err := Sync(t.TempDir(), compiler.ArtifactSet{Artifacts: []compiler.Artifact{
		{Path: "gen/A.java", Data: []byte("one")},
		{Path: "gen/A.java", Data: []byte("two")},
	}}, Options{})
	if err == nil {
		t.Fatal("duplicate artifact path accepted")
	}
}

func TestSyncRejectsSymbolicLinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "gen")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, err := Sync(root, compiler.ArtifactSet{Artifacts: []compiler.Artifact{{Path: "gen/A.java", Data: []byte("bad")}}}, Options{})
	if err == nil {
		t.Fatal("symbolic-link escape accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "A.java")); !os.IsNotExist(err) {
		t.Fatalf("outside path was written: %v", err)
	}
}

func TestCheckCanIgnoreEphemeralRuntimeSources(t *testing.T) {
	root := t.TempDir()
	set := compiler.ArtifactSet{Artifacts: []compiler.Artifact{
		{Path: "gen/A.java", Role: compiler.ArtifactSource, Data: []byte("source\n")},
		{Path: ".goplus/runtime/GpRuntime.java", Role: compiler.ArtifactRuntime, Data: []byte("runtime\n")},
		{Path: "gen/goplus-artifacts.json", Role: compiler.ArtifactManifest, Data: []byte("{\"artifacts\":[\"gen/A.java\",\".goplus/runtime/GpRuntime.java\"]}\n")},
	}}
	if _, err := Sync(root, set, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".goplus", "runtime", "GpRuntime.java")); err != nil {
		t.Fatal(err)
	}
	result, err := Sync(root, set, Options{Check: true, IgnoreRuntimeInCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stale) != 0 {
		t.Fatalf("stale = %v", result.Stale)
	}
}

func TestSyncThenCheckIsCleanProperty(t *testing.T) {
	law := func(data []byte) bool {
		root := t.TempDir()
		set := compiler.ArtifactSet{Artifacts: []compiler.Artifact{
			{Path: "gen/A.java", Role: compiler.ArtifactSource, Data: append([]byte(nil), data...)},
			{Path: "gen/goplus-artifacts.json", Role: compiler.ArtifactManifest, Data: []byte("{\"artifacts\":[\"gen/A.java\"]}\n")},
		}}
		if _, err := Sync(root, set, Options{}); err != nil {
			return false
		}
		result, err := Sync(root, set, Options{Check: true})
		return err == nil && len(result.Stale) == 0 && len(result.Orphans) == 0
	}
	if err := quick.Check(law, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatal(err)
	}
}
