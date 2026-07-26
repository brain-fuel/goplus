package process_test

import (
	"context"
	"strings"
	"testing"

	"goforge.dev/goplus/std/process"
)

// Clean runs with exactly the given environment; without it, Env is layered
// onto the inherited one.
func TestSpecCleanEnv(t *testing.T) {
	ctx := context.Background()
	t.Setenv("GPTEST_INHERITED", "yes")

	// Clean: only FOO is visible, GPTEST_INHERITED is not.
	out, err := process.Run(ctx, process.Spec{
		Path: "sh", Args: []string{"-c", "echo $FOO-$GPTEST_INHERITED"},
		Env: []string{"FOO=bar", "PATH=" + pathEnv()}, Clean: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out.Stdout)); got != "bar-" {
		t.Fatalf("clean env: got %q, want %q", got, "bar-")
	}

	// Not clean: FOO layered onto inherited env, GPTEST_INHERITED visible.
	out, err = process.Run(ctx, process.Spec{
		Path: "sh", Args: []string{"-c", "echo $FOO-$GPTEST_INHERITED"},
		Env: []string{"FOO=bar"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out.Stdout)); got != "bar-yes" {
		t.Fatalf("inherited env: got %q, want %q", got, "bar-yes")
	}
}

func pathEnv() string {
	out, _ := process.Run(context.Background(), process.Spec{Path: "sh", Args: []string{"-c", "echo $PATH"}})
	return strings.TrimSpace(string(out.Stdout))
}
