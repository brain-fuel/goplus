package javabackend

import "testing"

func TestReifiableJavaType(t *testing.T) {
	for input, want := range map[string]string{
		"Choice":       "Choice",
		"Result<T>":    "Result<?>",
		"Pair<A, B>":   "Pair<?, ?>",
		"broken<Value": "broken",
	} {
		got := reifiableJavaType(input)
		if got != want {
			t.Errorf("reifiableJavaType(%q) = %q, want %q", input, got, want)
		}
	}
}
