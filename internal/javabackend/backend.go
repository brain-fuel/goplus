// Package javabackend emits deterministic Java 25 source from the typed Go
// shadow produced by the Go+ front end.
package javabackend

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"goforge.dev/goplus/internal/javaclass"
)

const (
	RuntimeABI     = 1
	ArtifactSchema = "goplus.java.artifacts/v1"
	RuntimePackage = "dev.goforge.goplus.runtime"
)

// Config controls Java source layout and the stable host-language ABI.
type Config struct {
	Root             string
	Patterns         []string
	Overlay          map[string][]byte
	Release          int
	Kind             string
	SourceDir        string
	RuntimeSourceDir string
	PackagePrefix    string
	ModuleName       string
	MainPackage      string
	StrongModule     bool
	Bundle           bool
	IncludeTests     bool
	JDKHome          string
	Classpath        []string
	Modulepath       []string
}

// Role classifies an emitted artifact.
type Role string

const (
	RoleSource   Role = "source"
	RoleRuntime  Role = "runtime-source"
	RoleManifest Role = "manifest"
)

// Artifact is one deterministic output, with Path relative to Config.Root.
type Artifact struct {
	Path string
	Role Role
	Data []byte
}

// Diagnostic is a source-positioned Java portability error.
type Diagnostic struct {
	Pos     token.Position
	Message string
}

// Result is the Java artifact set or a list of unsupported-source diagnostics.
type Result struct {
	Artifacts   []Artifact
	Diagnostics []Diagnostic
	ModulePath  string
	ModuleName  string
	MainClass   string
	TestClasses []string
}

