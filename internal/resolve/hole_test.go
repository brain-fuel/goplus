package resolve

import "testing"

func TestHoleNameFromCarrier(t *testing.T) {

	for _, tc := range []struct {
		carrier string
		want    string
		ok      bool
	}{
		{"__gp_hole0_rest", "rest", true},
		{"__gp_hole12_someName", "someName", true},
		{"__gp_hole0_", "", false},
		{"__gp_try0", "", false},
		{"ordinaryCall", "", false},
	} {
		got, ok := holeCarrierName(tc.carrier)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%q: got (%q, %v), want (%q, %v)", tc.carrier, got, ok, tc.want, tc.ok)
		}
	}
}
