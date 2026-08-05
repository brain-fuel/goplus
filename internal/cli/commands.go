package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"goforge.dev/goplus/compiler"
	"goforge.dev/goplus/internal/artifactio"
	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/gen"
	"goforge.dev/goplus/internal/javatool"
	"goforge.dev/goplus/internal/projectconfig"
	"goforge.dev/goplus/internal/toolchain"
)

type targetFlags []string

func (t *targetFlags) String() string { return strings.Join(*t, ",") }
func (t *targetFlags) Set(value string) error {
	*t = append(*t, value)
	return nil
}

func runGen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("goplus gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "verify generated files are current; write nothing (exit 1 when stale)")
	stage := fs.Bool("stage", false, "git-add written and deleted files after generating")
	var targetArgs targetFlags
	fs.Var(&targetArgs, "target", "generation target: go or java (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 2
	}
	cfg, err := projectconfig.Load(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 2
	}
	targets, err := selectTargets(targetArgs, cfg.DefaultTargets)
	if err != nil {
		fmt.Fprintf(stderr, "goplus gen: %v\n", err)
		return 2
	}
	stale := false
	for _, target := range targets {
		switch target {
		case compiler.TargetGo:
			res, err := gen.Run(gen.Options{Dir: cwd, Patterns: fs.Args(), Check: *check, Stage: *stage})
			if err != nil {
				fmt.Fprintf(stderr, "goplus: %v\n", err)
				return 2
			}
			if !res.Ok() {
				diag.Render(stderr, res.Diags)
				return 2
			}
			for _, path := range res.Written {
				fmt.Fprintln(stdout, path)
			}
			if len(res.Stale) > 0 {
				stale = true
				renderStale(stderr, "Go", res.Stale)
			}
		case compiler.TargetJava:
			set, code := compileJava(cwd, cfg, fs.Args(), false, stderr)
			if code != 0 {
				return code
			}
			synced, err := artifactio.Sync(cfg.Root, set, artifactio.Options{
				Check: *check, Stage: *stage, IgnoreRuntimeInCheck: *check,
			})
			if err != nil {
				fmt.Fprintf(stderr, "goplus: %v\n", err)
				return 2
			}
			for _, path := range synced.Written {
				fmt.Fprintln(stdout, path)
			}
			if len(synced.Stale) > 0 {
				stale = true
				renderStale(stderr, "Java", synced.Stale)
			}
		}
	}
	if *check && stale {
		fmt.Fprintln(stderr, "run `goplus gen ./...` and re-stage.")
		return 1
	}
	return 0
}

// runDelegated builds one or more configured targets. Go retains its exact
// generate-then-delegate behavior; Java uses the source/JAR artifact pipeline.
func runDelegated(sub string, args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 2
	}
	rest, explicit, err := extractTargets(args)
	if err != nil {
		fmt.Fprintf(stderr, "goplus %s: %v\n", sub, err)
		return 2
	}
	cfg, err := projectconfig.Load(cwd)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 2
	}
	targets, err := selectTargets(explicit, cfg.DefaultTargets)
	if err != nil {
		fmt.Fprintf(stderr, "goplus %s: %v\n", sub, err)
		return 2
	}
	if sub == "run" && len(targets) != 1 {
		fmt.Fprintln(stderr, "goplus run: select exactly one target")
		return 2
	}
	for _, target := range targets {
		switch target {
		case compiler.TargetGo:
			res, err := gen.Run(gen.Options{Dir: cwd, Patterns: []string{"./..."}})
			if err != nil {
				fmt.Fprintf(stderr, "goplus: %v\n", err)
				return 2
			}
			if !res.Ok() {
				diag.Render(stderr, res.Diags)
				return 2
			}
			if code := toolchain.Go(cwd, sub, rest, stdout, stderr); code != 0 {
				return code
			}
		case compiler.TargetJava:
			if code := runJavaCommand(sub, cwd, cfg, rest, stdout, stderr); code != 0 {
				return code
			}
		}
	}
	return 0
}

