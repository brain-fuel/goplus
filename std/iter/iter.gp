// Package iter is Go+'s lazy sequence algebra: a Seq[T] wraps the standard
// library's stditer.Seq[T] and carries fluent, allocation-light combinators.
// The wrapper never hides the standard iterator — Seq.Seq() erases back to
// stditer.Seq[T] at any boundary — so Seq composes with range-over-func and every
// other stdlib iterator. Fallible pipelines carry std/result.Result elements;
// parallel evaluation lives in the sibling std/iter/parallel package so this
// core stays deterministic and pure.
//
// A Seq is single-pass unless its backing iter.Seq is itself replayable.
package iter

import (
	stditer "iter"

	"goforge.dev/goplus/std/option"
	"goforge.dev/goplus/std/result"
)

// Seq is a lazy sequence of T built over the standard stditer.Seq[T].
type Seq[T any] struct {
	seq stditer.Seq[T]
}

// Indexed pairs a value with its 0-based position in a sequence. It is what
// Enumerate yields, so the index is available to a following Map / Filter /
// ForEach without threading a counter through every callback.
type Indexed[T any] struct {
	Index int
	Value T
}

// Of wraps a standard iterator as a Seq.
func Of[T any](seq stditer.Seq[T]) Seq[T] {
	return Seq[T]{seq: seq}
}

// FromSlice yields the elements of s in order.
func FromSlice[T any](s []T) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		for i := range s {
			if !yield(s[i]) {
				return
			}
		}
	}}
}

// Enumerate pairs each element with its 0-based index (as in Rust's enumerate or
// Python's enumerate), so a following Map / Filter / ForEach can use the position
// without an ad-hoc index parameter on every callback:
//
//	iter.Enumerate(s).ForEach(func(it Indexed[T]) { use(it.Index, it.Value) })
//
// It is a free function, not a method, because a method on Seq[T] returning
// Seq[Indexed[T]] is an illegal instantiation cycle in Go's type system — the same
// constraint that pushes lo-style libraries to bake an int into every callback.
func Enumerate[T any](s Seq[T]) Seq[Indexed[T]] {
	return Seq[Indexed[T]]{seq: func(yield func(Indexed[T]) bool) {
		i := 0
		for v := range s.seq {
			if !yield(Indexed[T]{Index: i, Value: v}) {
				return
			}
			i++
		}
	}}
}

// Range yields 0,1,...,n-1 (empty when n <= 0).
func Range(n int) Seq[int] {
	return Seq[int]{seq: func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}}
}

// Seq erases the wrapper, returning the underlying standard iterator so a Seq
// can be ranged over or handed to any iter.Seq consumer.
func (s Seq[T]) Seq() stditer.Seq[T] {
	return s.seq
}

// Map lazily applies f to every element.
func (s Seq[T]) Map[U any](f func(T) U) Seq[U] {
	return Seq[U]{seq: func(yield func(U) bool) {
		for v := range s.seq {
			if !yield(f(v)) {
				return
			}
		}
	}}
}

// FilterMap lazily maps and keeps in one pass: f returns a value and whether
// to keep it.
func (s Seq[T]) FilterMap[U any](f func(T) (U, bool)) Seq[U] {
	return Seq[U]{seq: func(yield func(U) bool) {
		for v := range s.seq {
			if u, ok := f(v); ok && !yield(u) {
				return
			}
		}
	}}
}

// FlatMap lazily maps each element to a sub-sequence and concatenates them.
func (s Seq[T]) FlatMap[U any](f func(T) Seq[U]) Seq[U] {
	return Seq[U]{seq: func(yield func(U) bool) {
		for v := range s.seq {
			for u := range f(v).seq {
				if !yield(u) {
					return
				}
			}
		}
	}}
}

// Filter lazily keeps the elements satisfying keep.
func (s Seq[T]) Filter(keep func(T) bool) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		for v := range s.seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}}
}

// Take yields at most the first n elements.
func (s Seq[T]) Take(n int) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		count := 0
		for v := range s.seq {
			if !yield(v) {
				return
			}
			count++
			if count >= n {
				return
			}
		}
	}}
}

// Drop skips the first n elements.
func (s Seq[T]) Drop(n int) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		skipped := 0
		for v := range s.seq {
			if skipped < n {
				skipped++
				continue
			}
			if !yield(v) {
				return
			}
		}
	}}
}

// TakeWhile yields leading elements while pred holds, stopping at the first
// failure.
func (s Seq[T]) TakeWhile(pred func(T) bool) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		for v := range s.seq {
			if !pred(v) {
				return
			}
			if !yield(v) {
				return
			}
		}
	}}
}

