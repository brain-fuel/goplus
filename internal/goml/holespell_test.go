package goml

import "testing"

// The core answers a goal in its own notation; a goml author reads goml.
// A shape with no goml spelling keeps the original text rather than a
// mangled one.
func TestGomlSpelling(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Vec[a, n+1]", "Vec a (n + 1)"},
		{"Vec[a, n]", "Vec a n"},
		{"Vec[a]", "Vec a"},
		{"[]string", "Slice String"},
		{"map[string]int", "Map String Int"},
		{"*os.File", "Ptr os.File"},
		{"chan int", "Chan Int"},
		{"func(int) string", "Int -> String"},
		{"nat", "Nat"},
		{"n+1", "n + 1"},
		{"Fixed[p+q]", "Fixed (p + q)"},
		{"[][]int", "Slice (Slice Int)"},
		{"Region[Circle(n), n]", "Region[Circle(n), n]"},
		{"", ""},
	} {
		if got := gomlSpelling(tc.in); got != tc.want {
			t.Errorf("gomlSpelling(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
