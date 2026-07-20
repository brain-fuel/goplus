package directive

import "testing"

func TestRefinementMarkerRoundTrip(t *testing.T) {
	want := RefinementMarker{Name: "Port", Binder: "value", Base: "int", Predicate: "value > 0 && value < 1<<16"}
	got, ok := ParseRefinementMarker(want.String())
	if !ok || got != want {
		t.Fatalf("round trip = %#v, %v; want %#v, true", got, ok, want)
	}
}

func TestRefinementMarkerRejectsDamage(t *testing.T) {
	for _, line := range []string{"//goplus:refinement", "//goplus:refinement Port value int pred", `//goplus:refinement "P" "x" "int"`} {
		if _, ok := ParseRefinementMarker(line); ok {
			t.Fatalf("accepted damaged marker %q", line)
		}
	}
}
