package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestProcessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PROCESS_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "ok")
	fmt.Fprint(os.Stderr, "bad")
	os.Exit(7)
}

func helperSpec(sensitive bool) Spec {
	return Spec{Path: os.Args[0], Args: []string{"-test.run=TestProcessHelper", "--", "secret"}, Env: []string{"GO_WANT_PROCESS_HELPER=1"}, Sensitive: sensitive}
}

func TestRunCapturesOutputAndFailure(t *testing.T) {
	out, err := Run(context.Background(), helperSpec(false))
	if string(out.Stdout) != "ok" || string(out.Stderr) != "bad" {
		t.Fatalf("output = %#v", out)
	}
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != 7 {
		t.Fatalf("failure = %#v, %v", failure, err)
	}
}

func TestSensitiveArgumentsAreRedacted(t *testing.T) {
	_, err := Run(context.Background(), helperSpec(true))
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("secret leaked: %v", err)
	}
}
