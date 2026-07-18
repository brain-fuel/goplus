package bddtest

// Step definitions for the v0.5.0 frontend: class and instance
// declarations.

import (
	"fmt"

	"github.com/cucumber/godog"
)

func initParsingV5Steps(sc *godog.ScenarioContext, ps *parseState) {
	sc.Step(`^parsing succeeds with (\d+) class(?:es)? and (\d+) instances?$`, func(classes, instances int) error {
		if ps.err != nil {
			return fmt.Errorf("parsing failed: %v", ps.err)
		}
		if got := len(ps.file.Classes); got != classes {
			return fmt.Errorf("found %d classes, want %d", got, classes)
		}
		if got := len(ps.file.Instances); got != instances {
			return fmt.Errorf("found %d instances, want %d", got, instances)
		}
		return nil
	})

	sc.Step(`^class (\d+) has (\d+) embeds?, (\d+) ops?, and (\d+) laws?$`, func(idx, embeds, ops, laws int) error {
		if ps.err != nil {
			return fmt.Errorf("parsing failed: %v", ps.err)
		}
		if idx >= len(ps.file.Classes) {
			return fmt.Errorf("class %d out of range (%d classes)", idx, len(ps.file.Classes))
		}
		c := ps.file.Classes[idx]
		var ne, no, nl int
		for _, m := range c.Members {
			switch {
			case m.Embed != nil:
				ne++
			case m.LawPos.IsValid():
				nl++
			default:
				no++
			}
		}
		if ne != embeds || no != ops || nl != laws {
			return fmt.Errorf("class %d: %d embeds, %d ops, %d laws; want %d/%d/%d",
				idx, ne, no, nl, embeds, ops, laws)
		}
		return nil
	})

	sc.Step(`^class (\d+) op "([^"]+)" has a default body$`, func(idx int, name string) error {
		if idx >= len(ps.file.Classes) {
			return fmt.Errorf("class %d out of range", idx)
		}
		for _, m := range ps.file.Classes[idx].Members {
			if m.Name != nil && m.Name.Name == name && !m.LawPos.IsValid() {
				if m.Body == nil {
					return fmt.Errorf("op %s has no default body", name)
				}
				return nil
			}
		}
		return fmt.Errorf("class %d has no op %s", idx, name)
	})

	sc.Step(`^instance (\d+) is named "([^"]+)" for class "([^"]+)"$`, func(idx int, name, class string) error {
		if ps.err != nil {
			return fmt.Errorf("parsing failed: %v", ps.err)
		}
		if idx >= len(ps.file.Instances) {
			return fmt.Errorf("instance %d out of range (%d instances)", idx, len(ps.file.Instances))
		}
		inst := ps.file.Instances[idx]
		if inst.Name.Name != name {
			return fmt.Errorf("instance %d is named %q, want %q", idx, inst.Name.Name, name)
		}
		got := srcText(ps.file, inst.Class.Pos(), inst.Class.End())
		if got != class {
			return fmt.Errorf("instance %d class = %q, want %q", idx, got, class)
		}
		return nil
	})
}
