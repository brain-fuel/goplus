// Package mavencentral creates and uploads Maven Central Portal bundles.
package mavencentral

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// Metadata is the required Maven POM identity and provenance surface.
type Metadata struct {
	GroupID, ArtifactID, Version                             string
	Name, Description, URL                                   string
	LicenseName, LicenseURL                                  string
	DeveloperID, DeveloperName, DeveloperEmail, DeveloperURL string
	SCMURL, SCMConnection, SCMDeveloperConnection            string
}

// BundleOptions describes already-built Java inputs.
type BundleOptions struct {
	Root, Jar, SourceDir, RuntimeSourceDir, Output string
	Metadata                                       Metadata
	Signer                                         Signer
}

// BundleResult is a Central-ready archive and its Maven repository coordinate.
type BundleResult struct {
	Path, RepositoryPath string
	Files                []string
}

// Signer creates an ASCII-armored detached OpenPGP signature.
type Signer interface {
	Sign(context.Context, string) ([]byte, error)
}

type signatureVerifier interface {
	Verify(data, signature []byte) error
}

// GPGSigner signs through a user's configured gpg keyring or agent.
type GPGSigner struct{ Key string }

func (s GPGSigner) Sign(ctx context.Context, path string) ([]byte, error) {
	gpg, err := exec.LookPath("gpg")
	if err != nil {
		return nil, fmt.Errorf("OpenPGP signing requires gpg on PATH: %w", err)
	}
	args := []string{"--batch", "--armor", "--detach-sign", "--output", "-"}
	if strings.TrimSpace(s.Key) != "" {
		args = append(args, "--local-user", s.Key)
	}
	args = append(args, path)
	cmd := exec.CommandContext(ctx, gpg, args...)
	cmd.Stdin = nil
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg signing %s: %w: %s", filepath.Base(path), err, strings.TrimSpace(stderr.String()))
	}
	if !bytes.Contains(stdout.Bytes(), []byte("BEGIN PGP SIGNATURE")) {
		return nil, fmt.Errorf("gpg returned a non-armored signature for %s", filepath.Base(path))
	}
	return stdout.Bytes(), nil
}

// BuildBundle creates the Maven repository layout, required artifacts,
// signatures, checksums, and the final deterministic ZIP container.
func BuildBundle(ctx context.Context, o BundleOptions) (BundleResult, error) {
	if err := validate(o); err != nil {
		return BundleResult{}, err
	}
	root, err := filepath.Abs(o.Root)
	if err != nil {
		return BundleResult{}, err
	}
	jar := resolve(root, o.Jar)
	sourceDir := resolve(root, o.SourceDir)
	runtimeDir := resolve(root, o.RuntimeSourceDir)
	output := resolve(root, o.Output)
	jarData, err := os.ReadFile(jar)
	if err != nil {
		return BundleResult{}, fmt.Errorf("reading project JAR: %w", err)
	}
	base := o.Metadata.ArtifactID + "-" + o.Metadata.Version
	repo := strings.ReplaceAll(o.Metadata.GroupID, ".", "/") + "/" + o.Metadata.ArtifactID + "/" + o.Metadata.Version
	files := map[string][]byte{base + ".jar": jarData, base + ".pom": pom(o.Metadata)}
	files[base+"-sources.jar"], err = sourceJar(sourceDir, runtimeDir)
	if err != nil {
		return BundleResult{}, err
	}
	files[base+"-javadoc.jar"], err = documentationJar(o.Metadata)
	if err != nil {
		return BundleResult{}, err
	}

	work, err := os.MkdirTemp("", "goplus-central-sign-")
	if err != nil {
		return BundleResult{}, err
	}
	defer os.RemoveAll(work)
	var primary []string
	for name := range files {
		primary = append(primary, name)
	}
	sort.Strings(primary)
	for _, name := range primary {
		path := filepath.Join(work, name)
		if err := os.WriteFile(path, files[name], 0o600); err != nil {
			return BundleResult{}, err
		}
		sig, err := o.Signer.Sign(ctx, path)
		if err != nil {
			return BundleResult{}, err
		}
		files[name+".asc"] = sig
		if verifier, ok := o.Signer.(signatureVerifier); ok {
			if err := verifier.Verify(files[name], sig); err != nil {
				return BundleResult{}, fmt.Errorf("verifying %s signature: %w", name, err)
			}
		}
		m := md5.Sum(files[name])
		files[name+".md5"] = []byte(hex.EncodeToString(m[:]))
		s := sha1.Sum(files[name])
		files[name+".sha1"] = []byte(hex.EncodeToString(s[:]))
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return BundleResult{}, err
	}
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	if err := writeZip(output, func(w *zip.Writer) error {
		for _, name := range names {
			if err := addZip(w, repo+"/"+name, files[name]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return BundleResult{}, err
	}
	return BundleResult{Path: output, RepositoryPath: repo, Files: names}, nil
}

func validate(o BundleOptions) error {
	if o.Signer == nil {
		return fmt.Errorf("Maven Central bundle requires an OpenPGP signer")
	}
	for name, value := range map[string]string{
		"root": o.Root, "jar": o.Jar, "source directory": o.SourceDir, "output": o.Output,
		"groupId": o.Metadata.GroupID, "artifactId": o.Metadata.ArtifactID, "version": o.Metadata.Version,
		"name": o.Metadata.Name, "description": o.Metadata.Description, "url": o.Metadata.URL,
		"license name": o.Metadata.LicenseName, "license url": o.Metadata.LicenseURL,
		"developer id": o.Metadata.DeveloperID, "developer name": o.Metadata.DeveloperName,
		"developer email": o.Metadata.DeveloperEmail, "scm url": o.Metadata.SCMURL,
		"scm connection": o.Metadata.SCMConnection, "scm developer connection": o.Metadata.SCMDeveloperConnection,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("Maven Central %s must not be empty", name)
		}
	}
	return nil
}

type project struct {
	XMLName      xml.Name    `xml:"project"`
	XMLNS        string      `xml:"xmlns,attr"`
	XMLNSXSI     string      `xml:"xmlns:xsi,attr"`
	Schema       string      `xml:"xsi:schemaLocation,attr"`
	ModelVersion string      `xml:"modelVersion"`
	GroupID      string      `xml:"groupId"`
	ArtifactID   string      `xml:"artifactId"`
	Version      string      `xml:"version"`
	Packaging    string      `xml:"packaging"`
	Name         string      `xml:"name"`
	Description  string      `xml:"description"`
	URL          string      `xml:"url"`
	Licenses     []license   `xml:"licenses>license"`
	Developers   []developer `xml:"developers>developer"`
	SCM          scm         `xml:"scm"`
}
type license struct {
	Name         string `xml:"name"`
	URL          string `xml:"url"`
	Distribution string `xml:"distribution"`
}
type developer struct {
	ID    string `xml:"id"`
	Name  string `xml:"name"`
	Email string `xml:"email"`
	URL   string `xml:"url,omitempty"`
}
type scm struct {
	Connection          string `xml:"connection"`
	DeveloperConnection string `xml:"developerConnection"`
	URL                 string `xml:"url"`
}

func pom(m Metadata) []byte {
	p := project{XMLNS: "http://maven.apache.org/POM/4.0.0", XMLNSXSI: "http://www.w3.org/2001/XMLSchema-instance", Schema: "http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd", ModelVersion: "4.0.0", GroupID: m.GroupID, ArtifactID: m.ArtifactID, Version: m.Version, Packaging: "jar", Name: m.Name, Description: m.Description, URL: m.URL,
		Licenses: []license{{m.LicenseName, m.LicenseURL, "repo"}}, Developers: []developer{{m.DeveloperID, m.DeveloperName, m.DeveloperEmail, m.DeveloperURL}}, SCM: scm{m.SCMConnection, m.SCMDeveloperConnection, m.SCMURL}}
	b, _ := xml.MarshalIndent(p, "", "  ")
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), append(b, '\n')...)
}

