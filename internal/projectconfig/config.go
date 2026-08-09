// Package projectconfig loads the target configuration for a Go+ module.
//
// The format is deliberately small and versioned. It accepts only the TOML
// constructs used by goplus.toml so misspelled keys fail loudly instead of
// silently changing a build.
package projectconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const FileName = "goplus.toml"

// Config is the complete module configuration.
type Config struct {
	Root           string
	Path           string
	SchemaVersion  int
	DefaultTargets []string
	Java           Java
}

// Java configures the Java 25 source and artifact backend.
type Java struct {
	Release         int
	Kind            string
	SourceDir       string
	ClassDir        string
	Jar             string
	SourcesJar      string
	JavadocJar      string
	BuildManifest   string
	RuntimeJar      string
	PackagePrefix   string
	ModuleName      string
	MainPackage     string
	ClasspathFiles  []string
	ModulepathFiles []string
	Bundle          bool
	StrongModule    bool
}

// Defaults returns the no-file configuration. Go remains the default target.
func Defaults(root string) Config {
	return Config{
		Root:           root,
		SchemaVersion:  2,
		DefaultTargets: []string{"go"},
		Java: Java{
			Release:       25,
			Kind:          "library",
			SourceDir:     "gen/java",
			ClassDir:      ".goplus/build/java/classes",
			Jar:           "dist/project.jar",
			SourcesJar:    "dist/project-sources.jar",
			JavadocJar:    "dist/project-javadoc.jar",
			BuildManifest: ".goplus/build/java/publication.json",
			RuntimeJar:    ".goplus/build/java/goplus-runtime-abi1.jar",
		},
	}
}

// Load searches from start toward the filesystem root. Search stops at the
// nearest go.mod boundary; a missing file is not an error and returns Defaults.
func Load(start string) (Config, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return Config{}, err
	}
	if info, statErr := os.Stat(abs); statErr == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return Parse(path)
		} else if !os.IsNotExist(err) {
			return Config{}, err
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return Defaults(dir), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Defaults(abs), nil
		}
	}
}

