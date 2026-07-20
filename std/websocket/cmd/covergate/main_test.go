package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRequiredCoverage(t *testing.T) {
	path := writeTestFile(t, "coverage.feature", "Then handwritten statement coverage is 100 percent\n")
	want, err := requiredCoverage(path)
	if err != nil || want != 100 {
		t.Fatalf("requiredCoverage() = %v, %v", want, err)
	}

	missing := writeTestFile(t, "missing.feature", "Feature: no contract\n")
	if _, err := requiredCoverage(missing); err == nil {
		t.Fatal("requiredCoverage accepted a missing contract")
	}
}

func TestCoverageExcludesGeneratedFiles(t *testing.T) {
	profile := writeTestFile(t, "coverage.out", "mode: atomic\nexample.go:1.1,2.1 2 1\nexample_gp.go:1.1,2.1 9 0\nexample.go:3.1,4.1 3 0\n")
	covered, total, err := coverage(profile)
	if err != nil || covered != 2 || total != 5 {
		t.Fatalf("coverage() = %d/%d, %v", covered, total, err)
	}
}

func TestCoverageRejectsMalformedProfiles(t *testing.T) {
	for name, body := range map[string]string{
		"header": "not-a-header\n",
		"fields": "mode: set\nbad line\n",
		"range":  "mode: set\nmissing-colon 1 1\n",
		"empty":  "mode: set\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := coverage(writeTestFile(t, name, body)); err == nil {
				t.Fatal("coverage accepted a malformed profile")
			}
		})
	}
}
