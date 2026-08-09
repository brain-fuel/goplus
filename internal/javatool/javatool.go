// Package javatool compiles Java 25 artifact sets without relying on Maven,
// Gradle, or the JDK jar command.
package javatool

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const RuntimeModuleName = "dev.goforge.goplus.runtime"

// Config describes a materialized Java artifact set.
type Config struct {
	Root             string
	Release          int
	SourceDir        string
	RuntimeSourceDir string
	ClassDir         string
	Jar              string
	SourcesJar       string
	JavadocJar       string
	BuildManifest    string
	RuntimeJar       string
	ModuleName       string
	MainClass        string
	ClasspathFiles   []string
	ModulepathFiles  []string
	StrongModule     bool
	Bundle           bool
}

// Toolchain is a verified JDK installation.
type Toolchain struct {
	Javac   string
	Java    string
	Javadoc string
	Major   int
}

// BuildResult names the deterministic jars and resolved runtime classpath.
type BuildResult struct {
	Jar           string
	SourcesJar    string
	JavadocJar    string
	BuildManifest string
	RuntimeJar    string
	MainClass     string
	ModuleName    string
	StrongModule  bool
	Classpath     []string
	Modulepath    []string
}

// Resolve finds and verifies a JDK new enough for release. GOPLUS_JAVA_HOME is
// authoritative, followed by JAVA_HOME, asdf Temurin installs, and PATH.
func Resolve(ctx context.Context, release int) (Toolchain, error) {
	if release < 25 {
		release = 25
	}
	var candidates []string
	if home := strings.TrimSpace(os.Getenv("GOPLUS_JAVA_HOME")); home != "" {
		// An explicit Go+ override is authoritative: silently falling back could
		// compile against a different JDK than the build requested.
		candidates = append(candidates, filepath.Join(home, "bin", executable("javac")))
	} else {
		if home := strings.TrimSpace(os.Getenv("JAVA_HOME")); home != "" {
			candidates = append(candidates, filepath.Join(home, "bin", executable("javac")))
		}
		if userHome, err := os.UserHomeDir(); err == nil {
			matches, _ := filepath.Glob(filepath.Join(userHome, ".asdf", "installs", "java", "temurin-*", "bin", executable("javac")))
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			candidates = append(candidates, matches...)
		}
		if path, err := exec.LookPath("javac"); err == nil {
			candidates = append(candidates, path)
		}
	}
	seen := map[string]bool{}
	var found []string
	for _, javac := range candidates {
		if javac == "" || seen[javac] {
			continue
		}
		seen[javac] = true
		major, err := javacMajor(ctx, javac)
		if err != nil {
			continue
		}
		found = append(found, fmt.Sprintf("%s (JDK %d)", javac, major))
		if major < release {
			continue
		}
		java := filepath.Join(filepath.Dir(javac), executable("java"))
		if _, err := os.Stat(java); err != nil {
			continue
		}
		javadoc := filepath.Join(filepath.Dir(javac), executable("javadoc"))
		if _, err := os.Stat(javadoc); err != nil {
			continue
		}
		return Toolchain{Javac: javac, Java: java, Javadoc: javadoc, Major: major}, nil
	}
	detail := "none found"
	if len(found) > 0 {
		detail = strings.Join(found, ", ")
	}
	return Toolchain{}, fmt.Errorf("Java target requires JDK %d+ (found %s); set GOPLUS_JAVA_HOME to a JDK installation", release, detail)
}

