package vec

import "testing"

func fromSlice(xs []int) Vec[int] {
	v := Vec[int](Nil[int]{})
	for i := len(xs) - 1; i >= 0; i-- {
		v = Cons[int]{Head: xs[i], Tail: v}
	}
	return v
}

func toSlice(v Vec[int]) []int {
	var out []int
	for {
		c, ok := any(v).(Cons[int])
		if !ok {
			return out
		}
		out = append(out, c.Head)
		v = c.Tail
	}
}

func TestFirstRest(t *testing.T) {
	v := fromSlice([]int{1, 2, 3})
	if First(v) != 1 {
		t.Fatalf("First = %d", First(v))
	}
	if got := toSlice(Rest(v)); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("Rest = %v", got)
	}
}

func TestConcatLength(t *testing.T) {
	for n := 0; n < 5; n++ {
		for m := 0; m < 5; m++ {
			a, b := toVecN(n), toVecN(m)
			if got := Length(Concat(a, b)); got != n+m {
				t.Fatalf("Length(Concat(%d, %d)) = %d", n, m, got)
			}
		}
	}
}

func toVecN(n int) Vec[int] {
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i
	}
	return fromSlice(xs)
}

func TestReplicateLengthAgree(t *testing.T) {
	for n := 0; n < 6; n++ {
		if got := Length(Replicate(n, "x")); got != n {
			t.Fatalf("Length(Replicate(%d)) = %d", n, got)
		}
	}
}

func TestMapPreservesLength(t *testing.T) {
	v := fromSlice([]int{1, 2, 3})
	w := Map(func(x int) int { return x * 10 }, v)
	if got := toSlice(w); len(got) != 3 || got[0] != 10 {
		t.Fatalf("Map = %v", got)
	}
}

func TestZipAndBoundsEvidence(t *testing.T) {
	left := fromSlice([]int{1, 2, 3})
	right := fromSlice([]int{4, 5, 6})
	zipped := toPairSlice(Zip(left, right))
	if len(zipped) != 3 || zipped[1] != (Pair[int, int]{First: 2, Second: 5}) {
		t.Fatalf("Zip = %#v", zipped)
	}
	index := Fin(Succ{Prev: Zero{}})
	if got := At(index, left); got != 2 {
		t.Fatalf("At(1) = %d", got)
	}
}

func toPairSlice(v Vec[Pair[int, int]]) []Pair[int, int] {
	var out []Pair[int, int]
	for {
		c, ok := any(v).(Cons[Pair[int, int]])
		if !ok {
			return out
		}
		out = append(out, c.Head)
		v = c.Tail
	}
}

func TestGuardFiresFromPlainGo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("First(Nil) did not panic")
		}
	}()
	First[int](Nil[int]{})
}

func TestZipGuardFiresFromPlainGo(t *testing.T) {
	for _, pair := range [][2]Vec[int]{
		{fromSlice([]int{1}), fromSlice([]int{2, 3})},
		{fromSlice([]int{1, 2}), fromSlice([]int{3})},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("Zip accepted different runtime lengths")
				}
			}()
			_ = Zip(pair[0], pair[1])
		}()
	}
}

// AtIndex is the plain-number counterpart to At: the bound is proved at
// the call site and erased, so the generated function is an ordinary
// walk with no evidence to carry.
func TestAtIndex(t *testing.T) {
	v := fromSlice([]int{10, 20, 30})
	for i, want := range []int{10, 20, 30} {
		if got := AtIndex(i, v); got != want {
			t.Errorf("AtIndex(%d) = %d, want %d", i, got, want)
		}
	}
}

// Set is AtIndex's dual and preserves length, so its index is unchanged.
func TestSet(t *testing.T) {
	v := fromSlice([]int{10, 20, 30})
	for i, want := range [][]int{{99, 20, 30}, {10, 99, 30}, {10, 20, 99}} {
		got := toSlice(Set(i, 99, v))
		if len(got) != len(want) {
			t.Fatalf("Set(%d) changed the length: %v", i, got)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Errorf("Set(%d) = %v, want %v", i, got, want)
				break
			}
		}
	}
	// The original is untouched: Vec is a persistent cons list.
	if got := toSlice(v); got[0] != 10 {
		t.Errorf("Set mutated its input: %v", got)
	}
}

// A plain-Go caller has no proof, so the erased boundary is guarded
// rather than left to compute garbage.
func TestAtIndexGuardsPlainGoCallers(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("indexing an empty vector must panic at the boundary")
		}
	}()
	AtIndex(0, Vec[int](Nil[int]{}))
}
