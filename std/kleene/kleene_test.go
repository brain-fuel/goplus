package kleene

import (
	"iter"
	"slices"
	"testing"
)

var all3 = []Value{True{}, False{}, Undetermined{}}

func name(v Value) string {
	switch v.(type) {
	case True:
		return "T"
	case False:
		return "F"
	default:
		return "U"
	}
}

func seq(vs ...Value) iter.Seq[Value] { return slices.Values(vs) }

func TestFromBoolDef(t *testing.T) {
	if FromBool(true) != Value(True{}) || FromBool(false) != Value(False{}) {
		t.Fatal("FromBool")
	}
	for _, tc := range []struct {
		v         Value
		truth, ok bool
	}{{True{}, true, true}, {False{}, false, true}, {Undetermined{}, false, false}} {
		truth, ok := Def(tc.v)
		if truth != tc.truth || ok != tc.ok {
			t.Fatalf("Def(%s) = %v,%v", name(tc.v), truth, ok)
		}
	}
}

func TestResolve(t *testing.T) {
	if !Resolve(True{}, false) || Resolve(False{}, true) {
		t.Fatal("Resolve determined arms must ignore the default")
	}
	if !Resolve(Undetermined{}, true) || Resolve(Undetermined{}, false) {
		t.Fatal("Resolve(Undetermined) must be the default")
	}
}

func TestNotTable(t *testing.T) {
	want := map[string]Value{"T": False{}, "F": True{}, "U": Undetermined{}}
	for _, v := range all3 {
		if got := Not(v); got != want[name(v)] {
			t.Fatalf("Not(%s) = %s", name(v), name(got))
		}
	}
}

// The K3 truth tables, exhaustively.
func TestAndOrTables(t *testing.T) {
	and := map[[2]string]string{
		{"T", "T"}: "T", {"T", "F"}: "F", {"T", "U"}: "U",
		{"F", "T"}: "F", {"F", "F"}: "F", {"F", "U"}: "F",
		{"U", "T"}: "U", {"U", "F"}: "F", {"U", "U"}: "U",
	}
	or := map[[2]string]string{
		{"T", "T"}: "T", {"T", "F"}: "T", {"T", "U"}: "T",
		{"F", "T"}: "T", {"F", "F"}: "F", {"F", "U"}: "U",
		{"U", "T"}: "T", {"U", "F"}: "U", {"U", "U"}: "U",
	}
	for _, a := range all3 {
		for _, b := range all3 {
			if got := And(a, b); name(got) != and[[2]string{name(a), name(b)}] {
				t.Fatalf("And(%s,%s) = %s", name(a), name(b), name(got))
			}
			if got := Or(a, b); name(got) != or[[2]string{name(a), name(b)}] {
				t.Fatalf("Or(%s,%s) = %s", name(a), name(b), name(got))
			}
		}
	}
}

// De Morgan holds in K3: Not(And(a,b)) == Or(Not(a), Not(b)) and dually.
func TestDeMorgan(t *testing.T) {
	for _, a := range all3 {
		for _, b := range all3 {
			if Not(And(a, b)) != Or(Not(a), Not(b)) {
				t.Fatalf("de morgan and: %s %s", name(a), name(b))
			}
			if Not(Or(a, b)) != And(Not(a), Not(b)) {
				t.Fatalf("de morgan or: %s %s", name(a), name(b))
			}
		}
	}
}

func TestAllAny(t *testing.T) {
	if All(seq()) != Value(True{}) {
		t.Fatal("All identity")
	}
	if Any(seq()) != Value(False{}) {
		t.Fatal("Any identity")
	}
	if All(seq(True{}, Undetermined{}, True{})) != Value(Undetermined{}) {
		t.Fatal("All absorbs U")
	}
	if Any(seq(False{}, Undetermined{}, False{})) != Value(Undetermined{}) {
		t.Fatal("Any absorbs U")
	}
	// Short-circuit: the dominant value wins without consuming the rest.
	pulled := 0
	counting := func(yield func(Value) bool) {
		for _, v := range []Value{False{}, Undetermined{}, True{}} {
			pulled++
			if !yield(v) {
				return
			}
		}
	}
	if All(counting) != Value(False{}) || pulled != 1 {
		t.Fatalf("All short-circuit: pulled %d", pulled)
	}
	pulled = 0
	counting2 := func(yield func(Value) bool) {
		for _, v := range []Value{True{}, False{}, Undetermined{}} {
			pulled++
			if !yield(v) {
				return
			}
		}
	}
	if Any(counting2) != Value(True{}) || pulled != 1 {
		t.Fatalf("Any short-circuit: pulled %d", pulled)
	}
	// All/Any agree with the folds of their binary connectives.
	for _, a := range all3 {
		for _, b := range all3 {
			for _, c := range all3 {
				if All(seq(a, b, c)) != And(And(a, b), c) {
					t.Fatalf("All fold: %s %s %s", name(a), name(b), name(c))
				}
				if Any(seq(a, b, c)) != Or(Or(a, b), c) {
					t.Fatalf("Any fold: %s %s %s", name(a), name(b), name(c))
				}
			}
		}
	}
}