func sourceJar(dirs ...string) ([]byte, error) {
	entries := map[string][]byte{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, e os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".java") {
				return nil
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(rel)
			if _, ok := entries[name]; ok {
				return fmt.Errorf("duplicate Java source %s", name)
			}
			entries[name] = data
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no generated Java sources for sources JAR")
	}
	return zipBytes(entries)
}

func documentationJar(m Metadata) ([]byte, error) {
	text := "# " + m.Name + "\n\n" + m.Description + "\n\nGenerated Java API source and project documentation: " + m.URL + "\n"
	return zipBytes(map[string][]byte{"README.md": []byte(text)})
}
func zipBytes(entries map[string][]byte) ([]byte, error) {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	var n []string
	for k := range entries {
		n = append(n, k)
	}
	sort.Strings(n)
	for _, k := range n {
		if err := addZip(w, k, entries[k]); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func writeZip(path string, fill func(*zip.Writer) error) error {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	if err := fill(w); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}
func addZip(w *zip.Writer, name string, data []byte) error {
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetModTime(zipEpoch)
	h.SetMode(0o644)
	f, err := w.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}
func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

// InspectBundle verifies layout, signatures/checksums presence, and checksum
// content without requiring network access or a secret key.
func InspectBundle(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	files := map[string][]byte{}
	for _, f := range r.File {
		rc, e := f.Open()
		if e != nil {
			return e
		}
		b, e := io.ReadAll(rc)
		rc.Close()
		if e != nil {
			return e
		}
		files[f.Name] = b
	}
	var primary int
	for name, data := range files {
		if strings.HasSuffix(name, ".asc") || strings.HasSuffix(name, ".md5") || strings.HasSuffix(name, ".sha1") {
			continue
		}
		primary++
		if len(files[name+".asc"]) == 0 {
			return fmt.Errorf("%s has no signature", name)
		}
		m := md5.Sum(data)
		if string(files[name+".md5"]) != hex.EncodeToString(m[:]) {
			return fmt.Errorf("%s has invalid md5", name)
		}
		s := sha1.Sum(data)
		if string(files[name+".sha1"]) != hex.EncodeToString(s[:]) {
			return fmt.Errorf("%s has invalid sha1", name)
		}
	}
	if primary != 4 {
		return fmt.Errorf("bundle has %d primary artifacts, want 4", primary)
	}
	return nil
}
