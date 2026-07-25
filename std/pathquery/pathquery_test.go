package pathquery

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"", "", true},
		{"", "x", false},
		{"*", "", true},
		{"a*b", "ab", true},
		{"a*b", "axxxb", true},
		{"a?b", "a界b", true},
		{"?界", "世界", true},
		{"*界", "hello界", true},
		{`*\a`, "alpha", true},
		{`*\*`, "alpha", false},
		{`*\*`, "alpha*", true},
		{`*\?`, "friends", false},
		{`*\\?`, "friends", false},
		{"a*b", "axxxc", false},
		{"界?", "界", false},
	}
	for _, test := range tests {
		if got := Match(test.pattern, test.value); got != test.want {
			t.Errorf("Match(%q, %q) = %v, want %v", test.pattern, test.value, got, test.want)
		}
		if got := Compile(test.pattern).Match(test.value); got != test.want {
			t.Errorf("compiled %q against %q = %v, want %v", test.pattern, test.value, got, test.want)
		}
	}
}

func TestMatchAllocationBudget(t *testing.T) {
	pattern := Compile("prefix*中?尾")
	allocations := testing.AllocsPerRun(1000, func() {
		if !pattern.Match("prefix-value-中文尾") {
			panic("no match")
		}
	})
	if allocations != 0 {
		t.Fatalf("allocations = %v, want zero", allocations)
	}
}

func TestASCIIFastPathMatchesUnicodeKernel(t *testing.T) {
	patterns := []string{"", "*", "a?c", "a*b", `a\*b`, "**x?", "prefix*suffix"}
	values := []string{"", "abc", "a*b", "axxxb", "prefix-middle-suffix", "xx"}
	for _, pattern := range patterns {
		for _, value := range values {
			got := matchASCII(pattern, value)
			// Force the public implementation through the UTF-8 kernel by
			// appending and then removing a non-ASCII mismatch suffix.
			want := Match(pattern+"界", value+"界")
			if got != want {
				t.Errorf("ASCII %q / %q = %v, UTF-8 kernel analogue = %v",
					pattern, value, got, want)
			}
		}
	}
}

func TestRelationLaws(t *testing.T) {
	spellings := map[string]Relation{
		"=": Equal, "==": Equal, "!=": NotEqual,
		"<": Less, "<=": LessOrEqual, ">": Greater, ">=": GreaterOrEqual,
		"%": Like, "!%": NotLike,
	}
	for spelling, want := range spellings {
		got, ok := ParseRelation(spelling)
		if !ok || got != want {
			t.Errorf("ParseRelation(%q) = %v, %v; want %v, true", spelling, got, ok, want)
		}
	}
	if _, ok := ParseRelation("~"); ok {
		t.Fatal("unknown relation accepted")
	}

	equal := func(left, right int) bool { return left == right }
	less := func(left, right int) bool { return left < right }
	for left := -2; left <= 2; left++ {
		for right := -2; right <= 2; right++ {
			if Relate(left, right, Equal, equal, less) ==
				Relate(left, right, NotEqual, equal, less) {
				t.Errorf("equality complement failed for %d, %d", left, right)
			}
			if Relate(left, right, Less, equal, less) !=
				Relate(right, left, Greater, equal, less) {
				t.Errorf("ordering dual failed for %d, %d", left, right)
			}
			if Relate(left, right, LessOrEqual, equal, less) !=
				Relate(right, left, GreaterOrEqual, equal, less) {
				t.Errorf("inclusive ordering dual failed for %d, %d", left, right)
			}
		}
	}

	if !RelateString("gopher", "go*", Like) ||
		RelateString("gopher", "go*", NotLike) ||
		!RelateString("gopher", "rust*", NotLike) {
		t.Fatal("wildcard relation laws failed")
	}
}

func FuzzMatchNeverPanics(f *testing.F) {
	f.Add("a*b", "axxb")
	f.Add("?界", "世界")
	f.Fuzz(func(t *testing.T, pattern, value string) {
		_ = Match(pattern, value)
	})
}

var benchmarkMatched bool

func BenchmarkMatch(b *testing.B) {
	pattern := "service-*-region-?-中*尾"
	value := "service-payments-region-3-中文-path-尾"
	b.Run("goplus-stdlib", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkMatched = Match(pattern, value)
		}
	})
	b.Run("dynamic-programming-baseline", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkMatched = dynamicProgrammingMatch(pattern, value)
		}
	})
}

func BenchmarkMatchASCII(b *testing.B) {
	pattern := "service-*-region-?-tail"
	value := "service-payments-region-3-long-path-tail"
	b.Run("ascii-fast-path", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkMatched = Match(pattern, value)
		}
	})
	b.Run("utf8-general-path", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkMatched = matchUTF8(pattern, value)
		}
	})
}

func dynamicProgrammingMatch(pattern, value string) bool {
	patternRunes, valueRunes := []rune(pattern), []rune(value)
	previous := make([]bool, len(valueRunes)+1)
	previous[0] = true
	for _, token := range patternRunes {
		next := make([]bool, len(valueRunes)+1)
		if token == '*' {
			next[0] = previous[0]
			for index := 1; index <= len(valueRunes); index++ {
				next[index] = previous[index] || next[index-1]
			}
		} else {
			for index := 1; index <= len(valueRunes); index++ {
				next[index] = previous[index-1] && (token == '?' || token == valueRunes[index-1])
			}
		}
		previous = next
	}
	return previous[len(valueRunes)]
}
