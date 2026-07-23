package smt

import "testing"

func TestStringGroundOperations(t *testing.T) {
	value := StringConcat(StringVal("Go"), StringVal("+"), StringVal("🙂"))
	formula := And{Values: []Term[BoolSort]{
		Equal{Left: value, Right: StringVal("Go+🙂")},
		Equal{Left: StringLength(value), Right: Integer{Value: 4}},
		StringContains(value, StringVal("+")),
		StringHasPrefix(value, StringVal("Go")),
		StringHasSuffix(value, StringVal("🙂")),
	}}
	checked := Check(Assert(1, New(), formula))
	result, ok := checked.(Satisfiable)
	if !ok {
		t.Fatalf("result=%T (%#v)", checked, checked)
	}
	if actual, found := StringModelValue(result.Value, value); !found || actual != "Go+🙂" {
		t.Fatalf("value=(%q,%v)", actual, found)
	}
}

func TestStringSymbolModel(t *testing.T) {
	x := StringConst(1, "x")
	formula := And{Values: []Term[BoolSort]{
		Equal{Left: x, Right: StringVal("goforge")},
		StringHasPrefix(x, StringVal("go")),
		StringHasSuffix(x, StringVal("forge")),
	}}
	checked := Check(Assert(2, New(), formula))
	result, ok := checked.(Satisfiable)
	if !ok {
		t.Fatalf("result=%T (%#v)", checked, checked)
	}
	if actual, found := StringModelValue(result.Value, x); !found || actual != "goforge" {
		t.Fatalf("x=(%q,%v)", actual, found)
	}
}

func TestStringDisequalitySynthesizesDistinctValues(t *testing.T) {
	x := StringConst(1, "x")
	y := StringConst(2, "y")
	checked := Check(Assert(3, New(), Not{Value: Equal{Left: x, Right: y}}))
	result, ok := checked.(Satisfiable)
	if !ok {
		t.Fatalf("result=%T (%#v)", checked, checked)
	}
	left, leftOK := StringModelValue(result.Value, x)
	right, rightOK := StringModelValue(result.Value, y)
	if !leftOK || !rightOK || left == right {
		t.Fatalf("x=(%q,%v), y=(%q,%v)", left, leftOK, right, rightOK)
	}
}

func TestStringGroundContradiction(t *testing.T) {
	formula := StringHasPrefix(StringVal("forge"), StringVal("go"))
	if _, ok := Check(Assert(4, New(), formula)).(Unsatisfiable); !ok {
		t.Fatal("expected ground string contradiction to be unsatisfiable")
	}
}

func TestStringAssumptionModel(t *testing.T) {
	x := StringConst(1, "x")
	checked := CheckAssuming(New(), Equal{Left: x, Right: StringVal("assumed")})
	result, ok := checked.(AssumptionsSatisfiable)
	if !ok {
		t.Fatalf("result=%T (%#v)", checked, checked)
	}
	if actual, found := StringModelValue(result.Value, x); !found || actual != "assumed" {
		t.Fatalf("x=(%q,%v)", actual, found)
	}
}

func BenchmarkStringQFSLIA(b *testing.B) {
	x := StringConst(1, "x")
	formula := And{Values: []Term[BoolSort]{
		Equal{Left: x, Right: StringConcat(StringVal("go-"), StringVal("forge"))},
		Equal{Left: StringLength(x), Right: Integer{Value: 8}},
		StringContains(x, StringVal("-")),
		StringHasPrefix(x, StringVal("go")),
		StringHasSuffix(x, StringVal("forge")),
	}}
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if _, ok := Check(Assert(index+1, New(), formula)).(Satisfiable); !ok {
			b.Fatal("expected satisfiable string workload")
		}
	}
}