// Parse parses a goplus.toml file.
func Parse(path string) (Config, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Config{}, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	cfg := Defaults(filepath.Dir(abs))
	cfg.Path = abs
	section := ""
	seenSchema := false
	seenTables := map[string]bool{}
	seenKeys := map[string]bool{}
	s := bufio.NewScanner(f)
	for lineNo := 1; s.Scan(); lineNo++ {
		line, err := stripComment(s.Text())
		if err != nil {
			return Config{}, at(abs, lineNo, err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") || strings.HasPrefix(line, "[[") {
				return Config{}, at(abs, lineNo, fmt.Errorf("invalid table header"))
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			switch section {
			case "targets.go", "targets.java":
			default:
				return Config{}, at(abs, lineNo, fmt.Errorf("unknown table %q", section))
			}
			if seenTables[section] {
				return Config{}, at(abs, lineNo, fmt.Errorf("table %q is declared twice", section))
			}
			seenTables[section] = true
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, at(abs, lineNo, fmt.Errorf("expected key = value"))
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		if key == "" || raw == "" {
			return Config{}, at(abs, lineNo, fmt.Errorf("expected key = value"))
		}
		qualifiedKey := section + "." + key
		if seenKeys[qualifiedKey] {
			return Config{}, at(abs, lineNo, fmt.Errorf("key %q is declared twice", strings.TrimPrefix(qualifiedKey, ".")))
		}
		seenKeys[qualifiedKey] = true
		switch section {
		case "":
			switch key {
			case "schema_version":
				n, err := parseInt(raw)
				if err != nil {
					return Config{}, at(abs, lineNo, err)
				}
				cfg.SchemaVersion, seenSchema = n, true
			case "default_targets":
				v, err := parseStrings(raw)
				if err != nil {
					return Config{}, at(abs, lineNo, err)
				}
				cfg.DefaultTargets = v
			default:
				return Config{}, at(abs, lineNo, fmt.Errorf("unknown root key %q", key))
			}
		case "targets.go":
			return Config{}, at(abs, lineNo, fmt.Errorf("targets.go has no options; unknown key %q", key))
		case "targets.java":
			if err := setJava(&cfg.Java, key, raw); err != nil {
				return Config{}, at(abs, lineNo, err)
			}
		}
	}
	if err := s.Err(); err != nil {
		return Config{}, err
	}
	if !seenSchema {
		return Config{}, fmt.Errorf("%s: schema_version is required", abs)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", abs, err)
	}
	return cfg, nil
}

func setJava(j *Java, key, raw string) error {
	stringValue := func() (string, error) { return parseString(raw) }
	stringsValue := func() ([]string, error) { return parseStrings(raw) }
	switch key {
	case "release":
		v, err := parseInt(raw)
		j.Release = v
		return err
	case "kind":
		v, err := stringValue()
		j.Kind = v
		return err
	case "source_dir":
		v, err := stringValue()
		j.SourceDir = v
		return err
	case "class_dir":
		v, err := stringValue()
		j.ClassDir = v
		return err
	case "jar":
		v, err := stringValue()
		j.Jar = v
		return err
	case "sources_jar":
		v, err := stringValue()
		j.SourcesJar = v
		return err
	case "javadoc_jar":
		v, err := stringValue()
		j.JavadocJar = v
		return err
	case "build_manifest":
		v, err := stringValue()
		j.BuildManifest = v
		return err
	case "runtime_jar":
		v, err := stringValue()
		j.RuntimeJar = v
		return err
	case "package_prefix":
		v, err := stringValue()
		j.PackagePrefix = v
		return err
	case "module_name":
		v, err := stringValue()
		j.ModuleName = v
		return err
	case "main_package":
		v, err := stringValue()
		j.MainPackage = v
		return err
	case "classpath_files":
		v, err := stringsValue()
		j.ClasspathFiles = v
		return err
	case "modulepath_files":
		v, err := stringsValue()
		j.ModulepathFiles = v
		return err
	case "bundle":
		v, err := parseBool(raw)
		j.Bundle = v
		return err
	case "strong_module":
		v, err := parseBool(raw)
		j.StrongModule = v
		return err
	default:
		return fmt.Errorf("unknown targets.java key %q", key)
	}
}

func (c Config) validate() error {
	if c.SchemaVersion != 2 {
		return fmt.Errorf("unsupported schema_version %d (want 2)", c.SchemaVersion)
	}
	if len(c.DefaultTargets) == 0 {
		return fmt.Errorf("default_targets must not be empty")
	}
	seen := map[string]bool{}
	for _, target := range c.DefaultTargets {
		if target != "go" && target != "java" {
			return fmt.Errorf("unknown default target %q", target)
		}
		if seen[target] {
			return fmt.Errorf("default target %q is listed twice", target)
		}
		seen[target] = true
	}
	if c.Java.Release < 25 {
		return fmt.Errorf("targets.java.release must be at least 25")
	}
	if c.Java.Kind != "library" && c.Java.Kind != "app" {
		return fmt.Errorf("targets.java.kind must be %q or %q", "library", "app")
	}
	for _, item := range []struct{ name, value string }{
		{"package_prefix", c.Java.PackagePrefix},
		{"module_name", c.Java.ModuleName},
	} {
		if item.value != "" && !validJavaNamespace(item.value) {
			return fmt.Errorf("targets.java.%s is not a valid Java qualified name", item.name)
		}
	}
	if filepath.Clean(c.Java.Jar) == filepath.Clean(c.Java.RuntimeJar) {
		return fmt.Errorf("targets.java.jar and runtime_jar must be different paths")
	}
	for _, item := range []struct{ name, value string }{
		{"source_dir", c.Java.SourceDir},
		{"class_dir", c.Java.ClassDir},
		{"jar", c.Java.Jar},
		{"sources_jar", c.Java.SourcesJar},
		{"javadoc_jar", c.Java.JavadocJar},
		{"build_manifest", c.Java.BuildManifest},
		{"runtime_jar", c.Java.RuntimeJar},
	} {
		if item.value == "" {
			return fmt.Errorf("targets.java.%s must not be empty", item.name)
		}
		if filepath.IsAbs(item.value) {
			return fmt.Errorf("targets.java.%s must be relative to the module root", item.name)
		}
		clean := filepath.Clean(item.value)
		if clean == "." {
			return fmt.Errorf("targets.java.%s must name a path below the module root", item.name)
		}
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("targets.java.%s must stay within the module root", item.name)
		}
	}
	if relativePathsOverlap(c.Java.SourceDir, c.Java.ClassDir) ||
		relativePathsOverlap(c.Java.SourceDir, ".goplus/build/java/runtime-src") ||
		relativePathsOverlap(c.Java.ClassDir, ".goplus/build/java/runtime-src") {
		return fmt.Errorf("targets.java source and build directories must not overlap")
	}
	for _, archive := range []struct{ name, path string }{
		{"jar", c.Java.Jar}, {"runtime_jar", c.Java.RuntimeJar},
	} {
		for _, directory := range []struct{ name, path string }{
			{"source_dir", c.Java.SourceDir}, {"class_dir", c.Java.ClassDir},
			{"runtime source directory", ".goplus/build/java/runtime-src"},
		} {
			if relativePathInside(archive.path, directory.path) {
				return fmt.Errorf("targets.java.%s must not be inside %s", archive.name, directory.name)
			}
		}
	}
	return nil
}

func validJavaNamespace(value string) bool {
	keywords := map[string]bool{
		"_": true, "abstract": true, "assert": true, "boolean": true,
		"break": true, "byte": true, "case": true, "catch": true,
		"char": true, "class": true, "const": true, "continue": true,
		"default": true, "do": true, "double": true, "else": true,
		"enum": true, "exports": true, "extends": true, "false": true,
		"final": true, "finally": true, "float": true, "for": true,
		"goto": true, "if": true, "implements": true, "import": true,
		"instanceof": true, "int": true, "interface": true, "long": true,
		"module": true, "native": true, "new": true, "non-sealed": true,
		"null": true, "open": true, "opens": true, "package": true,
		"permits": true, "private": true, "protected": true, "provides": true,
		"public": true, "record": true, "requires": true, "return": true,
		"sealed": true, "short": true, "static": true, "strictfp": true,
		"super": true, "switch": true, "synchronized": true, "this": true,
		"throw": true, "throws": true, "to": true, "transient": true,
		"transitive": true, "true": true, "try": true, "uses": true,
		"var": true, "void": true, "volatile": true, "while": true,
		"with": true, "yield": true,
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" || keywords[part] {
			return false
		}
		for index, r := range part {
			if !(r == '_' || r == '$' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || index > 0 && r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

func relativePathsOverlap(left, right string) bool {
	left = filepath.Clean(filepath.FromSlash(left))
	right = filepath.Clean(filepath.FromSlash(right))
	return left == right || strings.HasPrefix(left, right+string(filepath.Separator)) || strings.HasPrefix(right, left+string(filepath.Separator))
}

func relativePathInside(path, directory string) bool {
	path = filepath.Clean(filepath.FromSlash(path))
	directory = filepath.Clean(filepath.FromSlash(directory))
	return path == directory || strings.HasPrefix(path, directory+string(filepath.Separator))
}

func stripComment(line string) (string, error) {
	inString, escaped := false, false
	for i, r := range line {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
			} else if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '#':
			return line[:i], nil
		}
	}
	if inString {
		return "", fmt.Errorf("unterminated string")
	}
	return line, nil
}

func parseInt(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", raw)
	}
	return n, nil
}

func parseString(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", fmt.Errorf("expected quoted string, got %q", raw)
	}
	v, err := strconv.Unquote(raw)
	if err != nil {
		return "", fmt.Errorf("expected quoted string, got %q", raw)
	}
	return v, nil
}

func parseBool(raw string) (bool, error) {
	switch strings.TrimSpace(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected boolean, got %q", raw)
	}
}

func parseStrings(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return nil, fmt.Errorf("expected string array, got %q", raw)
	}
	raw = strings.TrimSpace(raw[1 : len(raw)-1])
	if raw == "" {
		return []string{}, nil
	}
	var parts []string
	start, inString, escaped := 0, false, false
	for i, r := range raw {
		if inString {
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == '"' {
				inString = false
			}
			continue
		}
		if r == '"' {
			inString = true
		} else if r == ',' {
			parts = append(parts, raw[start:i])
			start = i + 1
		}
	}
	if inString {
		return nil, fmt.Errorf("unterminated string array")
	}
	parts = append(parts, raw[start:])
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v, err := parseString(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func at(path string, line int, err error) error {
	return fmt.Errorf("%s:%d: %w", path, line, err)
}
