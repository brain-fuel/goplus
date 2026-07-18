package gpp_test

import (
	"testing"

	"github.com/cucumber/godog"

	"goforge.dev/gpp/internal/bddtest"
)

// TestFeatures runs the Gherkin spec suite under features/ with Godog.
// The feature files plus spec/grammar-*.ebnf are the gpp specification.
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		Name: "gpp",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			bddtest.InitializeScenario(t, sc)
		},
		Options: &godog.Options{
			Format:   "progress",
			Paths:    []string{"features"},
			Strict:   true,
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("feature suite failed")
	}
}
