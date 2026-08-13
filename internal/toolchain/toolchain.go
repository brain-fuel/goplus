// Package toolchain delegates to the standard go tool.
package toolchain

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// Go runs `go <sub> <args...>` in dir, wiring through stdio, and returns
// the exit code.
func Go(dir, sub string, args []string, stdout, stderr io.Writer) int {
	return GoWith(context.Background(), dir, sub, args, nil, os.Stdin, stdout, stderr)
}

// GoWith is Go with an explicit context, extra environment, and an
// injectable stdin. A REPL passes a nil stdin so the compiled program
// cannot consume the lines meant for the prompt.
func GoWith(ctx context.Context, dir, sub string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := exec.CommandContext(ctx, "go", append([]string{sub}, args...)...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		return 1
	}
	return 0
}
