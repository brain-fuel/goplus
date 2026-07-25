package iter_test

import (
	"errors"
	"reflect"
	"testing"
	"testing/quick"

	"goforge.dev/goplus/std/iter"
	"goforge.dev/goplus/std/result"
)

func check(t *testing.T, name string, f any) {
	t.Helper()
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Errorf("%s: %v", name, err)
	}
}

// Law: Collect(FromSlice(xs)) == xs (round-trip identity).
func TestRoundTrip(t *testing.T) {
	check(t, "roundtrip", func(xs []int) bool {
		got := iter.FromSlice(xs).Collect()
		if len(xs) == 0 {
			return len(got) == 0
		}
		return reflect.DeepEqual(got, xs)
	})
}

// Law: Map fuses with the eager slice map (same result, lazily).
func TestMapLaw(t *testing.T) {
	check(t, "map", func(xs []int) bool {
		lazy := iter.Map(iter.FromSlice(xs), func(x int) int { return x*3 + 1 }).Collect()
		eager := make([]int, len(xs))
		for i, x := range xs {
			eager[i] = x*3 + 1
		}
		return intsEqual(lazy, eager)
	})
}

// Law: Filter keeps exactly the predicate-satisfying elements, in order.
func TestFilterLaw(t *testing.T) {
	check(t, "filter", func(xs []int) bool {
		lazy := iter.FromSlice(xs).Filter(func(x int) bool { return x%2 == 0 }).Collect()
		eager := make([]int, 0)
		for _, x := range xs {
			if x%2 == 0 {
				eager = append(eager, x)
			}
		}
		return intsEqual(lazy, eager)
	})
}

// Law: Map∘Filter composition equals filter-then-map on the slice (fusion).
func TestMapFilterComposition(t *testing.T) {
	check(t, "map∘filter", func(xs []int) bool {
		lazy := iter.Map(
			iter.FromSlice(xs).Filter(func(x int) bool { return x > 0 }),
			func(x int) int { return x * x },
		).Collect()
		eager := make([]int, 0)
		for _, x := range xs {
			if x > 0 {
				eager = append(eager, x*x)
			}
		}
		return intsEqual(lazy, eager)
	})
}

// Law: Take(n)∘Drop(m) equals slice[m:m+n] (clamped).
func TestTakeDropLaw(t *testing.T) {
	check(t, "take/drop", func(xs []int, m, n uint8) bool {
		got := iter.FromSlice(xs).Drop(int(m)).Take(int(n)).Collect()
		start := int(m)
		if start > len(xs) {
			start = len(xs)
		}
		end := start + int(n)
		if end > len(xs) {
			end = len(xs)
		}
		want := append([]int{}, xs[start:end]...)
		return intsEqual(got, want)
	})
}

// Law: Fold sums identically to a plain loop; Count == len.
func TestFoldCountLaw(t *testing.T) {
	check(t, "fold/count", func(xs []int) bool {
		sum := iter.Fold(iter.FromSlice(xs), 0, func(a, x int) int { return a + x })
		total := 0
		for _, x := range xs {
			total += x
		}
		return sum == total && iter.FromSlice(xs).Count() == len(xs)
	})
}

// Law: TryFold short-circuits on the first Err; otherwise equals a total fold.
func TestTryFoldShortCircuit(t *testing.T) {
	boom := errors.New("boom")
	check(t, "tryfold", func(xs []int) bool {
		step := func(a, x int) result.Result[int, error] {
			if x == 0 {
				return result.Err[int, error]{Err: boom}
			}
			return result.Ok[int, error]{Value: a + x}
		}
		got := iter.TryFold(iter.FromSlice(xs), 0, step)
		gotVal, gotErr := result.Unpack(got)

		wantSum, wantErr := 0, error(nil)
		for _, x := range xs {
			if x == 0 {
				wantErr = boom
				break
			}
			wantSum += x
		}
		if wantErr != nil {
			return gotErr != nil
		}
		return gotErr == nil && gotVal == wantSum
	})
}

// Law: CollectResults returns all Ok values, or the first Err.
func TestCollectResults(t *testing.T) {
	boom := errors.New("boom")
	check(t, "collectresults", func(xs []int) bool {
		rs := iter.Map(iter.FromSlice(xs), func(x int) result.Result[int, error] {
			if x < 0 {
				return result.Err[int, error]{Err: boom}
			}
			return result.Ok[int, error]{Value: x}
		})
		got := iter.CollectResults(rs)
		gotVals, gotErr := result.Unpack(got)

		hasNeg := false
		for _, x := range xs {
			if x < 0 {
				hasNeg = true
				break
			}
		}
		if hasNeg {
			return gotErr != nil
		}
		return gotErr == nil && (len(xs) == 0 && len(gotVals) == 0 || reflect.DeepEqual(gotVals, xs))
	})
}

func intsEqual(a, b []int) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

// --- laws for the expanded combinators ---

func TestUniqLaw(t *testing.T) {
	check(t, "uniq", func(xs []int) bool {
		got := iter.Uniq(iter.FromSlice(xs)).Collect()
		seen := map[int]bool{}
		want := []int{}
		for _, x := range xs {
			if !seen[x] {
				seen[x] = true
				want = append(want, x)
			}
		}
		return intsEqual(got, want)
	})
}

func TestFilterMapFlatMapLaw(t *testing.T) {
	check(t, "filtermap", func(xs []int) bool {
		got := iter.FilterMap(iter.FromSlice(xs), func(x int) (int, bool) { return x * 2, x%2 == 0 }).Collect()
		want := []int{}
		for _, x := range xs {
			if x%2 == 0 {
				want = append(want, x*2)
			}
		}
		return intsEqual(got, want)
	})
	check(t, "flatmap", func(xs []int) bool {
		got := iter.FlatMap(iter.FromSlice(xs), func(x int) iter.Seq[int] { return iter.FromSlice([]int{x, x}) }).Collect()
		want := []int{}
		for _, x := range xs {
			want = append(want, x, x)
		}
		return intsEqual(got, want)
	})
}

func TestTakeDropWhileReverseLaw(t *testing.T) {
	check(t, "takewhile/dropwhile partition", func(xs []int) bool {
		p := func(x int) bool { return x > 0 }
		head := iter.FromSlice(xs).TakeWhile(p).Collect()
		tail := iter.FromSlice(xs).DropWhile(p).Collect()
		return intsEqual(append(append([]int{}, head...), tail...), xs)
	})
	check(t, "reverse involution", func(xs []int) bool {
		got := iter.FromSlice(xs).Reverse().Reverse().Collect()
		return intsEqual(got, xs)
	})
	check(t, "contains", func(xs []int, v int) bool {
		want := false
		for _, x := range xs {
			if x == v {
				want = true
			}
		}
		return iter.Contains(iter.FromSlice(xs), v) == want
	})
}

func TestChunkConcatLaw(t *testing.T) {
	check(t, "chunk flattens back", func(xs []int, sz uint8) bool {
		size := int(sz)%5 + 1
		flat := []int{}
		for c := range iter.Chunk(iter.FromSlice(xs), size).Seq() {
			if len(c) > size {
				return false
			}
			flat = append(flat, c...)
		}
		return intsEqual(flat, xs)
	})
	check(t, "concat", func(a, b []int) bool {
		got := iter.Concat(iter.FromSlice(a), iter.FromSlice(b)).Collect()
		return intsEqual(got, append(append([]int{}, a...), b...))
	})
}