// DropWhile skips leading elements while pred holds, then yields the rest.
func (s Seq[T]) DropWhile(pred func(T) bool) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		dropping := true
		for v := range s.seq {
			if dropping && pred(v) {
				continue
			}
			dropping = false
			if !yield(v) {
				return
			}
		}
	}}
}

// Chunk groups the sequence into slices of at most size elements. It panics on
// a non-positive size. It is a free function because a method returning
// Seq[[]T] would form a generic instantiation cycle.
func Chunk[T any](s Seq[T], size int) Seq[[]T] {
	if size <= 0 {
		panic("iter.Chunk: size must be greater than 0")
	}
	return Seq[[]T]{seq: func(yield func([]T) bool) {
		batch := make([]T, 0, size)
		for v := range s.seq {
			batch = append(batch, v)
			if len(batch) == size {
				if !yield(batch) {
					return
				}
				batch = make([]T, 0, size)
			}
		}
		if len(batch) > 0 {
			yield(batch)
		}
	}}
}

// Contains reports whether the sequence yields want.
func Contains[T comparable](s Seq[T], want T) bool {
	for v := range s.seq {
		if v == want {
			return true
		}
	}
	return false
}

// Uniq yields the first occurrence of each distinct element, preserving order.
func Uniq[T comparable](s Seq[T]) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		seen := make(map[T]struct{})
		for v := range s.seq {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}}
}

// UniqBy yields the first element for each distinct key, preserving order.
func UniqBy[T any, K comparable](s Seq[T], key func(T) K) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		seen := make(map[K]struct{})
		for v := range s.seq {
			k := key(v)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}}
}

// Reverse materializes the sequence and yields it back-to-front (necessarily
// eager, since the tail must be reached first).
func (s Seq[T]) Reverse() Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		buf := s.Collect()
		for i := len(buf) - 1; i >= 0; i-- {
			if !yield(buf[i]) {
				return
			}
		}
	}}
}

// Concat yields every element of each sequence in turn.
func Concat[T any](seqs ...Seq[T]) Seq[T] {
	return Seq[T]{seq: func(yield func(T) bool) {
		for _, s := range seqs {
			for v := range s.seq {
				if !yield(v) {
					return
				}
			}
		}
	}}
}

// ForEach consumes the sequence, applying f to each element.
func (s Seq[T]) ForEach(f func(T)) {
	for v := range s.seq {
		f(v)
	}
}

// Fold left-folds the sequence into an accumulator.
func (s Seq[T]) Fold[A any](initial A, step func(A, T) A) A {
	acc := initial
	for v := range s.seq {
		acc = step(acc, v)
	}
	return acc
}

// Collect materializes the sequence into a slice, preserving order.
func (s Seq[T]) Collect() []T {
	out := make([]T, 0)
	for v := range s.seq {
		out = append(out, v)
	}
	return out
}

// Count consumes the sequence and returns its length.
func (s Seq[T]) Count() int {
	n := 0
	for range s.seq {
		n++
	}
	return n
}

// First returns the first element as an Option, consuming no more than one.
func (s Seq[T]) First() option.Option[T] {
	for v := range s.seq {
		return option.Some[T](v)
	}
	return option.None[T]()
}

// TryFold left-folds with a fallible step, short-circuiting on the first Err.
// The result carries the accumulator on success or the failing error.
func (s Seq[T]) TryFold[A any](initial A, step func(A, T) result.Result[A, error]) result.Result[A, error] {
	acc := initial
	var failure result.Result[A, error]
	failed := false
	for v := range s.seq {
		match step(acc, v) {
		case Ok(next):
			acc = next
		case Err(e):
			failure = result.Err[A, error](e)
			failed = true
		}
		if failed {
			break
		}
	}
	if failed {
		return failure
	}
	return result.Ok[A, error](acc)
}

// CollectResults materializes a sequence of Result elements, short-circuiting
// on the first Err and otherwise returning the Ok values in order.
func CollectResults[T any](s Seq[result.Result[T, error]]) result.Result[[]T, error] {
	out := make([]T, 0)
	var failure result.Result[[]T, error]
	failed := false
	for r := range s.seq {
		match r {
		case Ok(v):
			out = append(out, v)
		case Err(e):
			failure = result.Err[[]T, error](e)
			failed = true
		}
		if failed {
			break
		}
	}
	if failed {
		return failure
	}
	return result.Ok[[]T, error](out)
}
