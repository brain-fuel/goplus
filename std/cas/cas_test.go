package cas

import "testing"

func TestResultVariants(t *testing.T) {
	r := Updated[int]{Value: 2, Version: "v2"}
	got := Fold[int, string](r, ResultCases[int, string]{
		Updated: func(value int, version string) string { return version },
		Changed: func(int, string) string { return "changed" },
		Missing: func() string { return "missing" },
	})
	if got != "v2" {
		t.Fatalf("Fold = %q", got)
	}
}
