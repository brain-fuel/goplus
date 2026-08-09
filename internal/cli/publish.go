package cli

import (
	"fmt"
	"io"
)

// runPublish is retained only so existing automation fails with an actionable
// migration message. Publication credentials and network operations belong to
// assayxport; Go+ only produces unsigned build artifacts.
func runPublish(_ []string, _ io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stderr, "goplus publish has moved to assayxport; run `go tool goplus build --target java ./...` followed by `ax publish`")
	return 2
}
