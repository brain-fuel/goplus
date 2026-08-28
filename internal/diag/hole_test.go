package diag

import "testing"

func TestHoleInfoString(t *testing.T) {
	for _, tc := range []struct {
		name string
		info HoleInfo
		want string
	}{
		{
			name: "erased goal only",
			info: HoleInfo{Name: "choice", Type: "string"},
			want: "hole ?choice : string",
		},
		{
			name: "dependent goal reports both spellings",
			info: HoleInfo{Name: "rest", Type: "Vec[T]", DepType: "Vec[T, n]"},
			want: "hole ?rest : Vec[T, n]\n  erased: Vec[T]",
		},
		{
			name: "matching spellings print once",
			info: HoleInfo{Name: "n", Type: "int", DepType: "int"},
			want: "hole ?n : int",
		},
		{
			name: "no expectation names the fix",
			info: HoleInfo{Name: "mystery"},
			want: "hole ?mystery : cannot infer a type from this context\n" +
				"  hint: annotate the binding this hole initializes; an inferred binding and other untyped positions carry no expectation",
		},
		{
			name: "bindings prefer the un-erased spelling and mark erased ones",
			info: HoleInfo{
				Name: "rest", DepType: "Vec[T, n]",
				Bindings: []HoleBinding{
					{Name: "n", DepType: "nat", Erased: true},
					{Name: "v", Type: "Vec[T]", DepType: "Vec[T, n+1]"},
					{Name: "label", Type: "string"},
				},
			},
			want: "hole ?rest : Vec[T, n]\n" +
				"  in scope:\n" +
				"    n : nat (erased, quantity 0)\n" +
				"    v : Vec[T, n+1]\n" +
				"    label : string",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.info.String(); got != tc.want {
				t.Errorf("HoleInfo.String():\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
