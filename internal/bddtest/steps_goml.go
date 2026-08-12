package bddtest

import (
	"os"

	"github.com/cucumber/godog"

	"goforge.dev/goplus/internal/goml"
)

// initGomlSteps wires the goml front end's CLI into scenarios; goml
// features twin the .gp features they mirror (spec/goml-design.md §7).
func initGomlSteps(sc *godog.ScenarioContext, world func() *World) {
	sc.Step(`^I run goml with arguments "([^"]*)"$`, func(args string) error {
		return world().runGoml(splitArgs(args))
	})
}

// runGoml invokes the goml CLI in-process with the scenario dir as the
// working directory.
func (w *World) runGoml(args []string) error {
	w.Stdout.Reset()
	w.Stderr.Reset()
	if err := os.Chdir(w.Dir); err != nil {
		return err
	}
	defer os.Chdir(w.origWD)
	w.ExitCode = goml.CLIRun(args, &w.Stdout, &w.Stderr)
	return nil
}
