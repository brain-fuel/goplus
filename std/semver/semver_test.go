package semver

import "testing"

func TestParseCompareAndBump(t *testing.T) {
	v, err := Parse("1.2.3-rc.1+build.7")
	if err != nil || v.String() != "1.2.3-rc.1+build.7" {
		t.Fatalf("Parse = %v, %v", v, err)
	}
	if v.Compare(MustParse("1.2.3")) >= 0 {
		t.Fatal("prerelease must precede release")
	}
	if got := v.BumpMinor().String(); got != "1.3.0" {
		t.Fatalf("bump = %s", got)
	}
}

func TestRejectsInvalid(t *testing.T) {
	for _, s := range []string{"", "v1.2.3", "1.2", "01.2.3", "1.2.3-01", "1.2.3+"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) succeeded", s)
		}
	}
}
