package goml

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"goforge.dev/goplus/internal/diag"
	"goforge.dev/goplus/internal/version"
)

const usageText = `goml is the ML-family surface for the Go+ core.

Usage:

  goml gen [-check] [-stage] [patterns]   transpile .goml and generate *_gml.go
  goml convert [-o dir] file.goml...      print (or write) the .gp lowering
  goml version                            print the toolchain version

gen matches go-style package patterns (default ./...). Generation is
package-wide: packages mixing .gp and .goml regenerate both surfaces.
Exit codes: 0 ok, 1 stale outputs under -check, 2 usage or diagnostics.
`

// CLIRun is the goml command-line entry point.
func CLIRun(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	switch args[0] {
	case "gen":
		return runGen(args[1:], stdout, stderr)
	case "convert":
		return runConvert(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "goml version %s\n", version.Version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	}
	fmt.Fprintf(stderr, "goml: unknown command %q\n\n", args[0])
	fmt.Fprint(stderr, usageText)
	return 2
}

func runGen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "verify generated files are up to date; write nothing")
	stage := fs.Bool("stage", false, "git-add written outputs")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "goml: %v\n", err)
		return 2
	}
	res, err := Run(RunOptions{Dir: cwd, Patterns: fs.Args(), Check: *check, Stage: *stage})
	if err != nil {
		fmt.Fprintf(stderr, "goml: %v\n", err)
		return 2
	}
	if len(res.Gen.Diags) > 0 {
		diag.Render(stderr, res.Gen.Diags)
		return 2
	}
	if *check && len(res.Gen.Stale) > 0 {
		fmt.Fprintln(stdout, "goml: stale generated code:")
		for _, p := range res.Gen.Stale {
			fmt.Fprintf(stdout, "  %s\n", p)
		}
		return 1
	}
	for _, p := range res.Gen.Written {
		fmt.Fprintln(stdout, p)
	}
	return 0
}

func runConvert(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outDir := fs.String("o", "", "write <name>.gp files into this directory instead of stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "goml convert: no input files")
		return 2
	}
	for _, path := range paths {
		if !strings.HasSuffix(path, ".goml") {
			fmt.Fprintf(stderr, "goml convert: %s is not a .goml file\n", path)
			return 2
		}
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "goml convert: %v\n", err)
			return 2
		}
		gp, err := Convert(path, src)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		if *outDir == "" {
			if len(paths) > 1 {
				fmt.Fprintf(stdout, "-- %s\n", path)
			}
			stdout.Write(gp)
			continue
		}
		base := strings.TrimSuffix(filepath.Base(path), ".goml") + ".gp"
		target := filepath.Join(*outDir, base)
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			fmt.Fprintf(stderr, "goml convert: %v\n", err)
			return 2
		}
		if err := os.WriteFile(target, gp, 0o644); err != nil {
			fmt.Fprintf(stderr, "goml convert: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, target)
	}
	return 0
}
