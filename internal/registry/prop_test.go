package registry

import "testing"

// The declaration erases completely, so its marker is what a consumer
// receives — it has to survive the round trip exactly.
func TestPropMarkerRoundTrip(t *testing.T) {
	want := &PropDef{
		PkgPath: "example.com/lib", Name: "InRange",
		Params: []string{"i", "n"}, Body: "And[Le[0, i], Lt[i, n]]",
	}
	line := want.Marker()
	if line != "//goplus:prop InRange[i, n] And[Le[0, i], Lt[i, n]]" {
		t.Fatalf("Marker() = %q", line)
	}
	got, err := ParsePropMarker(want.PkgPath, line[len(PropPrefix)+1:])
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name || got.Body != want.Body || len(got.Params) != 2 {
		t.Errorf("round trip: got %+v, want %+v", *got, *want)
	}
}

func TestParsePropMarkerRejectsMalformed(t *testing.T) {
	for _, body := range []string{"", "InRange", "InRange[]", "InRange[i, n]", "[i] Body", "InRange[i] "} {
		if _, err := ParsePropMarker("p", body); err == nil {
			t.Errorf("ParsePropMarker(%q) accepted a malformed marker", body)
		}
	}
}

// A named proposition erases to a free-floating comment, not the doc of
// any declaration, so the scanner must read comments rather than decls.
func TestPropDefsFromFreeFloatingMarker(t *testing.T) {
	src := []byte("package lib\n\n//goplus:prop InRange[i, n] And[Le[0, i], Lt[i, n]]\n\nfunc F(x int) int { return x }\n")
	got, err := PropDefsFromMarkers("example.com/lib", "l_gp.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "InRange" || got[0].Body != "And[Le[0, i], Lt[i, n]]" {
		t.Fatalf("got %+v", got)
	}
	if as, err := PropDefsFromMarkers("p", "x.go", []byte("package p\n")); err != nil || as != nil {
		t.Errorf("marker-free file: (%v, %v)", as, err)
	}
}

func TestRegistryPropDefs(t *testing.T) {
	r := New()
	if err := r.AddPropDef(&PropDef{PkgPath: "p", Name: "Pos", Params: []string{"n"}, Body: "Lt[0, n]"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.LookupPropDef("p", "Pos"); !ok {
		t.Error("a registered proposition must be findable")
	}
	if _, ok := r.LookupPropDef("other", "Pos"); ok {
		t.Error("propositions are package-scoped")
	}
	// A conflicting redeclaration is an error, not a silent overwrite.
	if err := r.AddPropDef(&PropDef{PkgPath: "p", Name: "Pos", Params: []string{"n"}, Body: "Le[1, n]"}); err == nil {
		t.Error("a conflicting redeclaration must be refused")
	}
	if tbl := r.PropDefs(); tbl["Pos"] != [2]string{"n", "Lt[0, n]"} {
		t.Errorf("PropDefs table = %+v", tbl)
	}
}