// Generate loads the fully lowered Go shadow with go/types and emits Java 25
// sources. Unsupported constructs are diagnostics; no partial Java is returned.
func Generate(ctx context.Context, cfg Config) (Result, error) {
	if cfg.Release == 0 {
		cfg.Release = 25
	}
	if cfg.Release < 25 {
		return Result{}, fmt.Errorf("Java release must be at least 25")
	}
	if cfg.Kind == "" {
		cfg.Kind = "library"
	}
	if cfg.Kind != "library" && cfg.Kind != "app" {
		return Result{}, fmt.Errorf("Java kind must be %q or %q", "library", "app")
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = "gen/java"
	}
	if cfg.RuntimeSourceDir == "" {
		cfg.RuntimeSourceDir = ".goplus/build/java/runtime-src"
	}
	for _, item := range []struct{ name, path string }{
		{"source directory", cfg.SourceDir},
		{"runtime source directory", cfg.RuntimeSourceDir},
	} {
		if !safeRelativePath(item.path) {
			return Result{}, fmt.Errorf("Java %s %q must be relative and stay within the module root", item.name, item.path)
		}
	}
	if pathsOverlap(cfg.SourceDir, cfg.RuntimeSourceDir) {
		return Result{}, fmt.Errorf("Java source and runtime source directories must not overlap")
	}
	if cfg.PackagePrefix != "" && !javaNamespace(cfg.PackagePrefix) {
		return Result{}, fmt.Errorf("invalid Java package prefix %q", cfg.PackagePrefix)
	}
	if cfg.ModuleName != "" && !javaNamespace(cfg.ModuleName) {
		return Result{}, fmt.Errorf("invalid Java module name %q", cfg.ModuleName)
	}
	patterns := normalizePatterns(cfg.Patterns)
	pcfg := &packages.Config{
		Context: ctx,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedModule |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:        cfg.Root,
		Overlay:    cfg.Overlay,
		BuildFlags: []string{"-tags=goplus_java,java25,jvm64"},
		Tests:      cfg.IncludeTests,
	}
	pkgs, err := packages.Load(pcfg, patterns...)
	if err != nil {
		return Result{}, fmt.Errorf("loading Java target packages: %w", err)
	}
	result := Result{}
	for _, pkg := range pkgs {
		for _, perr := range pkg.Errors {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Pos:     parsePackagePosition(perr.Pos),
				Message: perr.Msg,
			})
		}
	}
	if len(result.Diagnostics) > 0 {
		sortDiagnostics(result.Diagnostics)
		return result, nil
	}

	modulePath, _ := rootModule(pkgs, cfg.Root)
	if modulePath == "" {
		return Result{}, fmt.Errorf("Java target requires a Go module")
	}
	result.ModulePath = modulePath
	prefix := cfg.PackagePrefix
	if prefix == "" {
		prefix = JavaPackage(modulePath)
	}
	moduleName := cfg.ModuleName
	if moduleName == "" {
		moduleName = prefix
	}
	result.ModuleName = moduleName

	mapper := packageMapper{modulePath: modulePath, prefix: prefix}
	pkgs = emissionPackages(pkgs, modulePath)
	if requirements := javaRequirements(pkgs); len(requirements) > 0 {
		index, err := javaclass.Open(cfg.JDKHome, append(append([]string{}, cfg.Classpath...), cfg.Modulepath...), cfg.Release)
		if err != nil {
			return Result{}, fmt.Errorf("loading Java metadata: %w", err)
		}
		for _, requirement := range requirements {
			class, lookupErr := index.Lookup(requirement.owner)
			if lookupErr != nil {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Pos: requirement.pos, Message: lookupErr.Error()})
				continue
			}
			found := true
			switch requirement.kind {
			case "constructor":
				found = class.HasConstructor(requirement.descriptor)
			case "static":
				found = class.HasStaticMethod(requirement.member, requirement.descriptor)
			case "virtual":
				found = class.HasVirtualMethod(requirement.member, requirement.descriptor)
			}
			if requirement.descriptor != "" && !found {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Pos: requirement.pos, Message: fmt.Sprintf("Java member %s.%s%s was not found", requirement.owner, requirement.member, requirement.descriptor)})
			}
		}
		if len(result.Diagnostics) > 0 {
			sortDiagnostics(result.Diagnostics)
			return result, nil
		}
	}
	var sourceArtifacts []Artifact
	seenPkg := map[string]bool{}
	javaPackageOwner := map[string]string{}
	sourceOwner := map[string]string{}
	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.PkgPath == "" || seenPkg[pkg.PkgPath] {
			continue
		}
		seenPkg[pkg.PkgPath] = true
		// packages.Load returns only the requested roots at this level. Imported
		// dependencies live under pkg.Imports and are intentionally not emitted.
		em := newEmitter(pkg, mapper)
		files, tests, diags := em.emitPackage(cfg.Kind == "app" && isMainPackage(cfg.MainPackage, pkg.PkgPath, modulePath), cfg.IncludeTests)
		result.Diagnostics = append(result.Diagnostics, diags...)
		javaPkg := mapper.javaPackage(pkg.PkgPath)
		if owner := javaPackageOwner[javaPkg]; owner != "" && owner != pkg.PkgPath {
			return Result{}, fmt.Errorf("Go packages %s and %s both map to Java package %s", owner, pkg.PkgPath, javaPkg)
		}
		javaPackageOwner[javaPkg] = pkg.PkgPath
		for name, data := range files {
			rel := filepath.ToSlash(filepath.Join(cfg.SourceDir, filepath.FromSlash(strings.ReplaceAll(javaPkg, ".", "/")), name))
			if owner := sourceOwner[rel]; owner != "" {
				return Result{}, fmt.Errorf("Java artifact path %s is emitted by both %s and %s", rel, owner, pkg.PkgPath)
			}
			sourceOwner[rel] = pkg.PkgPath
			sourceArtifacts = append(sourceArtifacts, Artifact{Path: rel, Role: RoleSource, Data: data})
		}
		if cfg.Kind == "app" && isMainPackage(cfg.MainPackage, pkg.PkgPath, modulePath) {
			result.MainClass = javaPkg + ".GpPackage"
		}
		if len(tests) > 0 {
			result.TestClasses = append(result.TestClasses, javaPkg+".GpTests")
		}
	}
	if len(result.Diagnostics) > 0 {
		sortDiagnostics(result.Diagnostics)
		return result, nil
	}
	if cfg.Kind == "app" && result.MainClass == "" {
		return Result{}, fmt.Errorf("Java app main package %q was not among %s", cfg.MainPackage, strings.Join(patterns, ", "))
	}

	result.Artifacts = append(result.Artifacts, sourceArtifacts...)
	for name, data := range RuntimeSources() {
		rel := filepath.ToSlash(filepath.Join(cfg.RuntimeSourceDir, name))
		if name != "module-info.java" {
			rel = filepath.ToSlash(filepath.Join(cfg.RuntimeSourceDir, filepath.FromSlash(strings.ReplaceAll(RuntimePackage, ".", "/")), name))
		}
		result.Artifacts = append(result.Artifacts, Artifact{Path: rel, Role: RoleRuntime, Data: data})
	}
	if cfg.StrongModule {
		var exports []string
		for _, pkg := range pkgs {
			if pkg.PkgPath != "" {
				exports = append(exports, mapper.javaPackage(pkg.PkgPath))
			}
		}
		sort.Strings(exports)
		result.Artifacts = append(result.Artifacts, Artifact{
			Path: filepath.ToSlash(filepath.Join(cfg.SourceDir, "module-info.java")),
			Role: RoleSource,
			Data: moduleInfo(moduleName, exports, cfg.Bundle),
		})
	}
	sort.Slice(result.Artifacts, func(i, j int) bool { return result.Artifacts[i].Path < result.Artifacts[j].Path })

	manifest := struct {
		Schema      string   `json:"schema"`
		Target      string   `json:"target"`
		Release     int      `json:"release"`
		RuntimeABI  int      `json:"runtime_abi"`
		ModulePath  string   `json:"module_path"`
		ModuleName  string   `json:"module_name"`
		Kind        string   `json:"kind"`
		MainClass   string   `json:"main_class,omitempty"`
		TestClasses []string `json:"test_classes,omitempty"`
		ModuleRoot  string   `json:"module_root"`
		Artifacts   []string `json:"artifacts"`
	}{
		Schema: ArtifactSchema, Target: "java", Release: cfg.Release,
		RuntimeABI: RuntimeABI, ModulePath: modulePath, ModuleName: moduleName,
		Kind: cfg.Kind, MainClass: result.MainClass, TestClasses: result.TestClasses, ModuleRoot: ".",
	}
	for _, artifact := range result.Artifacts {
		manifest.Artifacts = append(manifest.Artifacts, artifact.Path)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, err
	}
	data = append(data, '\n')
	result.Artifacts = append(result.Artifacts, Artifact{
		Path: filepath.ToSlash(filepath.Join(cfg.SourceDir, "goplus-artifacts.json")),
		Role: RoleManifest,
		Data: data,
	})
	sort.Slice(result.Artifacts, func(i, j int) bool { return result.Artifacts[i].Path < result.Artifacts[j].Path })
	return result, nil
}