func compileJava(cwd string, cfg projectconfig.Config, patterns []string, includeTests bool, stderr io.Writer) (compiler.ArtifactSet, int) {
	classpath, err := javatool.ReadPathFiles(cfg.Root, cfg.Java.ClasspathFiles)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return compiler.ArtifactSet{}, 2
	}
	modulepath, err := javatool.ReadPathFiles(cfg.Root, cfg.Java.ModulepathFiles)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return compiler.ArtifactSet{}, 2
	}
	result, err := compiler.Compile(context.Background(), compiler.Request{
		Dir: cfg.Root, Patterns: patternsFrom(cwd, cfg.Root, patterns),
		Targets: []compiler.Target{compiler.TargetJava},
		Java: compiler.JavaOptions{
			Release: cfg.Java.Release, Kind: cfg.Java.Kind,
			SourceDir: cfg.Java.SourceDir, RuntimeSourceDir: ".goplus/build/java/runtime-src",
			PackagePrefix: cfg.Java.PackagePrefix, ModuleName: cfg.Java.ModuleName,
			MainPackage: cfg.Java.MainPackage, StrongModule: cfg.Java.StrongModule,
			Bundle:       cfg.Java.Bundle,
			IncludeTests: includeTests,
			Classpath:    classpath, Modulepath: modulepath,
		},
	})
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return compiler.ArtifactSet{}, 2
	}
	if !result.Ok() {
		renderCompilerDiagnostics(stderr, result.Diagnostics)
		return compiler.ArtifactSet{}, 2
	}
	return result.ArtifactSets[0], 0
}

func runJavaCommand(sub, cwd string, cfg projectconfig.Config, args []string, stdout, stderr io.Writer) int {
	patterns, programArgs, err := javaCommandArgs(sub, args)
	if err != nil {
		fmt.Fprintf(stderr, "goplus %s --target java: %v\n", sub, err)
		return 2
	}
	set, code := compileJava(cwd, cfg, patterns, sub == "test", stderr)
	if code != 0 {
		return code
	}
	if _, err := artifactio.Sync(cfg.Root, set, artifactio.Options{}); err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 2
	}
	tool, err := javatool.Resolve(context.Background(), cfg.Java.Release)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 1
	}
	build, err := javatool.Build(context.Background(), tool, javatool.Config{
		Root: cfg.Root, Release: cfg.Java.Release,
		SourceDir: cfg.Java.SourceDir, RuntimeSourceDir: ".goplus/build/java/runtime-src",
		ClassDir: cfg.Java.ClassDir, Jar: cfg.Java.Jar, RuntimeJar: cfg.Java.RuntimeJar,
		ModuleName: set.ModuleName, MainClass: set.MainClass,
		ClasspathFiles: cfg.Java.ClasspathFiles, ModulepathFiles: cfg.Java.ModulepathFiles,
		StrongModule: cfg.Java.StrongModule, Bundle: cfg.Java.Bundle,
	}, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "goplus: %v\n", err)
		return 1
	}
	if sub == "run" {
		if err := javatool.Run(context.Background(), tool, build, programArgs, os.Stdin, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "goplus: %v\n", err)
			return 1
		}
	} else if sub == "test" {
		if len(set.TestClasses) == 0 {
			fmt.Fprintln(stdout, "?\tjava target\t[no test files]")
			return 0
		}
		for _, testClass := range set.TestClasses {
			testBuild := build
			testBuild.MainClass = testClass
			if err := javatool.Run(context.Background(), tool, testBuild, nil, os.Stdin, stdout, stderr); err != nil {
				return 1
			}
		}
	} else if sub == "build" {
		fmt.Fprintln(stdout, cfg.Java.Jar)
	}
	return 0
}

func selectTargets(explicit, defaults []string) ([]compiler.Target, error) {
	values := explicit
	if len(values) == 0 {
		values = defaults
	}
	seen := map[compiler.Target]bool{}
	var targets []compiler.Target
	for _, raw := range values {
		target := compiler.Target(strings.ToLower(strings.TrimSpace(raw)))
		if target != compiler.TargetGo && target != compiler.TargetJava {
			return nil, fmt.Errorf("unknown target %q (want go or java)", raw)
		}
		if seen[target] {
			return nil, fmt.Errorf("target %q is listed twice", target)
		}
		seen[target] = true
		targets = append(targets, target)
	}
	return targets, nil
}

func extractTargets(args []string) (rest []string, targets []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if value, ok := strings.CutPrefix(arg, "--target="); ok {
			targets = append(targets, value)
			continue
		}
		if value, ok := strings.CutPrefix(arg, "-target="); ok {
			targets = append(targets, value)
			continue
		}
		if arg == "--target" || arg == "-target" {
			if i+1 == len(args) {
				return nil, nil, fmt.Errorf("%s needs a value", arg)
			}
			i++
			targets = append(targets, args[i])
			continue
		}
		rest = append(rest, arg)
	}
	return rest, targets, nil
}

func patternsFrom(cwd, root string, patterns []string) []string {
	rel, err := filepath.Rel(root, cwd)
	if err != nil || rel == "." {
		return patterns
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if filepath.IsAbs(pattern) {
			out = append(out, filepath.ToSlash(pattern))
			continue
		}
		pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "./")
		if pattern == "." || pattern == "" {
			out = append(out, "./"+filepath.ToSlash(rel))
			continue
		}
		out = append(out, "./"+filepath.ToSlash(filepath.Join(rel, filepath.FromSlash(pattern))))
	}
	return out
}