// Build compiles runtime and project sources with --release 25+ and creates
// deterministic jars internally.
func Build(ctx context.Context, tool Toolchain, cfg Config, stdout, stderr io.Writer) (BuildResult, error) {
	if cfg.Release < 25 {
		cfg.Release = 25
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return BuildResult{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return BuildResult{}, fmt.Errorf("resolving Java module root: %w", err)
	}
	sourceDir, err := inside(root, cfg.SourceDir)
	if err != nil {
		return BuildResult{}, err
	}
	runtimeSourceDir, err := inside(root, cfg.RuntimeSourceDir)
	if err != nil {
		return BuildResult{}, err
	}
	classDir, err := inside(root, cfg.ClassDir)
	if err != nil {
		return BuildResult{}, err
	}
	jarPath, err := inside(root, cfg.Jar)
	if err != nil {
		return BuildResult{}, err
	}
	sourcesJar, err := inside(root, cfg.SourcesJar)
	if err != nil {
		return BuildResult{}, err
	}
	javadocJar, err := inside(root, cfg.JavadocJar)
	if err != nil {
		return BuildResult{}, err
	}
	buildManifest, err := inside(root, cfg.BuildManifest)
	if err != nil {
		return BuildResult{}, err
	}
	runtimeJar, err := inside(root, cfg.RuntimeJar)
	if err != nil {
		return BuildResult{}, err
	}
	if jarPath == runtimeJar {
		return BuildResult{}, fmt.Errorf("project JAR and runtime JAR must be different paths")
	}
	runtimeClasses := filepath.Join(filepath.Dir(classDir), "runtime-classes")
	if directoriesOverlap(classDir, runtimeClasses) {
		return BuildResult{}, fmt.Errorf("project and runtime class directories must be different")
	}
	for _, sources := range []string{sourceDir, runtimeSourceDir} {
		if directoriesOverlap(classDir, sources) || directoriesOverlap(runtimeClasses, sources) {
			return BuildResult{}, fmt.Errorf("Java class output directory must not overlap source directory %s", sources)
		}
	}
	for _, archive := range []string{jarPath, runtimeJar} {
		for _, directory := range []string{sourceDir, runtimeSourceDir, classDir, runtimeClasses} {
			if pathInside(archive, directory) {
				return BuildResult{}, fmt.Errorf("Java JAR output %s must not be inside source or class directory %s", archive, directory)
			}
		}
	}
	for _, path := range []string{sourceDir, runtimeSourceDir, classDir, runtimeClasses, jarPath, runtimeJar} {
		if err := rejectSymlinkComponents(root, path); err != nil {
			return BuildResult{}, fmt.Errorf("unsafe Java build path: %w", err)
		}
	}
	if err := ensureOwnedBuildDir(root, classDir); err != nil {
		return BuildResult{}, err
	}
	if err := ensureOwnedBuildDir(root, runtimeClasses); err != nil {
		return BuildResult{}, err
	}
	if err := cleanDir(classDir); err != nil {
		return BuildResult{}, err
	}
	if err := cleanDir(runtimeClasses); err != nil {
		return BuildResult{}, err
	}

	runtimeSources, err := javaFiles(runtimeSourceDir)
	if err != nil {
		return BuildResult{}, err
	}
	projectSources, err := javaFiles(sourceDir)
	if err != nil {
		return BuildResult{}, err
	}
	if len(runtimeSources) == 0 {
		return BuildResult{}, fmt.Errorf("no Go+ Java runtime sources under %s; run `goplus gen --target java`", cfg.RuntimeSourceDir)
	}
	if len(projectSources) == 0 {
		return BuildResult{}, fmt.Errorf("no Java project sources under %s; run `goplus gen --target java`", cfg.SourceDir)
	}

	if err := javac(ctx, tool, cfg.Release, runtimeClasses, nil, nil, runtimeSources, stdout, stderr); err != nil {
		return BuildResult{}, fmt.Errorf("compiling Go+ Java runtime: %w", err)
	}
	if err := createJar(runtimeJar, RuntimeModuleName, "", []jarTree{{Root: runtimeClasses}}, nil); err != nil {
		return BuildResult{}, err
	}
	classpath, err := ReadPathFiles(root, cfg.ClasspathFiles)
	if err != nil {
		return BuildResult{}, err
	}
	modulepath, err := ReadPathFiles(root, cfg.ModulepathFiles)
	if err != nil {
		return BuildResult{}, err
	}
	compileClasspath := append([]string{}, classpath...)
	compileModulepath := append([]string{}, modulepath...)
	compileSources := projectSources
	if cfg.StrongModule && cfg.Bundle {
		compileSources = append(append([]string{}, projectSources...), withoutModuleInfo(runtimeSources)...)
	} else if cfg.StrongModule {
		compileModulepath = append([]string{runtimeJar}, compileModulepath...)
	} else {
		compileClasspath = append([]string{runtimeJar}, compileClasspath...)
	}
	if err := javac(ctx, tool, cfg.Release, classDir, compileClasspath, compileModulepath, compileSources, stdout, stderr); err != nil {
		return BuildResult{}, fmt.Errorf("compiling Go+ Java module: %w", err)
	}
	trees := []jarTree{{Root: classDir}}
	if cfg.Bundle && !cfg.StrongModule {
		trees = append(trees, jarTree{Root: runtimeClasses, Skip: map[string]bool{"module-info.class": true}})
	}
	if err := createJar(jarPath, cfg.ModuleName, cfg.MainClass, trees, nil); err != nil {
		return BuildResult{}, err
	}
	sourceTrees := []jarTree{{Root: sourceDir}}
	if cfg.Bundle {
		sourceTrees = append(sourceTrees, jarTree{Root: runtimeSourceDir, Skip: map[string]bool{"module-info.java": true}})
	}
	if err := createJar(sourcesJar, "", "", sourceTrees, nil); err != nil {
		return BuildResult{}, err
	}
	docDir := filepath.Join(filepath.Dir(classDir), "javadoc")
	if err := ensureOwnedBuildDir(root, docDir); err != nil {
		return BuildResult{}, err
	}
	if err := cleanDir(docDir); err != nil {
		return BuildResult{}, err
	}
	docSources := projectSources
	docClasspath := append([]string{}, compileClasspath...)
	if cfg.Bundle {
		docSources = append(append([]string{}, projectSources...), withoutModuleInfo(runtimeSources)...)
	}
	if err := runJavadoc(ctx, tool, cfg.Release, docDir, docClasspath, compileModulepath, docSources, stdout, stderr); err != nil {
		return BuildResult{}, fmt.Errorf("documenting Go+ Java module: %w", err)
	}
	if err := createJar(javadocJar, "", "", []jarTree{{Root: docDir}}, nil); err != nil {
		return BuildResult{}, err
	}
	if err := writeBuildManifest(root, buildManifest, tool, append(append([]string{}, projectSources...), runtimeSources...), []string{jarPath, sourcesJar, javadocJar}); err != nil {
		return BuildResult{}, err
	}
	resultRuntimeJar := runtimeJar
	if cfg.Bundle {
		resultRuntimeJar = ""
	}
	return BuildResult{
		Jar: jarPath, SourcesJar: sourcesJar, JavadocJar: javadocJar, BuildManifest: buildManifest,
		RuntimeJar: resultRuntimeJar, MainClass: cfg.MainClass,
		ModuleName: cfg.ModuleName, StrongModule: cfg.StrongModule,
		Classpath: classpath, Modulepath: modulepath,
	}, nil
}

func runJavadoc(ctx context.Context, tool Toolchain, release int, output string, classpath, modulepath, sources []string, stdout, stderr io.Writer) error {
	args := []string{"--release", strconv.Itoa(release), "-encoding", "UTF-8", "-notimestamp", "-Xdoclint:all,-missing", "-d", output}
	if len(classpath) > 0 {
		args = append(args, "--class-path", strings.Join(classpath, string(os.PathListSeparator)))
	}
	if len(modulepath) > 0 {
		args = append(args, "--module-path", strings.Join(modulepath, string(os.PathListSeparator)))
	}
	args = append(args, sources...)
	cmd := exec.CommandContext(ctx, tool.Javadoc, args...)
	cmd.Env, cmd.Stdout, cmd.Stderr = cleanJDKEnvironment(os.Environ()), stdout, stderr
	return cmd.Run()
}

type publicationManifest struct {
	Schema          string           `json:"schema"`
	InputTreeSHA256 string           `json:"input_tree_sha256"`
	JDK             string           `json:"jdk"`
	Outputs         []manifestOutput `json:"outputs"`
}
type manifestOutput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func writeBuildManifest(root, path string, tool Toolchain, inputs, outputs []string) error {
	sort.Strings(inputs)
	h := sha256.New()
	for _, input := range inputs {
		rel, err := filepath.Rel(root, input)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		_, _ = h.Write(data)
	}
	m := publicationManifest{Schema: "goplus.java.build/v2", InputTreeSHA256: hex.EncodeToString(h.Sum(nil)), JDK: fmt.Sprintf("jdk-%d", tool.Major)}
	for _, output := range outputs {
		data, err := os.ReadFile(output)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, output)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		m.Outputs = append(m.Outputs, manifestOutput{Path: filepath.ToSlash(rel), SHA256: hex.EncodeToString(sum[:])})
	}
	sort.Slice(m.Outputs, func(i, j int) bool { return m.Outputs[i].Path < m.Outputs[j].Path })
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func withoutModuleInfo(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if filepath.Base(path) != "module-info.java" {
			out = append(out, path)
		}
	}
	return out
}