type javaRequirement struct {
	kind, owner, member, descriptor string
	pos                             token.Position
}

func javaRequirements(pkgs []*packages.Package) []javaRequirement {
	seen := map[string]bool{}
	var out []javaRequirement
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(strings.TrimSpace(comment.Text))
					if len(fields) == 3 && fields[0] == "//goplus:java-type" {
						key := fields[2]
						if !seen[key] {
							seen[key] = true
							out = append(out, javaRequirement{owner: fields[2], pos: pkg.Fset.Position(comment.Pos())})
						}
					}
				}
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				binding, err := parseJavaBinding(fn)
				if err != nil || binding == nil {
					continue
				}
				member := binding.Member
				if binding.Kind == "constructor" {
					member = "<init>"
				}
				key := binding.Owner + member + binding.Descriptor
				if !seen[key] {
					seen[key] = true
					out = append(out, javaRequirement{kind: binding.Kind, owner: binding.Owner, member: member, descriptor: binding.Descriptor, pos: pkg.Fset.Position(fn.Pos())})
				}
			}
		}
	}
	return out
}

func emissionPackages(pkgs []*packages.Package, modulePath string) []*packages.Package {
	best := map[string]*packages.Package{}
	seenID := map[string]bool{}
	var visit func(*packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil || seenID[pkg.ID] {
			return
		}
		seenID[pkg.ID] = true
		if pkg.PkgPath == "" || strings.HasSuffix(pkg.PkgPath, ".test") {
			return
		}
		if current := best[pkg.PkgPath]; current == nil || len(pkg.Syntax) > len(current.Syntax) {
			best[pkg.PkgPath] = pkg
		}
		for path, imported := range pkg.Imports {
			if path == modulePath || strings.HasPrefix(path, modulePath+"/") {
				visit(imported)
			}
		}
	}
	for _, pkg := range pkgs {
		visit(pkg)
	}
	paths := make([]string, 0, len(best))
	for path := range best {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]*packages.Package, 0, len(paths))
	for _, path := range paths {
		out = append(out, best[path])
	}
	return out
}

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return []string{"./..."}
	}
	out := make([]string, len(patterns))
	for i, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if pattern != "." && !strings.HasPrefix(pattern, "./") && !strings.HasPrefix(pattern, "/") {
			pattern = "./" + pattern
		}
		out[i] = pattern
	}
	return out
}

