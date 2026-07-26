// Terminal task/step lifecycle outcomes.
//
// The saga runner (workflow.gp) drives a forward sequence that knows only
// success or error. A task DAG needs a richer terminal state: a step may be
// Skipped because a precondition already made it unnecessary (up-to-date, gated
// off), run to Completion, or Fail. Outcome makes those three exhaustive — the
// "done | failed | skipped" typestate — without conflating a skip with a
// failure.
package workflow

// Outcome is the terminal lifecycle state of a task or step.
type Outcome enum {
	// Skipped: a precondition made the step unnecessary (Reason explains which).
	Skipped(Reason string)
	// Completed: the step ran successfully.
	Completed()
	// Failed: the step ran and errored.
	Failed(Err error)
}

// OutcomeError projects an Outcome onto Go's error convention: a Failed outcome
// yields its error; Skipped and Completed both yield nil, because a skipped step
// is not a failure.
func OutcomeError(o Outcome) error {
	match o {
	case Failed(err):
		return err
	case Skipped(_):
		return nil
	case Completed():
		return nil
	}
}

// Succeeded reports whether the outcome is non-failing (Completed or Skipped).
func Succeeded(o Outcome) bool {
	match o {
	case Failed(_):
		return false
	case _:
		return true
	}
}

// SkipReason returns the skip reason and true when the outcome is Skipped.
func SkipReason(o Outcome) (string, bool) {
	match o {
	case Skipped(reason):
		return reason, true
	case _:
		return "", false
	}
}