// Run executes a built app with no ambient CLASSPATH or module path.
func Run(ctx context.Context, tool Toolchain, build BuildResult, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if build.MainClass == "" {
		return fmt.Errorf("Java target is a library; set targets.java.kind = %q and main_package to run it", "app")
	}
	var command []string
	if build.StrongModule {
		modulepath := append([]string{build.Jar, build.RuntimeJar}, build.Modulepath...)
		command = append(command, "--module-path", strings.Join(modulepath, string(os.PathListSeparator)))
		command = append(command, "--module", build.ModuleName+"/"+build.MainClass)
	} else {
		classpath := append([]string{build.Jar}, build.Classpath...)
		if build.RuntimeJar != "" {
			classpath = append(classpath, build.RuntimeJar)
		}
		command = append(command, "-cp", strings.Join(classpath, string(os.PathListSeparator)), build.MainClass)
	}
	command = append(command, args...)
	cmd := exec.CommandContext(ctx, tool.Java, command...)
	cmd.Env = cleanJDKEnvironment(os.Environ())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run()
}

func javac(ctx context.Context, tool Toolchain, release int, output string, classpath, modulepath, sources []string, stdout, stderr io.Writer) error {
	args := []string{"--release", strconv.Itoa(release), "-encoding", "UTF-8", "-proc:none", "-Xlint:all", "-Werror", "-d", output}
	if len(classpath) == 0 {
		// An explicit, owned empty directory prevents javac from consulting the
		// working directory or an ambient CLASSPATH.
		classpath = []string{output}
	}
	args = append(args, "--class-path", strings.Join(classpath, string(os.PathListSeparator)))
	if len(modulepath) > 0 {
		args = append(args, "--module-path", strings.Join(modulepath, string(os.PathListSeparator)))
	}
	args = append(args, sources...)
	cmd := exec.CommandContext(ctx, tool.Javac, args...)
	cmd.Env = cleanJDKEnvironment(os.Environ())
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

func javacMajor(ctx context.Context, path string) (int, error) {
	cmd := exec.CommandContext(ctx, path, "-version")
	cmd.Env = cleanJDKEnvironment(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected javac version %q", output)
	}
	version := strings.TrimSpace(fields[1])
	majorText := strings.SplitN(version, ".", 2)[0]
	major, err := strconv.Atoi(majorText)
	if err != nil {
		return 0, err
	}
	if major == 1 {
		parts := strings.Split(version, ".")
		if len(parts) > 1 {
			major, err = strconv.Atoi(parts[1])
		}
	}
	return major, err
}

func cleanJDKEnvironment(values []string) []string {
	blocked := map[string]bool{
		"CLASSPATH": true, "JAVA_TOOL_OPTIONS": true,
		"_JAVA_OPTIONS": true, "JDK_JAVA_OPTIONS": true,
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		name, _, _ := strings.Cut(value, "=")
		if !blocked[strings.ToUpper(name)] {
			out = append(out, value)
		}
	}
	return out
}

// ReadPathFiles resolves newline-delimited classpath or module-path manifests.
// Relative manifest paths and entries are anchored at root. Blank lines and
// trailing # comments are ignored, and every referenced entry must exist.
func ReadPathFiles(root string, listFiles []string) ([]string, error) {
	var result []string
	seen := map[string]bool{}
	for _, listFile := range listFiles {
		path := listFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(path))
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading Java path file %s: %w", path, err)
		}
		for lineNo, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
			if line == "" {
				continue
			}
			entry := filepath.FromSlash(line)
			if !filepath.IsAbs(entry) {
				entry = filepath.Join(root, entry)
			}
			entry = filepath.Clean(entry)
			if _, err := os.Stat(entry); err != nil {
				return nil, fmt.Errorf("%s:%d: Java path %s: %w", path, lineNo+1, entry, err)
			}
			if !seen[entry] {
				seen[entry] = true
				result = append(result, entry)
			}
		}
	}
	return result, nil
}

func javaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".java") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	sort.Strings(files)
	return files, err
}

type jarTree struct {
	Root string
	Skip map[string]bool
}

func createJar(path, moduleName, mainClass string, trees []jarTree, extra map[string][]byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	entries := map[string][]byte{}
	for _, tree := range trees {
		err := filepath.WalkDir(tree.Root, func(file string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(tree.Root, file)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(rel)
			if tree.Skip[name] {
				return nil
			}
			if _, duplicate := entries[name]; duplicate {
				return fmt.Errorf("duplicate bundled JAR entry %s", name)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			entries[name] = data
			return nil
		})
		if err != nil {
			return err
		}
	}
	for name, data := range extra {
		if _, duplicate := entries[name]; duplicate {
			return fmt.Errorf("duplicate JAR entry %s", name)
		}
		entries[name] = data
	}
	entries["META-INF/MANIFEST.MF"] = manifest(moduleName, mainClass)
	var names []string
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	stamp := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(stamp)
		header.SetMode(0o644)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := writer.Write(entries[name]); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o644)
}

func manifest(moduleName, mainClass string) []byte {
	var b strings.Builder
	b.WriteString("Manifest-Version: 1.0\r\n")
	if moduleName != "" {
		b.WriteString("Automatic-Module-Name: " + moduleName + "\r\n")
	}
	if mainClass != "" {
		b.WriteString("Main-Class: " + mainClass + "\r\n")
	}
	b.WriteString("Created-By: goplus\r\n\r\n")
	return []byte(b.String())
}

func inside(root, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("Java build output path must not be empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("Java build output %q must be relative", rel)
	}
	path := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	back, err := filepath.Rel(root, path)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("Java build output %q escapes module root", rel)
	}
	return path, nil
}

func ensureOwnedBuildDir(root, path string) error {
	back, err := filepath.Rel(root, path)
	if err != nil || back == "." || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing broad Java build directory %s", path)
	}
	return nil
}

func directoriesOverlap(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	return left == right || strings.HasPrefix(left, right+string(filepath.Separator)) || strings.HasPrefix(right, left+string(filepath.Separator))
}

func pathInside(path, directory string) bool {
	path, directory = filepath.Clean(path), filepath.Clean(directory)
	return path == directory || strings.HasPrefix(path, directory+string(filepath.Separator))
}

func rejectSymlinkComponents(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuses symbolic-link component %s", current)
		}
	}
	return nil
}

func cleanDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func executable(name string) string {
	if os.PathSeparator == '\\' {
		return name + ".exe"
	}
	return name
}
