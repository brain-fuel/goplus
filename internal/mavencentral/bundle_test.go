package mavencentral

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSigner struct{}

func (fakeSigner) Sign(_ context.Context, path string) ([]byte, error) {
	return []byte("-----BEGIN PGP SIGNATURE-----\n" + filepath.Base(path) + "\n-----END PGP SIGNATURE-----\n"), nil
}

func TestBuildBundleHasCentralLayoutAndIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeTest(t, filepath.Join(root, "dist", "demo.jar"), "jar")
	writeTest(t, filepath.Join(root, "gen", "Demo.java"), "package dev.goforge.demo; public class Demo {}\n")
	writeTest(t, filepath.Join(root, "runtime", "Runtime.java"), "package dev.goforge.runtime; public class Runtime {}\n")
	metadata := Metadata{GroupID: "dev.goforge", ArtifactID: "demo", Version: "1.2.3", Name: "Demo", Description: "A demo", URL: "https://goforge.dev/demo/", LicenseName: "MIT License", LicenseURL: "https://opensource.org/license/mit", DeveloperID: "brain-fuel", DeveloperName: "brain-fuel", DeveloperEmail: "opensource@goforge.dev", DeveloperURL: "https://github.com/brain-fuel", SCMURL: "https://github.com/brain-fuel/demo", SCMConnection: "scm:git:https://github.com/brain-fuel/demo.git", SCMDeveloperConnection: "scm:git:ssh://git@github.com/brain-fuel/demo.git"}
	o := BundleOptions{Root: root, Jar: "dist/demo.jar", SourceDir: "gen", RuntimeSourceDir: "runtime", Output: "dist/bundle.zip", Metadata: metadata, Signer: fakeSigner{}}
	first, err := BuildBundle(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if err := InspectBundle(first.Path); err != nil {
		t.Fatal(err)
	}
	b1, err := os.ReadFile(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	o.Output = "dist/bundle2.zip"
	second, err := BuildBundle(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := os.ReadFile(second.Path)
	if string(b1) != string(b2) {
		t.Fatal("bundle is not deterministic with a deterministic signer")
	}
	r, err := zip.OpenReader(first.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	seen := map[string]bool{}
	for _, f := range r.File {
		seen[f.Name] = true
	}
	base := "dev/goforge/demo/1.2.3/demo-1.2.3"
	for _, suffix := range []string{".jar", ".pom", "-sources.jar", "-javadoc.jar"} {
		for _, extra := range []string{"", ".asc", ".md5", ".sha1"} {
			if !seen[base+suffix+extra] {
				t.Fatalf("missing %s", base+suffix+extra)
			}
		}
	}
}

func TestInspectBundleRejectsMissingSignature(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	if err := writeZip(path, func(w *zip.Writer) error { return addZip(w, "dev/goforge/x/1/x-1.jar", []byte("x")) }); err != nil {
		t.Fatal(err)
	}
	if err := InspectBundle(path); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("error = %v", err)
	}
}
func writeTest(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPOMEscapesMetadata(t *testing.T) {
	b := pom(Metadata{GroupID: "dev.goforge", ArtifactID: "x", Version: "1", Name: "A & B", Description: "<safe>", URL: "https://example.com"})
	s := string(b)
	if !strings.Contains(s, "A &amp; B") || !strings.Contains(s, "&lt;safe&gt;") {
		t.Fatal(s)
	}
}
