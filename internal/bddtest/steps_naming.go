package bddtest

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"

	"goforge.dev/gpp/internal/naming"
	"goforge.dev/gpp/internal/registry"
	"goforge.dev/gpp/internal/syntax"
)

type namingState struct {
	recvType, method, override string

	loweredNames []string
	errs         []error
}

func initNamingSteps(sc *godog.ScenarioContext, w func() *World, ns *namingState) {
	sc.Step(`^a receiver type "([^"]+)" and method "([^"]+)"$`, func(recvType, method string) error {
		ns.recvType, ns.method, ns.override = recvType, method, ""
		return nil
	})
	sc.Step(`^a receiver type "([^"]+)" and method "([^"]+)" with name override "([^"]+)"$`, func(recvType, method, override string) error {
		ns.recvType, ns.method, ns.override = recvType, method, override
		return nil
	})
	sc.Step(`^the lowered function name is "([^"]+)"$`, func(want string) error {
		got := naming.FuncName(ns.recvType, ns.method, ns.override)
		if got != want {
			return fmt.Errorf("FuncName(%q, %q, %q) = %q, want %q", ns.recvType, ns.method, ns.override, got, want)
		}
		return nil
	})

	sc.Step(`^I compute lowered names$`, func() error {
		world := w()
		src, err := os.ReadFile(filepath.Join(world.Dir, world.LastGppFile))
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := syntax.ParseFile(fset, world.LastGppFile, src)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
		tbl := naming.NewTable()
		for _, d := range naming.TopLevelDecls(fset, f.AST) {
			tbl.AddAuthored(d.Name, d.Position)
		}
		methods, errs := registry.MethodsFromFile("example.test/pkg", f, tbl)
		ns.errs = errs
		ns.loweredNames = nil
		for _, m := range methods {
			ns.loweredNames = append(ns.loweredNames, m.FuncName)
		}
		return nil
	})
	sc.Step(`^name generation fails with an error containing "([^"]*)"$`, func(want string) error {
		if len(ns.errs) == 0 {
			return fmt.Errorf("name generation succeeded (%v), expected error containing %q", ns.loweredNames, want)
		}
		var all []string
		for _, e := range ns.errs {
			if strings.Contains(e.Error(), want) {
				return nil
			}
			all = append(all, e.Error())
		}
		return fmt.Errorf("no error contains %q; errors:\n%s", want, strings.Join(all, "\n"))
	})
	sc.Step(`^the lowered names are "([^"]*)"$`, func(want string) error {
		if len(ns.errs) > 0 {
			return fmt.Errorf("unexpected errors: %v", ns.errs)
		}
		got := strings.Join(ns.loweredNames, ", ")
		if got != want {
			return fmt.Errorf("lowered names = %q, want %q", got, want)
		}
		return nil
	})
}
