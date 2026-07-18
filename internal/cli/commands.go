package cli

import (
	"fmt"
	"io"
)

func runGen(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "gpp gen: not implemented yet")
	return 2
}

func runDelegated(cmd string, args []string, stdout, stderr io.Writer) int {
	fmt.Fprintf(stderr, "gpp %s: not implemented yet\n", cmd)
	return 2
}
