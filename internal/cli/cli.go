// Package cli implements the goplus command-line interface. It is a separate
// package from cmd/goplus so the Godog spec suite can drive the CLI in-process.
package cli

import (
	"fmt"
	"io"
	"os"

	"goforge.dev/goplus/internal/lsp"
	"goforge.dev/goplus/internal/version"
)

// Version is the goplus toolchain version reported by `goplus version`.
const Version = version.Version

const usageText = `goplus is the Go+ toolchain: a strict superset of Go targeting Go and Java 25+.

Usage:

	goplus <command> [arguments]

Commands:

	gen      generate target sources (flags: -target, -check, -stage)
	init     scaffold //go:generate wiring for this package (flag: -hook)
	lsp      speak the Language Server Protocol over stdio
	build    generate and build selected targets
	publish  build and publish a Java library to Maven Central
	test     generate and test selected targets
	run      generate and run one selected target
	vet      validate selected targets
	version  print goplus version
	help     print this help

Targets default to ["go"], or default_targets in goplus.toml. Use a repeatable
--target go|java flag to override the configured collection.
`

// Run executes the goplus CLI and returns its exit code.
//
// Exit codes: 0 success; 1 stale outputs under gen -check; 2 usage errors or
// Go+ diagnostics; delegated go commands pass their exit code through.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "gen":
		return runGen(rest, stdout, stderr)
	case "init":
		return runInit(rest, stdout, stderr)
	case "lsp":
		if err := lsp.Serve(os.Stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "goplus lsp: %v\n", err)
			return 1
		}
		return 0
	case "build", "test", "run", "vet":
		return runDelegated(cmd, rest, stdout, stderr)
	case "publish":
		return runPublish(rest, stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "goplus version %s\n", Version)
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	default:
		fmt.Fprintf(stderr, "goplus: unknown command %q\nRun 'goplus help' for usage.\n", cmd)
		return 2
	}
}