func javaCommandArgs(sub string, args []string) (patterns, programArgs []string, err error) {
	if sub == "run" {
		separator := len(args)
		for i, arg := range args {
			if arg == "--" {
				separator = i
				break
			}
		}
		patterns = args[:separator]
		if separator < len(args) {
			programArgs = args[separator+1:]
		}
	} else {
		patterns = args
	}
	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "-") {
			return nil, nil, fmt.Errorf("unsupported Java-target option %q", pattern)
		}
	}
	if len(patterns) == 0 {
		if sub == "run" {
			patterns = []string{"."}
		} else {
			patterns = []string{"./..."}
		}
	}
	return patterns, programArgs, nil
}

func renderCompilerDiagnostics(w io.Writer, diagnostics []compiler.Diagnostic) {
	for _, diagnostic := range diagnostics {
		position := diagnostic.Filename
		if diagnostic.Line > 0 {
			position += fmt.Sprintf(":%d", diagnostic.Line)
		}
		if diagnostic.Column > 0 {
			position += fmt.Sprintf(":%d", diagnostic.Column)
		}
		if position != "" {
			position += ": "
		}
		fmt.Fprintf(w, "%s%s\n", position, diagnostic.Message)
	}
}

func renderStale(w io.Writer, target string, paths []string) {
	label := "generated code"
	if target != "Go" {
		label = "generated " + target + " code"
	}
	fmt.Fprintf(w, "goplus: stale %s:\n", label)
	for _, path := range paths {
		fmt.Fprintf(w, "  %s\n", path)
	}
}

// runInit scaffolds the go-generate wiring: a goplus_generate.go carrying
// the //go:generate directive, so `go generate ./...` regenerates the
// module and plain `go build` takes it from there — the canonical
// workflow, with the goplus wrapper as convenience. -hook also writes a
// pre-commit config entry.
func runInit(args []string, stdout, stderr io.Writer) int {
	hook := false
	for _, a := range args {
		switch a {
		case "-hook", "--hook":
			hook = true
		default:
			fmt.Fprintf(stderr, "goplus init: unknown argument %q\n", a)
			return 2
		}
	}
	pkg, err := detectPackageName(".")
	if err != nil {
		fmt.Fprintf(stderr, "goplus init: %v\n", err)
		return 2
	}
	const genFile = "goplus_generate.go"
	if _, err := os.Stat(genFile); err == nil {
		fmt.Fprintf(stderr, "goplus init: %s already exists\n", genFile)
		return 2
	}
	content := "// Scaffolded by goplus init. `go generate ./...` regenerates every\n" +
		"// *_gp.go in the module; plain `go build` works from there.\n\n" +
		"//go:generate go tool goplus gen ./...\n\n" +
		"package " + pkg + "\n"
	if err := os.WriteFile(genFile, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "goplus init: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote %s\n", genFile)
	if hook {
		const hookFile = ".pre-commit-config.yaml"
		if _, err := os.Stat(hookFile); err == nil {
			fmt.Fprintf(stdout, "%s already exists; add the goplus hook from the README manually\n", hookFile)
		} else {
			hookYAML := "repos:\n" +
				"  - repo: https://github.com/brain-fuel/goplus\n" +
				"    rev: " + Version + "\n" +
				"    hooks:\n" +
				"      - id: goplus-gen\n"
			if err := os.WriteFile(hookFile, []byte(hookYAML), 0o644); err != nil {
				fmt.Fprintf(stderr, "goplus init: %v\n", err)
				return 2
			}
			fmt.Fprintf(stdout, "wrote %s\n", hookFile)
		}
	}
	fmt.Fprintf(stdout, "next steps:\n")
	fmt.Fprintf(stdout, "\tgo get -tool goforge.dev/goplus/cmd/goplus@latest   # pins goplus in go.mod (Go 1.24+)\n")
	fmt.Fprintf(stdout, "\tgo generate ./...                             # regenerate\n")
	fmt.Fprintf(stdout, "\tgo build ./...                                # plain Go from here\n")
	return 0
}

// detectPackageName reads the package clause of any Go or Go+ file in
// dir, defaulting to "main" in an empty directory.
func detectPackageName(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?m)^package\s+(\w+)`)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || (!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".gp")) {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			continue
		}
		if m := re.FindSubmatch(src); m != nil {
			return string(m[1]), nil
		}
	}
	return "main", nil
}