func rootModule(pkgs []*packages.Package, fallback string) (path, root string) {
	for _, pkg := range pkgs {
		if pkg.Module != nil && pkg.Module.Path != "" {
			return pkg.Module.Path, pkg.Module.Dir
		}
	}
	return "", fallback
}

func isMainPackage(configured, pkgPath, modulePath string) bool {
	if configured != "" {
		return configured == pkgPath
	}
	return pkgPath == modulePath
}

func moduleInfo(name string, exports []string, bundledRuntime bool) []byte {
	var b strings.Builder
	b.WriteString("// Code generated by goplus for Java 25+. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "module %s {\n", name)
	if !bundledRuntime {
		b.WriteString("    requires dev.goforge.goplus.runtime;\n")
	}
	for _, export := range exports {
		fmt.Fprintf(&b, "    exports %s;\n", export)
	}
	if bundledRuntime {
		fmt.Fprintf(&b, "    exports %s;\n", RuntimePackage)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

func parsePackagePosition(text string) token.Position {
	if text == "" || text == "-" {
		return token.Position{}
	}
	rest := text
	var column, line int
	if index := strings.LastIndex(rest, ":"); index > 1 {
		if number, err := strconv.Atoi(rest[index+1:]); err == nil {
			column = number
			rest = rest[:index]
		}
	}
	if index := strings.LastIndex(rest, ":"); index > 1 {
		if number, err := strconv.Atoi(rest[index+1:]); err == nil {
			line = number
			rest = rest[:index]
		}
	}
	if line == 0 {
		line, column = column, 0
	}
	return token.Position{Filename: rest, Line: line, Column: column}
}

func sortDiagnostics(diags []Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].Pos.Filename != diags[j].Pos.Filename {
			return diags[i].Pos.Filename < diags[j].Pos.Filename
		}
		if diags[i].Pos.Line != diags[j].Pos.Line {
			return diags[i].Pos.Line < diags[j].Pos.Line
		}
		if diags[i].Pos.Column != diags[j].Pos.Column {
			return diags[i].Pos.Column < diags[j].Pos.Column
		}
		return diags[i].Message < diags[j].Message
	})
}

func safeRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func pathsOverlap(left, right string) bool {
	left = filepath.Clean(filepath.FromSlash(left))
	right = filepath.Clean(filepath.FromSlash(right))
	return left == right || strings.HasPrefix(left, right+string(filepath.Separator)) || strings.HasPrefix(right, left+string(filepath.Separator))
}

func javaNamespace(name string) bool {
	if !javaQualifiedName(name) {
		return false
	}
	for _, part := range strings.Split(name, ".") {
		if javaReserved[part] {
			return false
		}
	}
	return true
}
