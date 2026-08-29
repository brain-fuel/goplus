package registry

import "testing"

// A marker is what carries an assumption to a consumer, so it has to
// survive the round trip exactly — including a proposition with spaces.
func TestAssumeMarkerRoundTrip(t *testing.T) {
	for _, want := range []*Assumption{
		{PkgPath: "example.com/lib", Fn: "Widen", Callee: "Cast", Param: "p", Proposition: "2 = 3"},
		{PkgPath: "example.com/lib", Fn: "F", Callee: "G", Param: "q", Proposition: "n * 2 = 2 * n"},
	} {
		line := want.Marker()
		body, ok := cutMarker(line)
		if !ok {
			t.Fatalf("marker %q lacks its prefix", line)
		}
		got, err := ParseAssumeMarker(want.PkgPath, body)
		if err != nil {
			t.Fatalf("ParseAssumeMarker(%q): %v", body, err)
		}
		if *got != *want {
			t.Errorf("round trip: got %+v, want %+v", *got, *want)
		}
	}
}

func cutMarker(line string) (string, bool) {
	if len(line) <= len(AssumePrefix)+1 || line[:len(AssumePrefix)] != AssumePrefix {
		return "", false
	}
	return line[len(AssumePrefix)+1:], true
}

func TestParseAssumeMarkerRejectsMalformed(t *testing.T) {
	for _, body := range []string{"", "Widen", "Widen Cast", "Widen Cast p", "Widen Cast p   "} {
		if _, err := ParseAssumeMarker("p", body); err == nil {
			t.Errorf("ParseAssumeMarker(%q) accepted a malformed marker", body)
		}
	}
}

// Reading assumptions back out of a distributed file is the whole point:
// the .gp that recorded them is not shipped.
func TestAssumptionsFromMarkers(t *testing.T) {
	src := []byte(`package lib

//goplus:assume Widen Cast p 2 = 3
func Widen(v int) int { return v }

func Plain(v int) int { return v }
`)
	got, err := AssumptionsFromMarkers("example.com/lib", "lib_gp.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d assumptions, want 1: %+v", len(got), got)
	}
	want := Assumption{PkgPath: "example.com/lib", Fn: "Widen", Callee: "Cast", Param: "p", Proposition: "2 = 3"}
	if *got[0] != want {
		t.Errorf("got %+v, want %+v", *got[0], want)
	}
	// A file with no markers costs nothing and yields nothing.
	if as, err := AssumptionsFromMarkers("p", "x.go", []byte("package p\n")); err != nil || as != nil {
		t.Errorf("marker-free file: got (%v, %v)", as, err)
	}
}

// Registered assumptions dedupe exactly and order deterministically.
func TestRegistryAssumptions(t *testing.T) {
	r := New()
	a := &Assumption{PkgPath: "b", Fn: "F", Callee: "C", Param: "p", Proposition: "1 = 1"}
	r.AddAssumption(a)
	r.AddAssumption(&Assumption{PkgPath: "b", Fn: "F", Callee: "C", Param: "p", Proposition: "1 = 1"})
	r.AddAssumption(&Assumption{PkgPath: "a", Fn: "G", Callee: "C", Param: "q", Proposition: "2 = 2"})
	got := r.Assumptions()
	if len(got) != 2 {
		t.Fatalf("got %d, want 2 (an exact repeat must collapse): %+v", len(got), got)
	}
	if got[0].PkgPath != "a" || got[1].PkgPath != "b" {
		t.Errorf("order is not deterministic by package: %+v", got)
	}
}
