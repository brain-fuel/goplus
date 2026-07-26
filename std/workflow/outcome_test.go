package workflow_test

import (
	"errors"
	"testing"

	"goforge.dev/goplus/std/workflow"
)

func TestOutcomeError(t *testing.T) {
	boom := errors.New("boom")
	if got := workflow.OutcomeError(workflow.Failed{Err: boom}); got != boom {
		t.Fatalf("Failed → %v, want boom", got)
	}
	if got := workflow.OutcomeError(workflow.Skipped{Reason: "up to date"}); got != nil {
		t.Fatalf("Skipped → %v, want nil", got)
	}
	if got := workflow.OutcomeError(workflow.Completed{}); got != nil {
		t.Fatalf("Completed → %v, want nil", got)
	}
}

func TestOutcomeSucceeded(t *testing.T) {
	if !workflow.Succeeded(workflow.Completed{}) {
		t.Fatal("Completed succeeds")
	}
	if !workflow.Succeeded(workflow.Skipped{Reason: "x"}) {
		t.Fatal("Skipped succeeds (not a failure)")
	}
	if workflow.Succeeded(workflow.Failed{Err: errors.New("e")}) {
		t.Fatal("Failed does not succeed")
	}
}

func TestSkipReason(t *testing.T) {
	if r, ok := workflow.SkipReason(workflow.Skipped{Reason: "up to date"}); !ok || r != "up to date" {
		t.Fatalf("SkipReason(Skipped) = %q,%v", r, ok)
	}
	if _, ok := workflow.SkipReason(workflow.Completed{}); ok {
		t.Fatal("Completed is not a skip")
	}
}
