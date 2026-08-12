package main

import (
	"os"

	"goforge.dev/goplus/internal/goml"
)

func main() {
	os.Exit(goml.CLIRun(os.Args[1:], os.Stdout, os.Stderr))
}
