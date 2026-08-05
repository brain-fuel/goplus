// Package compiler exposes the target-neutral Go+ compilation API.
//
// Compile is side-effect free: it returns versioned artifact sets and
// diagnostics. The CLI materializes those artifacts, which lets embedders and
// build systems keep full ownership of filesystem and caching policy.
package compiler

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"goforge.dev/goplus/internal/gen"
	"goforge.dev/goplus/internal/javabackend"
	"goforge.dev/goplus/internal/javaffi"
)

// Target selects a compiler backend.
type Target string

const (
	TargetGo   Target = "go"
	TargetJava Target = "java"
)

// ArtifactRole describes a file without tying clients to a directory layout.
type ArtifactRole string

const (
	ArtifactSource   ArtifactRole = "source"
	ArtifactRuntime  ArtifactRole = "runtime-source"
	ArtifactManifest ArtifactRole = "manifest"
)

// Artifact is one output. Path is relative to Request.Dir and Data is stable
// for identical inputs and compiler versions.
type Artifact struct {
	Path string
	Role ArtifactRole
	Data []byte
}

// ArtifactSet is one backend's independently versioned output contract.
type ArtifactSet struct {
	Schema      string
	Target      Target
	RuntimeABI  int
	ModulePath  string
	ModuleName  string
	MainClass   string
	TestClasses []string
	Artifacts   []Artifact
}

// Diagnostic is a stable public diagnostic record. Internal AST and type
// objects deliberately do not cross this API boundary.
type Diagnostic struct {
	Filename string
	Line     int
	Column   int
	Message  string
}

// JavaOptions configures the Java 25+ artifact set.
type JavaOptions struct {
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

// Request describes a pure compilation.
type Request struct {
	Dir      string
	Patterns []string
	Targets  []Target
	Overlay  map[string][]byte
	Java     JavaOptions
}

// Result contains no artifacts when diagnostics are present.
type Result struct {
	ArtifactSets []ArtifactSet
	Diagnostics  []Diagnostic
}

// Ok reports whether compilation completed without diagnostics.
func (r Result) Ok() bool { return len(r.Diagnostics) == 0 }

// Compile elaborates Go+ for each target tag set, then feeds the checked Go
// shadow to the selected backend. Common, untagged sources remain identical.
func Compile(ctx context.Context, request Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	dir := request.Dir
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Result{}, err
	}
	targets := request.Targets
	if len(targets) == 0 {
		targets = []Target{TargetGo}
	}
	seen := map[Target]bool{}
	for _, target := range targets {
		if target != TargetGo && target != TargetJava {
			return Result{}, fmt.Errorf("unknown compilation target %q", target)
		}
		if seen[target] {
			return Result{}, fmt.Errorf("compilation target %q is listed twice", target)
		}
		seen[target] = true
	}

	result := Result{}
	shadows := make(map[Target]*gen.Result, len(targets))
	for _, target := range targets {
		var tags []string
		overlay := request.Overlay
		if target == TargetJava {
			tags = []string{"goplus_java", "java25", "jvm64"}
			overlay, err = javaffi.Prepare(abs, request.Overlay)
			if err != nil {
				return Result{}, err
			}
		}
		shadow, err := gen.Run(gen.Options{
			Dir: abs, Patterns: request.Patterns, Overlay: overlay,
			DryRun: true, BuildTags: tags,
		})
		if err != nil {
			return Result{}, err
		}
		shadows[target] = shadow
		for _, diagnostic := range shadow.Diags {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Filename: diagnostic.Pos.Filename,
				Line:     diagnostic.Pos.Line, Column: diagnostic.Pos.Column,
				Message: diagnostic.Msg,
			})
		}
	}
	if !result.Ok() {
		return result, nil
	}

	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		switch target {
		case TargetGo:
			shadow := shadows[target]
			set := ArtifactSet{Schema: "goplus.go.artifacts/v1", Target: TargetGo}
			var paths []string
			for path := range shadow.Outputs {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			for _, path := range paths {
				rel, relErr := filepath.Rel(abs, path)
				if relErr != nil {
					rel = path
				}
				set.Artifacts = append(set.Artifacts, Artifact{
					Path: filepath.ToSlash(rel), Role: ArtifactSource,
					Data: append([]byte(nil), shadow.Outputs[path]...),
				})
			}
			result.ArtifactSets = append(result.ArtifactSets, set)
		case TargetJava:
			shadow := shadows[target]
			javaResult, err := javabackend.Generate(ctx, javabackend.Config{
				Root: abs, Patterns: request.Patterns, Overlay: shadow.Outputs,
				Release: request.Java.Release, Kind: request.Java.Kind,
				SourceDir: request.Java.SourceDir, RuntimeSourceDir: request.Java.RuntimeSourceDir,
				PackagePrefix: request.Java.PackagePrefix, ModuleName: request.Java.ModuleName,
				MainPackage: request.Java.MainPackage, StrongModule: request.Java.StrongModule,
				Bundle:       request.Java.Bundle,
				IncludeTests: request.Java.IncludeTests,
				JDKHome:      request.Java.JDKHome,
				Classpath:    append([]string(nil), request.Java.Classpath...),
				Modulepath:   append([]string(nil), request.Java.Modulepath...),
			})
			if err != nil {
				return Result{}, err
			}
			for _, diagnostic := range javaResult.Diagnostics {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Filename: diagnostic.Pos.Filename, Line: diagnostic.Pos.Line,
					Column: diagnostic.Pos.Column, Message: diagnostic.Message,
				})
			}
			if len(javaResult.Diagnostics) > 0 {
				return result, nil
			}
			set := ArtifactSet{
				Schema: javabackend.ArtifactSchema, Target: TargetJava,
				RuntimeABI: javabackend.RuntimeABI, ModulePath: javaResult.ModulePath,
				ModuleName: javaResult.ModuleName, MainClass: javaResult.MainClass,
				TestClasses: append([]string(nil), javaResult.TestClasses...),
			}
			for _, artifact := range javaResult.Artifacts {
				role := ArtifactSource
				switch artifact.Role {
				case javabackend.RoleRuntime:
					role = ArtifactRuntime
				case javabackend.RoleManifest:
					role = ArtifactManifest
				}
				set.Artifacts = append(set.Artifacts, Artifact{
					Path: artifact.Path, Role: role, Data: append([]byte(nil), artifact.Data...),
				})
			}
			result.ArtifactSets = append(result.ArtifactSets, set)
		}
	}
	return result, nil
}
